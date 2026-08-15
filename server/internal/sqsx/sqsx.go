// Package sqsx は messaging のポートを SNS + SQS で実装する (ADR-[[202608150830]])。
// ローカルと CI は同じ API を話す kumo に向ける。
package sqsx

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/rin2yh/study-architecture/server/internal/messaging"
	"github.com/rin2yh/study-architecture/server/internal/strconvx"
)

const (
	// ハンドラが内包する同期呼び出しの最悪ケース (resilience の AttemptTimeout 3s × MaxAttempts 3 +
	// バックオフ ≒ 10s) より十分長く取る。恒久的に失敗するメッセージの再試行間隔もこの値で決まる。
	visibilityTimeout = 30
	// DB 再起動程度の一過性障害では正常なメッセージを隔離しない値に取る (隔離までの猶予は
	// visibilityTimeout 倍で効く)。
	maxReceiveCount = 10
	// 空振りのポーリングでキューを叩き続けないよう、受信は long polling にする。
	waitTimeSeconds = 20
	receiveMax      = 10
)

type Client struct {
	sqs *sqs.Client
	sns *sns.Client
	// トピック ARN は起動後不変なので、Publish のたびに CreateTopic を往復しない。
	topicARN sync.Map
}

// AWS_ENDPOINT_URL が指定されていれば kumo などの互換実装 (SDK が自前で解決する)。実際の
// 資格情報は要らないが、SDK は署名のために何らかの資格情報を要求するのでダミーを入れる。
func NewClient(ctx context.Context) (*Client, error) {
	var opts []func(*config.LoadOptions) error
	if os.Getenv("AWS_ENDPOINT_URL") != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("kumo", "kumo", "")))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{sqs: sqs.NewFromConfig(cfg), sns: sns.NewFromConfig(cfg)}, nil
}

var (
	_ messaging.Publisher  = (*Client)(nil)
	_ messaging.Subscriber = (*Client)(nil)
)

func (c *Client) Publish(ctx context.Context, topic string, values map[string]any) error {
	arn, err := c.ensureTopic(ctx, topic)
	if err != nil {
		return err
	}
	body, err := json.Marshal(values)
	if err != nil {
		return err
	}
	_, err = c.sns.Publish(ctx, &sns.PublishInput{TopicArn: aws.String(arn), Message: aws.String(string(body))})
	return err
}

// Subscribe は queue と その DLQ を用意し、topic からの配信を繋ぐ。いずれも作成 API が冪等なので
// 起動のたびに呼んでよい。
func (c *Client) Subscribe(ctx context.Context, topic, queue string) (messaging.Subscription, error) {
	_, dlqARN, err := c.ensureQueue(ctx, queue+"-dlq", nil)
	if err != nil {
		return nil, err
	}
	redrive, err := json.Marshal(map[string]string{
		"deadLetterTargetArn": dlqARN,
		"maxReceiveCount":     strconvx.FormatInt64(maxReceiveCount),
	})
	if err != nil {
		return nil, err
	}
	url, queueARN, err := c.ensureQueue(ctx, queue, map[string]string{
		string(sqstypes.QueueAttributeNameRedrivePolicy):                 string(redrive),
		string(sqstypes.QueueAttributeNameVisibilityTimeout):             strconvx.FormatInt64(visibilityTimeout),
		string(sqstypes.QueueAttributeNameReceiveMessageWaitTimeSeconds): strconvx.FormatInt64(waitTimeSeconds),
	})
	if err != nil {
		return nil, err
	}
	topicARN, err := c.ensureTopic(ctx, topic)
	if err != nil {
		return nil, err
	}
	// SNS の封筒を剥がす処理を購読側に持たせないため raw で受け取る。
	if _, err := c.sns.Subscribe(ctx, &sns.SubscribeInput{
		TopicArn:   aws.String(topicARN),
		Protocol:   aws.String("sqs"),
		Endpoint:   aws.String(queueARN),
		Attributes: map[string]string{"RawMessageDelivery": "true"},
	}); err != nil {
		return nil, err
	}
	return &subscription{sqs: c.sqs, url: url}, nil
}

func (c *Client) ensureTopic(ctx context.Context, topic string) (string, error) {
	if arn, ok := c.topicARN.Load(topic); ok {
		return arn.(string), nil
	}
	out, err := c.sns.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String(topic)})
	if err != nil {
		return "", err
	}
	arn := aws.ToString(out.TopicArn)
	c.topicARN.Store(topic, arn)
	return arn, nil
}

func (c *Client) ensureQueue(ctx context.Context, queue string, attrs map[string]string) (url, arn string, err error) {
	created, err := c.sqs.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(queue), Attributes: attrs})
	if err != nil {
		return "", "", err
	}
	url = aws.ToString(created.QueueUrl)
	out, err := c.sqs.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(url),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return "", "", err
	}
	return url, out.Attributes[string(sqstypes.QueueAttributeNameQueueArn)], nil
}

type subscription struct {
	sqs *sqs.Client
	url string
}

func (s *subscription) Receive(ctx context.Context) ([]messaging.Received, error) {
	out, err := s.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(s.url),
		MaxNumberOfMessages: receiveMax,
		WaitTimeSeconds:     waitTimeSeconds,
	})
	if err != nil {
		return nil, err
	}
	received := make([]messaging.Received, 0, len(out.Messages))
	for _, m := range out.Messages {
		values, err := decode(aws.ToString(m.Body))
		if err != nil {
			// 壊れた payload は再配送しても直らないので、隔離をブローカに任せて ack しない。
			slog.ErrorContext(ctx, "undecodable message body", "queue", s.url, "error", err)
			continue
		}
		received = append(received, messaging.Received{Handle: aws.ToString(m.ReceiptHandle), Values: values})
	}
	return received, nil
}

func (s *subscription) Ack(ctx context.Context, handle string) error {
	_, err := s.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(s.url),
		ReceiptHandle: aws.String(handle),
	})
	return err
}

// 数値は json.Number のままだと扱いづらいので int64 へ寄せる (outbox の payload と同じ扱い)。
func decode(body string) (map[string]any, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	var values map[string]any
	if err := dec.Decode(&values); err != nil {
		return nil, err
	}
	for k, v := range values {
		n, ok := v.(json.Number)
		if !ok {
			continue
		}
		if i, err := n.Int64(); err == nil {
			values[k] = i
		} else {
			values[k] = n.String()
		}
	}
	return values, nil
}
