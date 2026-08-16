// Package eventcontract は発行するイベントの wire スキーマを一箇所に集め、非互換な変更を CI で
// 弾く (ADR-[[202608160730]])。ADR-[[202607020343]] のフィットネス関数の枠に乗る 2 本目。
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

// SchemaPath はリポジトリ root からの記録の位置。go test と CI のツールがこれ 1 つを見るので、
// 置き場所を動かしても検査が無言で空を比べ始めることがない。
const SchemaPath = "server/internal/eventcontract/testdata/schema.json"

// Field は payload 1 フィールドの記録。Optional は Parse が欠落を許容する状態で、追加直後
// (全 producer が発行し始める前) と廃止予定の両方がここに入る。
type Field struct {
	Type     string `json:"type"`
	Optional bool   `json:"optional,omitempty"`
}

// Schema はフィールド名から記録への対応。Optional は記録側だけが持つ情報で、発行される payload
// からは必須か任意かを読み取れない。
type Schema map[string]Field

// Contract は 1 イベント種の発行 (Values) と復元 (Parse) の対。Event は全フィールドが埋まった
// 有効なサンプルで、検査はここから 1 フィールドずつ欠いた payload を作って Parse に通す。
type Contract struct {
	Event outbox.Event
	Parse func(values map[string]any) error
}

// Contracts は発行するイベント種の一覧。ここに足さないイベント種は互換検査を素通りする。
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

// Check は記録 recorded とコードが一致しているかを検査する。記録とコードのフィールド集合は常に
// 一致させる (廃止するなら両方から消す)。ずれを許すと、記録だけ残したまま発行をやめる 1 PR が通り、
// 旧 consumer が必須として待っているフィールドが無言で消える。
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

// CheckSchemas は main の記録 previous に対する PR の記録 current が後方互換かを検査する。Check だけ
// では、削除の前に optional へ落とす PR を挟んだかを判定できない。
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
	// サンプルが Parse を通らないと、以降の「欠かすと失敗するか」がすべて真になり検査が空回りする。
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

// プロセス内の Go 型のまま記録すると、送出経路の往復で化ける型 (float64 の整数値は int64、非整数は
// string になる) を記録が見逃すため、実際に JSON 化して復元し直す。
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
