// Package eventcontract は発行するイベントの wire スキーマを一箇所に集め、非互換な変更を CI で
// 弾く (ADR-[[202608160730]] / ADR-[[202607020343]])。
package eventcontract

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"

	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
)

// 参照を分散させると、検査が無言で空を比べ始める。
const SchemaPath = "server/internal/eventcontract/testdata/schema.json"

// Optional は「追加直後」と「廃止予定」を兼ねる (ADR-[[202608160730]])。
type Field struct {
	Type     string `json:"type"`
	Optional bool   `json:"optional,omitempty"`
}

type Schema map[string]Field

// Event には全フィールドが埋まった有効な値を渡す。
type Contract struct {
	Event outbox.Event
	Parse func(values map[string]any) error
}

// 登録漏れは検査を素通りする (ADR-[[202608160730]])。
func Contracts() ([]Contract, error) {
	orderID, err := order.Parse("1")
	if err != nil {
		return nil, err
	}
	return []Contract{
		{
			Event: paymentevent.Settled{PaymentID: 1, OrderID: orderID, AmountCents: 1},
			Parse: func(values map[string]any) error { _, err := paymentevent.ParseSettled(values); return err },
		},
		{
			Event: orderevent.Cancelled{OrderID: orderID},
			Parse: func(values map[string]any) error { _, err := orderevent.ParseCancelled(values); return err },
		},
	}, nil
}

func LoadSchemas(path string) (map[string]Schema, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSchemas(raw, path)
}

// CI は作業ツリーではなく git の blob から記録を読むため、ファイルを開く前段と分けてある。
func ParseSchemas(raw []byte, source string) (map[string]Schema, error) {
	var schemas map[string]Schema
	if err := json.Unmarshal(raw, &schemas); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	return schemas, nil
}

// 記録とコードのフィールド集合は常に一致させる (ADR-[[202608160730]])。
func Check(c Contract, recorded Schema) []error {
	values, err := wireValues(c.Event)
	if err != nil {
		return []error{err}
	}
	current := schemaOf(values)

	var errs []error
	for name, was := range recorded {
		now, ok := current[name]
		if !ok {
			errs = append(errs, fmt.Errorf("%s: 記録にあるが発行されていない。廃止するなら記録からも消す", name))
			continue
		}
		if now.Type != was.Type {
			errs = append(errs, fmt.Errorf("%s: 記録は %s だが発行は %s。型変更は非互換なので、別フィールドを optional で追加して移行する", name, was.Type, now.Type))
		}
	}
	for name := range current {
		if _, ok := recorded[name]; !ok {
			errs = append(errs, fmt.Errorf("%s: 発行されているが記録に無い。旧 producer は発行しないため optional として記録する", name))
		}
	}
	return append(errs, checkParse(c, recorded, values)...)
}

// previous は main の記録 (ADR-[[202608160730]])。
func CheckSchemas(previous, current map[string]Schema) []error {
	var errs []error
	for eventType, was := range previous {
		now, ok := current[eventType]
		if !ok {
			errs = append(errs, fmt.Errorf("%s: 記録済みイベント種の削除は非互換。購読を外し切ってから消す", eventType))
			continue
		}
		for _, err := range compatible(was, now) {
			errs = append(errs, fmt.Errorf("%s: %w", eventType, err))
		}
	}
	return errs
}

func compatible(previous, current Schema) []error {
	var errs []error
	for name, was := range previous {
		now, ok := current[name]
		if !ok {
			if !was.Optional {
				errs = append(errs, fmt.Errorf("%s: 記録済みフィールドの削除は非互換。先に optional へ落とす PR を main へ入れてから、別 PR で削除する", name))
			}
			continue
		}
		if now.Type != was.Type {
			errs = append(errs, fmt.Errorf("%s: 型が %s から %s へ変わっている。型変更は非互換なので、別フィールドを optional で追加して移行する", name, was.Type, now.Type))
		}
	}
	return errs
}

// 記録だけ optional にして Parse が追随していないと、廃止の 1 段階目を踏んだつもりで旧 payload を
// 拒み続ける。
func checkParse(c Contract, recorded Schema, full map[string]any) []error {
	// 続けると以降の欠落検査がすべて真になり空回りする。
	if err := c.Parse(full); err != nil {
		return []error{fmt.Errorf("サンプルが Parse を通らない: %w", err)}
	}
	var errs []error
	for name, field := range recorded {
		if _, ok := full[name]; !ok {
			continue
		}
		without := maps.Clone(full)
		delete(without, name)
		err := c.Parse(without)
		switch {
		case field.Optional && err != nil:
			errs = append(errs, fmt.Errorf("%s: optional と記録しているが Parse が欠落を拒んでいる: %v", name, err))
		case !field.Optional && err == nil:
			errs = append(errs, fmt.Errorf("%s: 必須と記録しているが Parse が欠落を受理している。欠落をゼロ値へ化けさせない", name))
		}
	}
	return errs
}

// (ADR-[[202608160730]])
func wireValues(ev outbox.Event) (map[string]any, error) {
	payload, err := json.Marshal(ev.Values())
	if err != nil {
		return nil, err
	}
	return outbox.DecodePayload(payload, "")
}

func schemaOf(values map[string]any) Schema {
	schema := Schema{}
	for name, v := range values {
		schema[name] = Field{Type: fmt.Sprintf("%T", v)}
	}
	return schema
}
