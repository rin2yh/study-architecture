// Command eventcompat は main に記録済みのイベントスキーマと PR のスキーマを突き合わせ、非互換な
// 変更を落とす (ADR-[[202608160730]])。go test 側の検査はコードと記録の一致しか見られず、削除の前に
// 「optional へ落とす PR」を挟んだかは過去の記録と比べないと判定できないため、CI からこれを呼ぶ。
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rin2yh/study-architecture/server/internal/eventcontract"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: eventcompat <recorded schema.json> <current schema.json>")
		os.Exit(2)
	}
	recorded, err := load(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	current, err := load(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	errs := eventcontract.CheckSchemas(recorded, current)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if len(errs) > 0 {
		os.Exit(1)
	}
}

func load(path string) (map[string]eventcontract.Schema, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var schemas map[string]eventcontract.Schema
	if err := json.Unmarshal(raw, &schemas); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return schemas, nil
}
