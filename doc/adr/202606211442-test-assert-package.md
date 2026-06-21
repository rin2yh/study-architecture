# ADR-202606211442: テストヘルパーを test/assert パッケージに集約する

- Status: Accepted
- Date: 2026-06-21
- Relates to: ADR-[[202606180901]] (API エラーモデル)

## Context

テスト用ヘルパーが用途別に 2 パッケージへ分散していた。

- `internal/test/apitest`: `AssertErrorCode` のみ (エラーレスポンスの code 検証)
- `internal/test/cmptest`: `Equal` / `EqualSlice` (cmp.Diff 定型の比較)

呼び出し側 (handler テスト) は両方を import しており、アサーションの入口が分かれていた。
それぞれ 1〜2 関数しか持たず、パッケージを分ける積極的な理由がない。

## Decision

**両パッケージを `internal/test/assert` に統合する。**

- 公開 API: `assert.DeepEqual` / `assert.DeepEqualSlice` / `assert.ErrorCode`。
- cmp.Diff を使う比較は浅い `==` でなく深い構造比較なので、`reflect.DeepEqual` に倣って
  `Deep` 接頭辞を付け、浅い等価でないことを名前で示す。
- `apitest.AssertErrorCode` は `assert` 配下では `assert.Assert...` の stutter になるため、
  Go の慣習に従い `Assert` 接頭辞を落として `assert.ErrorCode` にする。
- 旧 `apitest` / `cmptest` パッケージは削除し、参照を全面的に差し替える。

## Consequences

- テストのアサーションは `assert` 単一の入口に揃い、import が 1 つに減る。
- `apitest` / `cmptest` を参照する外部はテストコードのみで、移行は機械的置換で完結した。
- 関数が増えて 1 ファイルが肥大化したら、`assert` 配下でファイル分割する余地は残す
  (パッケージは分けない)。

## Alternatives considered

- **現状維持 (2 パッケージ)**: 用途で分かれている見た目はあるが、各 1〜2 関数では
  分割の便益が薄く、import が増えるコストだけが残る。
- **`apitest` に寄せる**: `apitest` は HTTP レスポンス前提の命名で、cmp 比較 (DB 層テストでも
  使う) を含めると名前と実態がずれる。中立な `assert` を入口にする方が収まりが良い。
