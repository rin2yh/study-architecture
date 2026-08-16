# ADR-202606190901: orval 生成物を gen/ ディレクトリに集約する

- Status: Accepted
- Date: 2026-06-19

## Context

`client/app/api/src/` 配下で orval 生成物と手書き（mutator・バレル）が混在し、生成物かどうかが
ファイル先頭のヘッダコメントでしか判別できなかった。

生成物を lint/format から外す `ignorePatterns` が `package/` 時代の古いパスを指したまま壊れており、
`pnpm format` が orval 出力を再整形していた（パス依存の ignore が壊れても気づけない実例）。

## Decision

Go 側の規約（sqlc / oapi-codegen の生成物は専用の場所。ADR-[[202606170901]]）を client 側にも適用し、
orval の出力先を `src/gen/<service>/**` に集約する。

- 手書き（mutator・バレル）は `src/` 直下に維持し、`package.json` の `exports` もバレルを指したままにする。
  **公開境界はバレル**であり、生成レイアウトの移動に追従させない。
- lint/format の `ignorePatterns` を実パスへ修正し、`.gitattributes` で `linguist-generated=true` にする。

## Consequences

- 生成物と手書きがディレクトリ境界で一目で区別でき、ignore がパスのリネームで壊れても
  「生成物が整形される」形では露見しなくなる。
- 生成物は format 対象外となり orval ネイティブの整形がそのまま残る。手書きのみ oxfmt の対象。
- 生成物をコミットする方針（ADR-[[202606170901]]）は維持する。

## Alternatives considered

- **ヘッダコメントのみで区別を続ける**: ignore がパス依存で壊れやすく、現状の問題が残る。
  ディレクトリ境界で根治する。
- **バレルも `gen/` 配下に移す**: バレルはエラーモデル（ADR-[[202606180901]]）の型を明示 re-export する
  手書きであり、生成物ではない。公開境界として `src/` 直下に残す。
