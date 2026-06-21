# エラーハンドリング規約

エラーを **noop で握り潰さない**。`try-catch`（TS）や `if err != nil`（Go）で捕まえたエラーは、
必ず次のいずれかで「意味のある」扱いをする。

- **意味のあるフォールバックにする**: 失敗を想定済みの値へ変換する（例: セッション検証失敗 →
  `return null` で未ログイン扱い、bind 失敗 → 400 を積む）
- **ログを出す**: その場で握って続行するなら最低限ログに残す（TS は `console.warn` 等、Go は
  `slog`）。後段の挙動が変わらなくても、失敗が起きた事実は可視化する
- **再 throw / 伝播する**: ここで扱えないなら上位へ返す（Go は `return err`、TS は rethrow）

理由: 握り潰すと失敗が不可視になり、デバッグ不能・「嘘の成功」の温床になる。`catch {}` や
`catch { /* noop */ }`、`_ = err` での無言の破棄は、後から「なぜ動かないのか分からない」を生む。

- ✗ `try { await deleteSession(t); } catch { /* noop */ }`（失敗が消える）
- ✓ `try { await deleteSession(t); } catch (e) { console.warn("...", e); }`（続行するが可視化）

補足（強制の限界）:

- TS は oxlint の `no-empty`（`allowEmptyCatch: false`）で **空の** `catch {}` をエラーにしている。
  ただしコメントを 1 つ入れると回避できる（`catch { /* ... */ }` は通る）ため、lint だけでは
  noop を完全には防げない。最終的な担保はこの規約。
- Go の `_ = err` も同様に lint で機械的に全部は止められない。捕まえたら上の 3 つのどれかにする。
