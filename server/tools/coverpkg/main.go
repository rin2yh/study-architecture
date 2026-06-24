// Command coverpkg は go test -coverpkg に渡すパッケージ一覧を、go list の結果から
// 設定 YAML の除外パターンで絞り込んで出力する。CI の workflow から grep を追い出し、
// 除外条件を YAML 一箇所で宣言的に管理するための補助ツール。
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

type config struct {
	Exclude []string `yaml:"exclude"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "coverpkg:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	configPath := "server/coverpkg.yaml"
	if len(args) >= 2 && args[0] == "-config" {
		configPath, args = args[1], args[2:]
	}
	if len(args) == 0 {
		return fmt.Errorf("usage: coverpkg [-config path] followed by one or more go-list patterns")
	}

	excludes, err := loadExcludes(configPath)
	if err != nil {
		return err
	}

	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	var pkgs []string
	for _, pkg := range strings.Fields(string(out)) {
		if !matchesAny(pkg, excludes) {
			pkgs = append(pkgs, pkg)
		}
	}
	fmt.Println(strings.Join(pkgs, ","))
	return nil
}

func loadExcludes(path string) ([]*regexp.Regexp, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	res := make([]*regexp.Regexp, 0, len(cfg.Exclude))
	for _, pat := range cfg.Exclude {
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", pat, err)
		}
		res = append(res, re)
	}
	return res, nil
}

func matchesAny(s string, res []*regexp.Regexp) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}
