package eventcontract_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/eventcontract"
)

// SchemaPath はリポジトリ root 起点なので、パッケージ配下で回るテストからは root へ戻して開く。
var schemaPath = filepath.Join("..", "..", "..", eventcontract.SchemaPath)

// 発行中のイベントが記録済みスキーマと一致していることを CI で守るフィットネス関数
// (ADR-[[202608160730]] / ADR-[[202607020343]])。
func TestContractsMatchRecordedSchema(t *testing.T) {
	recorded, err := eventcontract.LoadSchemas(schemaPath)
	if err != nil {
		t.Fatalf("LoadSchemas: %v", err)
	}
	contracts, err := eventcontract.Contracts()
	if err != nil {
		t.Fatalf("Contracts(): %v", err)
	}

	implemented := map[string]bool{}
	for _, c := range contracts {
		eventType := c.Event.EventType()
		implemented[eventType] = true
		t.Run(eventType, func(t *testing.T) {
			schema, ok := recorded[eventType]
			if !ok {
				t.Fatalf("%s が %s に記録されていない。新しいイベント種は記録を足してから発行する", eventType, eventcontract.SchemaPath)
			}
			for _, err := range eventcontract.Check(c, schema) {
				t.Errorf("%s: %v", eventType, err)
			}
		})
	}

	for eventType := range recorded {
		if !implemented[eventType] {
			t.Errorf("%s が記録済みだが発行されていない。購読側が残っている可能性があるためイベント種の廃止は先に購読を外す", eventType)
		}
	}
}

// 記録と Parse のずれ自体が検査対象なので、実イベントでは前提を崩せない。
type stubEvent struct{}

func (stubEvent) EventType() string   { return "stub.event" }
func (stubEvent) AggregateID() string { return "1" }

func (stubEvent) Values() map[string]any {
	return map[string]any{"required": "a", "loose": "b"}
}

func parseStub(values map[string]any) error {
	if _, ok := values["required"].(string); !ok {
		return os.ErrInvalid
	}
	return nil
}

func TestCheck(t *testing.T) {
	type args struct {
		parse    func(values map[string]any) error
		recorded eventcontract.Schema
	}
	matching := eventcontract.Schema{"required": {Type: "string"}, "loose": {Type: "string", Optional: true}}
	tests := []struct {
		name     string
		args     args
		wantErrs int
	}{
		{"正常系 記録とコードが一致している", args{parseStub, matching}, 0},
		{
			"準正常系 発行しているフィールドが記録に無い",
			args{parseStub, eventcontract.Schema{"required": {Type: "string"}}},
			1,
		},
		{
			"準正常系 記録にあるフィールドを発行していない",
			args{parseStub, eventcontract.Schema{
				"required": {Type: "string"},
				"loose":    {Type: "string", Optional: true},
				"gone":     {Type: "string", Optional: true},
			}},
			1,
		},
		{
			"準正常系 記録と発行で型が違う",
			args{parseStub, eventcontract.Schema{"required": {Type: "int64"}, "loose": {Type: "string", Optional: true}}},
			1,
		},
		{
			"準正常系 optional と記録しているが Parse が欠落を拒む",
			args{parseStub, eventcontract.Schema{"required": {Type: "string", Optional: true}, "loose": {Type: "string", Optional: true}}},
			1,
		},
		{
			"準正常系 必須と記録しているが Parse が欠落を受理する",
			args{parseStub, eventcontract.Schema{"required": {Type: "string"}, "loose": {Type: "string"}}},
			1,
		},
		{
			"準正常系 サンプルが Parse を通らないと検査が空回りするので弾く",
			args{func(map[string]any) error { return os.ErrInvalid }, matching},
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract := eventcontract.Contract{Event: stubEvent{}, Parse: tt.args.parse}
			if got := eventcontract.Check(contract, tt.args.recorded); len(got) != tt.wantErrs {
				t.Fatalf("Check() = %v, want %d errors", got, tt.wantErrs)
			}
		})
	}
}

func TestCheckSchemas(t *testing.T) {
	type args struct {
		previous map[string]eventcontract.Schema
		current  map[string]eventcontract.Schema
	}
	settled := map[string]eventcontract.Schema{
		"payment.settled": {"orderId": {Type: "string"}, "amountCents": {Type: "int64"}},
	}
	optional := map[string]eventcontract.Schema{
		"payment.settled": {"orderId": {Type: "string"}, "amountCents": {Type: "int64", Optional: true}},
	}
	removed := map[string]eventcontract.Schema{"payment.settled": {"orderId": {Type: "string"}}}
	tests := []struct {
		name     string
		args     args
		wantErrs int
	}{
		{"正常系 変更が無ければ互換", args{settled, settled}, 0},
		{
			"正常系 イベント種の追加は互換",
			args{settled, map[string]eventcontract.Schema{
				"payment.settled": settled["payment.settled"],
				"payment.failed":  {"orderId": {Type: "string"}},
			}},
			0,
		},
		{"正常系 main で optional になっていれば削除できる", args{optional, removed}, 0},
		{"準正常系 optional を挟まない削除は 1 PR では通せない", args{settled, removed}, 1},
		{
			"準正常系 型変更は非互換",
			args{settled, map[string]eventcontract.Schema{
				"payment.settled": {"orderId": {Type: "string"}, "amountCents": {Type: "string"}},
			}},
			1,
		},
		{"準正常系 イベント種ごとの削除は非互換", args{settled, map[string]eventcontract.Schema{}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eventcontract.CheckSchemas(tt.args.previous, tt.args.current); len(got) != tt.wantErrs {
				t.Fatalf("CheckSchemas() = %v, want %d errors", got, tt.wantErrs)
			}
		})
	}
}
