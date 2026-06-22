# ADR-202606220900: サービスの HTTP 起動処理を共通 httpx に集約しカバレッジを通す

- Status: Accepted
- Date: 2026-06-22
- Relates to: ADR-[[202606170900]] (サービスベースアーキテクチャ) / ADR-[[202606170907]] (Gin 採用) / ADR-[[202606180901]] (API エラーモデル)

## Context

各サービスの `main.go` (`run()`) は、Gin エンジン構築 (Recovery + ErrorJSON) → `RegisterHandlers` →
`http.Server` 起動 → SIGTERM でグレースフルシャットダウン、という同一処理を 5 サービス
(member / order / payment / product / shipping-server) でコピペしていた。

カバレッジは coverpkg (`server/coverpkg.yaml`) の通り main を分母に含める。だが既存の `main_test.go` は
`DATABASE_URL` 未設定で `di.InitHandler` が失敗する異常系しか踏んでおらず、計測すると `run` は 25%・
`main` は 0% に留まっていた。原因は (1) 起動本体が DB 接続を必要とし、(2) `:80` 固定で bind し、
(3) `run()` が内部で `signal.NotifyContext` を作るためテストから ctx をキャンセルできない、の 3 点。

## Decision

**共通の起動処理を `server/internal/httpx` に切り出し、各 `main.go` を薄くする。あわせて `run` を
テスト可能な形へ整える。**

- `httpx.NewEngine()` / `httpx.Serve(ctx, addr, handler)` / `httpx.ListenAddr()` を提供する。
  `httpx` は DB 非依存なので単体テストで起動〜グレースフル停止まで実測でき、coverpkg の分母にも入る。
- `signal.NotifyContext` を `main()` 側へ移し、`run(ctx)` は注入された ctx を使う。テストは
  キャンセル済み ctx を渡せば起動直後にグレースフル停止し、happy path を踏める。
- 待ち受けは既定 `:80` のまま、`LISTEN_ADDR` で上書き可能にする。テストは `127.0.0.1:0` を渡す。
- happy path テストが実 DB 無しで通るのは、`pgxpool.New` / `redis.ParseURL` が遅延接続で、到達不能
  URL でも `InitHandler` / `InitConsumer` が成功するため (接続は初回クエリまで起きない)。

## Consequences

- **計測対象の main が薄くなり happy path も踏める**: 起動本体は `httpx` に移って単体テストで覆われ、
  各 `run` は異常系 (init 失敗) と正常系 (キャンセルでグレースフル停止) の両方を踏むようになる。
- **重複の単一化**: Gin の組み立てと shutdown 手順が 1 箇所になり、5 サービスで挙動が揃う。worker は
  HTTP を持たないため `httpx` は使わず、`run(ctx)` 化のみ行う。
- **`main()` は計測上 0% のまま**: `os.Exit` を含むため意味のあるテストが書けない。signal 配線と
  `run` 呼び出しだけに保ち、ロジックは `run` 以下へ寄せる。
- **`LISTEN_ADDR` という新しい設定面が増える**: 本番は未設定で従来通り `:80`。テスト/ローカル専用。

## Alternatives considered

- **main をカバレッジ分母から除外する**: coverpkg.yaml は意図して main を含めている。除外は計測の
  ごまかしで、起動経路の回帰 (shutdown 漏れ等) を見逃すため採らない。
- **実 DB を立てて結合テストで happy path を踏む**: 既に integration ジョブがあるが、`run` は `:80`
  bind とブロッキングで単体から扱えない。遅延接続を使えば DB 無しで起動経路を踏めるため不要。
- **共通化せず各 main に happy path テストを足す**: テスト 5 重複に加え、起動本体の重複自体は残る。
  起動ロジックを 1 箇所 (`httpx`) に集約する方が重複も計測も綺麗。
