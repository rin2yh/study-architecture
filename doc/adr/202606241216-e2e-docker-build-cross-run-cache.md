# ADR-202606241216: E2E の docker ビルドを cross-run キャッシュ前提に変える (まず root .dockerignore)

- Status: Accepted
- Date: 2026-06-24
- Relates to: ADR-[[202606170902]] (単一ルート go.mod モノレポ), ADR-[[202606220716]] (CI を量子で 2 分割), [[ci.md]] (CI 規約), [[docker.md]] (docker 規約)

## Context

「E2E の start stack が遅い」を計測した (main, run 28071921118 / e2e job)。e2e ジョブの内訳:

- install playwright browser: 30s
- **start stack (`scripts/e2e-up.sh store`): 158s**
- e2e テスト本体: **5s**

テストは 5s で、ジョブ時間はほぼ全部スタック起動。158s の内訳 (BuildKit ログ実測):

- DB 起動 + migrate(5 本): ~16s
- **バックエンド 6 サービスの逐次ビルド: ~94s** ([[docker.md]] により 1 つずつ build)
- store(UI) ビルド: ~30s
- compose up --wait + healthcheck: ~16s

94s の支配項は **最初の 1 本 (product) が 66s** で突出する。内部はさらに:

- golang ベースイメージ pull: 8s
- `go mod download`: 21s
- `go build` (stdlib + 全依存の初回コンパイル): 30s

2 本目以降が 4〜11s で済むのは、ベースイメージ・`go mod download` レイヤ (モノレポで go.mod
が全サービス共通 = 同一レイヤ。ADR-[[202606170902]]) と、`RUN --mount=type=cache` の go-build /
go-mod マウントが **同一 run 内で温まって**再利用されるため。つまり product 固有が重いのではなく、
一回限りのコスト (pull + mod download + 初回コンパイル ≒ 59s) を 1 本目が全部背負っている。

問題は CI runner が毎回まっさらで、**run をまたいだキャッシュが無い**こと。毎回 7 イメージを
ゼロからビルドするため、158s 中 ~124s がビルドに化けている。

ここで効くはずの cross-run キャッシュには 2 つの壁がある:

1. **どの `cache_to` バックエンド (gha / registry / local) も `RUN --mount=type=cache` のマウント
   内容は永続化しない**。永続化されるのはイメージ「レイヤ」キャッシュだけ。よって cross-run で
   確実に効くのは「ベースイメージ」「`go mod download` レイヤ (go.mod/go.sum 依存)」「変更が無い
   サービスのイメージ丸ごと」で、変更されたサービスの `go build` 再コンパイル (~30s) はマウントが
   空のまま残る。
2. **ルートに `.dockerignore` が無く**、Go ビルドの `COPY . .` がリポジトリルート全体 (client/ や
   doc/ を含む) をコンテキストに取り込む。このため client だけの変更でも全 Go サービスの `COPY . .`
   レイヤが無効化され、上の「変更が無いサービスのレイヤ再利用」が成立しない。レイヤキャッシュの
   前提を壊している。

## Decision

cross-run レイヤキャッシュを効かせるための **前提として、まず root `.dockerignore` を入れる**。

- Go イメージのビルドコンテキストはリポジトリルート (`context: .`)。`go build` が必要とするのは
  `go.mod` / `go.sum` / `server/` だけで、`go:embed` も無い (計測で確認)。
- よって `client/` `doc/` `infra/` `scripts/` `.github/` `.claude/` `.git/` `**/*.md` 等を除外する。
  これで (a) build context 転送 (実測 8s) が縮み、(b) `COPY . .` レイヤのキャッシュキーが「Go に
  関係するファイルの変更」だけで動くようになり、無関係な client/doc 変更でレイヤが無効化されない。
- store(UI) ビルドのコンテキストは `./client` で `client/.dockerignore` が別にあるため、root の
  `.dockerignore` は Go イメージ (root context) だけに効く。

この ADR は `.dockerignore` の導入を確定する。実際に run をまたいでキャッシュを保存・復元する配線
(BuildKit のレイヤキャッシュ `cache_from`/`cache_to`) は、バックエンドの選択 (GitHub Actions cache
`type=gha` か、GHCR への `type=registry` か) と、それに伴う buildx container driver / 認証 action の
追加という外部影響を含む判断のため、別途決める (下記 Consequences)。

## Consequences

- `.dockerignore` 単体では **cold run (キャッシュ未保存の初回) の start stack は速くならない**。
  これは cross-run キャッシュの「ヒット率を上げる前提」であり、配線が入って 2 回目以降の run で
  キャッシュが復元されて初めて効く。
- 配線後に確実に取り戻せるのは「ベースイメージ pull (~8s)」「`go mod download` レイヤ (~21s)」
  「変更が無いサービスのイメージ丸ごと」。典型的な 1 サービス変更の PR では、変更外サービスが
  レイヤ復元で済み、`.dockerignore` のおかげで client/doc だけの変更なら Go 6 本が全ヒットしうる。
- 残る課題は、変更されたサービスの `go build` 再コンパイル (~30s)。これは `--mount=type=cache` が
  cross-run で復元されないため。さらに削るには (a) cache mount を `actions/cache` 等で別途持ち回る
  (buildkit-cache-dance)、または (b) 各 Dockerfile の `COPY . .` を「自サービス + server/internal」
  だけの選択 COPY に絞り、サービス間でレイヤを独立させる、が候補。いずれも本 ADR の範囲外。
- [[docker.md]] の「1 つずつ build」は OrbStack (ローカル) の daemon I/O 競合対策で、CI runner の
  挙動とは別。CI で並列 build に振るのは別判断 (本 ADR では触れない)。

## Alternatives considered

- **`.dockerignore` を入れず cache だけ配線する**: client/doc 変更で Go 6 本のレイヤが毎回無効化
  され、「変更外サービスのレイヤ復元」がほぼ効かない。前提を欠いた最適化になるため不採用。
- **ビルド対象を E2E 用に間引く**: store の external スタックは edge-proxy が shipping に
  depends_on するため、shipping/shipping-worker も起動が要る。間引きはテストの健全性を崩す risk が
  あり、得る時間に見合わないため不採用。
- **cache mount を即 buildkit-cache-dance で持ち回る**: 再コンパイル (~30s) まで削れるが、新規
  action 追加 + container driver + compose の load 挙動という未検証要素が増える。前提 (`.dockerignore`)
  を先に確定し、効果を計測してから判断する。
