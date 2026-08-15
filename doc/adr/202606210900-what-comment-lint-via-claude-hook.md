# ADR-202606210900: what コメント検出を Claude Code hook + LLM 判定で行う

- Status: Accepted
- Date: 2026-06-21
- Relates to: `.claude/rules/comments.md` (コメント規約)

## Context

- [[comments.md]] で「コメントは why だけ」と定めているが、AI 補完を含めて what コメントが繰り返し混入する。
  規約文だけでは防げず、書かれた時点で気づける仕組みが要る。
- what と why の区別は**意味判定**で、コードの字面からは機械的に落とせない。

## Decision

**PostToolUse hook で、編集に含まれるコメントを LLM (claude -p / Haiku) に判定させ、what コメントを
検出したら Claude に差し戻す。**

- 規約は hook に**転記せず** [[comments.md]] を読み込んで渡す。ソースを 1 つに保つため。
- LLM を呼ぶ前に決定論的に絞る (対象拡張子・生成コード除外・コメント記号の有無)。呼び出しコストの削減。
- **非同期で起こす** (`asyncRewake`)。編集自体はブロックせず、違反時だけ是正させる。毎編集の待ち時間を
  ゼロにするため。
- 判定不能・LLM エラー・空応答は **fail-open**。止めすぎて正当な編集を妨げる方が害が大きい。
- hook が呼ぶ `claude -p` では hook を無効化する。再入を防ぐため。

配線とフィルタの具体値は `.claude/settings.json` と `.claude/hooks/check-what-comments.sh` が SSOT。

## Consequences

- **適用範囲は Claude Code セッション内のみ**。人手の手書きや他エディタには効かない。チーム全体での
  強制が要るなら CI 側の追加が別途要る。
- **非決定的**。LLM の判定揺れがあり 100% ではない。fail-open なので見逃し側に倒れる。
- コメントを含む編集ごとに Haiku を 1 回呼ぶ。大半の編集は前段のフィルタで LLM に届かない。
- settings.json はセッション開始時に読まれるため、初回は `/hooks` を開くか再起動が要る。

## Alternatives considered

- **golangci-lint** — 一般的なコメント衛生は入るが what/why の意味判定はできず、Go 限定で
  [[comments.md]] の「全言語対象」とも噛み合わない。
- **自作 go/analysis アナライザ** — ヒューリスティックでは誤検知・見逃しが多くノイズ過大。
- **CI で LLM レビュー** — チーム全体に効くが、是正が書いた瞬間から離れる。本 ADR の目的
  (書いた瞬間に直す) には hook が直接効く。将来の併用余地は残す。
