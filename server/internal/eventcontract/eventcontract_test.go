package eventcontract_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/rin2yh/study-architecture/server/internal/eventcontract"
)

const schemaPath = "testdata/schema.json"

func recordedSchemas(t *testing.T) map[string]eventcontract.Schema {
	t.Helper()
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read %s: %v", schemaPath, err)
	}
	var schemas map[string]eventcontract.Schema
	if err := json.Unmarshal(raw, &schemas); err != nil {
		t.Fatalf("unmarshal %s: %v", schemaPath, err)
	}
	return schemas
}

// 発行中のイベントが記録済みスキーマと後方互換であることを CI で守るフィットネス関数
// (ADR-[[202608160730]] / ADR-[[202607020343]])。
func TestContractsStayBackwardCompatible(t *testing.T) {
	recorded := recordedSchemas(t)
	implemented := map[string]bool{}

	contracts, err := eventcontract.Contracts()
	if err != nil {
		t.Fatalf("Contracts(): %v", err)
	}
	for _, c := range contracts {
		eventType := c.Event.EventType()
		implemented[eventType] = true
		t.Run(eventType, func(t *testing.T) {
			schema, ok := recorded[eventType]
			if !ok {
				t.Fatalf("%s が %s に記録されていない。新しいイベント種は記録を足してから発行する", eventType, schemaPath)
			}
			current := eventcontract.SchemaOf(c.Event)
			for _, err := range eventcontract.CheckCompatible(schema, current) {
				t.Errorf("%s: %v", eventType, err)
			}
			for _, err := range eventcontract.CheckRecorded(schema, current) {
				t.Errorf("%s: %v", eventType, err)
			}
			for _, err := range eventcontract.CheckParse(c, schema) {
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

func TestCheckCompatible(t *testing.T) {
	type args struct {
		recorded eventcontract.Schema
		current  eventcontract.Schema
	}
	tests := []struct {
		name     string
		args     args
		wantErrs int
	}{
		{
			"正常系 変更が無ければ互換",
			args{
				eventcontract.Schema{"orderId": {Type: "string"}},
				eventcontract.Schema{"orderId": {Type: "string"}},
			},
			0,
		},
		{
			"正常系 optional にした記録済みフィールドは削除できる",
			args{
				eventcontract.Schema{"orderId": {Type: "string"}, "amountCents": {Type: "int64", Optional: true}},
				eventcontract.Schema{"orderId": {Type: "string"}},
			},
			0,
		},
		{
			"準正常系 必須のまま削除すると非互換",
			args{
				eventcontract.Schema{"orderId": {Type: "string"}, "amountCents": {Type: "int64"}},
				eventcontract.Schema{"orderId": {Type: "string"}},
			},
			1,
		},
		{
			"準正常系 型変更は非互換",
			args{
				eventcontract.Schema{"orderId": {Type: "string"}},
				eventcontract.Schema{"orderId": {Type: "int64"}},
			},
			1,
		},
		{
			"正常系 フィールドの追加は互換",
			args{
				eventcontract.Schema{"orderId": {Type: "string"}},
				eventcontract.Schema{"orderId": {Type: "string"}, "currency": {Type: "string", Optional: true}},
			},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eventcontract.CheckCompatible(tt.args.recorded, tt.args.current); len(got) != tt.wantErrs {
				t.Fatalf("CheckCompatible() = %v, want %d errors", got, tt.wantErrs)
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

func TestCheckRecorded(t *testing.T) {
	type args struct {
		recorded eventcontract.Schema
		current  eventcontract.Schema
	}
	tests := []struct {
		name     string
		args     args
		wantErrs int
	}{
		{
			"正常系 発行中のフィールドがすべて記録されている",
			args{
				eventcontract.Schema{"orderId": {Type: "string"}},
				eventcontract.Schema{"orderId": {Type: "string"}},
			},
			0,
		},
		{
			"正常系 発行しなくなったフィールドの記録が残っていてよい",
			args{
				eventcontract.Schema{"orderId": {Type: "string"}, "gone": {Type: "string", Optional: true}},
				eventcontract.Schema{"orderId": {Type: "string"}},
			},
			0,
		},
		{
			"準正常系 記録に無いフィールドの発行は記録漏れ",
			args{
				eventcontract.Schema{"orderId": {Type: "string"}},
				eventcontract.Schema{"orderId": {Type: "string"}, "currency": {Type: "string"}},
			},
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eventcontract.CheckRecorded(tt.args.recorded, tt.args.current); len(got) != tt.wantErrs {
				t.Fatalf("CheckRecorded() = %v, want %d errors", got, tt.wantErrs)
			}
		})
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

func TestCheckParse(t *testing.T) {
	type args struct {
		parse    func(values map[string]any) error
		recorded eventcontract.Schema
	}
	tests := []struct {
		name     string
		args     args
		wantErrs int
	}{
		{
			"正常系 必須 / 任意が Parse の挙動と一致する",
			args{parseStub, eventcontract.Schema{"required": {Type: "string"}, "loose": {Type: "string", Optional: true}}},
			0,
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
			"準正常系 削除済みフィールドの記録は Parse の検査対象外",
			args{parseStub, eventcontract.Schema{"required": {Type: "string"}, "gone": {Type: "string", Optional: true}}},
			0,
		},
		{
			"準正常系 サンプルが Parse を通らないと検査が空回りするので弾く",
			args{
				func(map[string]any) error { return os.ErrInvalid },
				eventcontract.Schema{"required": {Type: "string"}, "loose": {Type: "string", Optional: true}},
			},
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract := eventcontract.Contract{Event: stubEvent{}, Parse: tt.args.parse}
			if got := eventcontract.CheckParse(contract, tt.args.recorded); len(got) != tt.wantErrs {
				t.Fatalf("CheckParse() = %v, want %d errors", got, tt.wantErrs)
			}
		})
	}
}

func TestSchemaOf(t *testing.T) {
	got := eventcontract.SchemaOf(stubEvent{})
	want := eventcontract.Schema{"required": {Type: "string"}, "loose": {Type: "string"}}
	if len(got) != len(want) {
		t.Fatalf("SchemaOf() = %v, want %v", got, want)
	}
	for name, field := range want {
		if got[name] != field {
			t.Fatalf("SchemaOf()[%q] = %v, want %v", name, got[name], field)
		}
	}
}
