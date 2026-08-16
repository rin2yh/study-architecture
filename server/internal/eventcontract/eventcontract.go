// Package eventcontract は発行するイベントの wire スキーマを一箇所に集め、非互換な変更を CI で
// 弾く (ADR-[[202608160730]])。ADR-[[202607020343]] のフィットネス関数の枠に乗る 2 本目。
package eventcontract

import (
	"fmt"

	"github.com/rin2yh/study-architecture/server/internal/order"
	"github.com/rin2yh/study-architecture/server/internal/orderevent"
	"github.com/rin2yh/study-architecture/server/internal/outbox"
	"github.com/rin2yh/study-architecture/server/internal/paymentevent"
)

// Field は payload 1 フィールドの記録。Optional は Parse が欠落を許容する状態で、追加直後
// (全 producer が発行し始める前) と廃止予定の両方がここに入る。
type Field struct {
	Type     string `json:"type"`
	Optional bool   `json:"optional,omitempty"`
}

// Schema はフィールド名から記録への対応。Optional は記録側だけが持つ情報で、Values から導出した
// スキーマには載らない (発行される payload からは必須か任意かを読み取れないため)。
type Schema map[string]Field

// Contract は 1 イベント種の発行 (Values) と復元 (Parse) の対。Event は全フィールドが埋まった
// 有効なサンプルで、検査はここから 1 フィールドずつ欠いた payload を作って Parse に通す。
type Contract struct {
	Event outbox.Event
	Parse func(values map[string]any) error
}

const sampleOrderID = "1"

// Contracts は発行するイベント種の一覧。ここに足さないイベント種は互換検査を素通りする。
func Contracts() ([]Contract, error) {
	orderID, err := order.Parse(sampleOrderID)
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

// SchemaOf は Values の実出力からフィールド名と型を導出する。スキーマを別途手書きすると Values と
// 二重管理になり記録が嘘をつくため、導出元は実際に発行される payload だけにする。
func SchemaOf(ev outbox.Event) Schema {
	schema := Schema{}
	for name, v := range ev.Values() {
		schema[name] = Field{Type: fmt.Sprintf("%T", v)}
	}
	return schema
}

// CheckCompatible は previous に対する current の差分が後方互換かを検査する。記録とコードの突き合わせ
// にも、main の記録と PR の記録の突き合わせにも同じ規則を使う。
func CheckCompatible(previous, current Schema) []error {
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

// CheckSchemas は main の記録 previous に対する PR の記録 current が後方互換かを、イベント種ごとに
// 検査する。コードと記録の突き合わせだけでは、削除の前に optional へ落とす PR を挟んだかを判定できない。
func CheckSchemas(previous, current map[string]Schema) []error {
	var errs []error
	for eventType, was := range previous {
		now, ok := current[eventType]
		if !ok {
			errs = append(errs, fmt.Errorf("%s: 記録済みイベント種の削除は非互換。購読を外し切ってから消す", eventType))
			continue
		}
		for _, err := range CheckCompatible(was, now) {
			errs = append(errs, fmt.Errorf("%s: %w", eventType, err))
		}
	}
	return errs
}

// CheckRecorded は発行される current のフィールドがすべて recorded に載っているかを検査する。記録漏れ
// があると、そのフィールドは以降の削除・型変更の検査から外れてしまう。
func CheckRecorded(recorded, current Schema) []error {
	var errs []error
	for name := range current {
		if _, ok := recorded[name]; !ok {
			errs = append(errs, fmt.Errorf("%s: 記録に無いフィールドが発行されている。旧 producer は発行しないため optional として記録する", name))
		}
	}
	return errs
}

// CheckParse は記録した必須 / 任意が Parse の実挙動と一致するかを検査する。記録だけ optional にして
// Parse が追随していないと、廃止の 1 段階目を踏んだつもりで旧 payload を拒み続ける。
func CheckParse(c Contract, recorded Schema) []error {
	var errs []error
	full := c.Event.Values()
	// サンプルが Parse を通らないと、以降の「欠かすと失敗するか」がすべて真になり検査が空回りする。
	if err := c.Parse(full); err != nil {
		return []error{fmt.Errorf("サンプルが Parse を通らない: %w", err)}
	}
	for name, field := range recorded {
		if _, ok := full[name]; !ok {
			continue
		}
		err := c.Parse(withoutField(full, name))
		switch {
		case field.Optional && err != nil:
			errs = append(errs, fmt.Errorf("%s: optional と記録しているが Parse が欠落を拒んでいる: %v", name, err))
		case !field.Optional && err == nil:
			errs = append(errs, fmt.Errorf("%s: 必須と記録しているが Parse が欠落を受理している。欠落をゼロ値へ化けさせない", name))
		}
	}
	return errs
}

func withoutField(values map[string]any, drop string) map[string]any {
	rest := make(map[string]any, len(values))
	for name, v := range values {
		if name != drop {
			rest[name] = v
		}
	}
	return rest
}
