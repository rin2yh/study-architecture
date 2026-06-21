# ADR 0017: what コメント検出を Claude Code hook + LLM 判定で行う

- Status: Proposed
- Date: 2026-06-21
- Relates to: `.claude/rules/comments.md` (コメント規約)

## Context

`.claude/rules/comments.md` で「コメントは why だけ・what は書かない」と定めているが、
AI 補完を含めて what コメント (コードの言い換え) が繰り返し混入する。規約文があっても
書かれてしまうため、書かれた時点で気づける仕組みが要る。

what と why の区別は **意味判定**であり、コードの字面からは機械的に落とせない。

## Decision

**PostToolUse hook で、編集に含まれるコメントを LLM (claude -p / Haiku) に
`comments.md` 基準で判定させ、what コメントを検出したら Claude に差し戻す。**

- 配線: `.claude/settings.json` の `PostToolUse` / matcher `Edit|Write|MultiEdit`。
  実体は `.claude/hooks/check-what-comments.sh`。
- 規約は **転記せず** `comments.md` を読み込んで渡す。規約のソースは comments.md 単一。
- LLM を呼ぶ前に決定論的に絞る (呼び出しコスト削減):
  - 対象拡張子 (`go/ts/tsx/js/jsx/sql/yaml/yml/sh`) 以外は対象外
  - 生成コード (`*.gen.go` / `*.sql.go` / `Code generated` マーカー) は対象外
  - コメント記号を含まない編集は LLM を呼ばずに通す
- `asyncRewake` で実行する。編集自体はブロックせず、違反検出時だけ Claude を起こして
  是正させる (毎編集の待ち時間をゼロにする)。
- 判定不能・LLM エラー・空応答は **fail-open** (exit 0) で正当な編集を止めない。
- 再入防止: hook が起動する `claude -p` には `CLAUDE_WHAT_COMMENT_HOOK=1` と
  `--settings '{"disableAllHooks":true}'` を渡す。

## Consequences

- **適用範囲は Claude Code セッション内のみ**。人手の手書きや他エディタには効かない。
  チーム全体での強制が要るなら CI 側の追加が別途必要 (本 ADR の対象外)。
- **非決定的**。LLM の判定揺れがあり 100% ではない。fail-open なので「止めすぎない」側に倒す。
- **コスト**: コメントを含む編集ごとに Haiku を 1 回呼ぶ。前段の決定論フィルタで
  大半の編集は LLM を呼ばない。
- settings.json はセッション開始時に無いと監視対象に入らないため、初回は `/hooks` を
  開くかセッション再起動で有効化が要る。

## Alternatives considered

- **golangci-lint**: 一般的なコメント衛生 (godot/dupword 等) は入るが、what/why の意味判定は
  できず、本件は解決しない。Go 限定で comments.md の「全言語対象」とも噛み合わない。
- **自作 go/analysis アナライザ**: ヒューリスティック (直後の行と識別子が重複 等) に頼ると
  誤検知・見逃しが多くノイズ過大。費用対効果が低い。
- **CI で LLM レビュー**: チーム全体に効く利点はあるが、是正は別途人/AI が行う。書いた瞬間の
  是正という今回の目的 (「毎回 AI が追加してめんどくさい」) には hook の方が直接効く。
  将来 CI ゲートとして併用する余地は残す。
