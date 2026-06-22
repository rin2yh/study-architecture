# ADR-202606221300: 社外フロントを store に統合し mypage を退役する

- Status: Accepted
- Date: 2026-06-22
- Relates to: ADR-[[202606170900]] / ADR-[[202606170909]] / ADR-[[202606211100]] / ADR-[[202606220300]]

## Context

社外向け UI を `store`（買い物: 商品一覧 / カート / チェックアウト）と `mypage`（会員: ログイン /
ログアウト / 注文履歴）の 2 アプリに分けていた。両者は独立デプロイ量子として compose の別
service・別ポート（5173 / 5174）・CI matrix の別エントリを持つ。

しかし `mypage` は login / logout / 注文履歴の 3 ルートのみと小さく、`store` と次が重複している:

- `entities/session`（Cookie セッション、ADR-[[202606211100]]）がほぼ同一実装。`store` は読み取り
  (`currentMemberId`) だけ、`mypage` は書き込み (`sessionCookie` / `clearSessionCookie`) も持つ
  という差分しかない。
- 依存先サービス（member / order）が重なる。`store` のチェックアウトは既に member セッションを
  前提にしており、認証導線は本来同じ顧客体験の一部。
- 共有 `ui`（ADR-[[202606220300]]）のデザイン適用先が二重になる。

独立デプロイ量子に分ける便益（個別スケール・個別デプロイ）より、重複コードと運用面
（compose service / CI / edge-proxy 経路 / 見た目同期）の維持コストが上回ると判断した。本リポジトリは
学習用途で、社外顧客体験を 1 アプリに集約する単純さの価値が高い。

## Decision

`mypage` を廃止し、その機能を `store` のルートとして取り込む。社外フロントは `store` 単一アプリにする。

- ルート追加: `/login`・`/logout`・`/orders`（注文履歴。旧 `mypage` の index）。`store` の index は
  従来どおり商品一覧。
- `entities/session` は `store` に一本化し、Cookie 書き込み関数（`sessionCookie` /
  `clearSessionCookie`）を統合する。`features/auth`（`LoginForm` / `LogoutButton`）と注文履歴の
  表示コンポーネントを `store` へ移す。FSD の層分け（ADR-[[202606220300]]）はそのまま踏襲する。
- 退役: compose の `mypage` service とポート 5174、CI の client matrix の `mypage`、`mypage`
  パッケージ一式を削除する。edge-proxy は member / order を `store` が引き続き使うため変更しない。

社外/社内のネットワーク分離（ADR-[[202606170909]]）の方針は変わらない（`store` は従来どおり
external 側）。

## Consequences

- 社外デプロイ量子が `store` / `mypage` の 2 つから `store` 1 つに減る。session / auth の重複が
  解消し、認証導線が買い物と同じアプリに収まる。フロントのデプロイ量子は `store` / `backoffice`
  の 2 つになる（issue #38 の量子一覧もそれに従う）。
- トレードオフ: 買い物と会員機能が同一デプロイになり、独立デプロイ性を失う。一方の変更が他方の
  デプロイに乗る。学習用途では許容する。将来分離が必要になったら、共通化した `entities/session`
  をパッケージに切り出して再分割できる。
- `store` の責務が増える（依存サービスに member / order の認証・履歴系が明示的に乗る）。

## Alternatives considered

- **現状維持（store + mypage の 2 アプリ）**: 独立デプロイ性は保てるが、session / auth の重複と
  compose / CI / デザイン同期の運用コストが残る。mypage が小さく便益が薄いため却下。
- **mypage を残し、共通分を shared パッケージへ切り出す**: 重複は減るが、小規模な 2 アプリのために
  パッケージ境界を増やすのは過剰。単一アプリ化の方が単純で却下。
