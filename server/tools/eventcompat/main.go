// Command eventcompat は base の記録と作業ツリーの記録を突き合わせ、非互換な変更を落とす
// (ADR-[[202608160730]])。
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/rin2yh/study-architecture/server/internal/eventcontract"
)

func main() {
	base := flag.String("base", "origin/main", "比較元のコミット")
	flag.Parse()

	if err := run(*base); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func run(base string) error {
	previous, err := recordedAt(base)
	if err != nil {
		return err
	}
	current, err := eventcontract.LoadSchemas(eventcontract.SchemaPath)
	if err != nil {
		return err
	}
	errs := eventcontract.CheckSchemas(previous, current)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if len(errs) > 0 {
		os.Exit(1)
	}
	return nil
}

// ref を解決できないときに空として扱うと検査が無言で無効化される。記録そのものがまだ無い導入直後
// だけ空で続ける。
func recordedAt(base string) (map[string]eventcontract.Schema, error) {
	if out, err := exec.Command("git", "rev-parse", "--verify", base+"^{commit}").CombinedOutput(); err != nil {
		return nil, fmt.Errorf("base %q を解決できない: %s", base, out)
	}
	blob := base + ":" + eventcontract.SchemaPath
	if err := exec.Command("git", "cat-file", "-e", blob).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s が無いため空の記録として比較する\n", blob)
		return map[string]eventcontract.Schema{}, nil
	}
	raw, err := exec.Command("git", "show", blob).Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s: %w", blob, err)
	}
	return eventcontract.ParseSchemas(raw, blob)
}
