# ADR 0009: 認証は HttpOnly Cookie + member 所有のサーバ側セッション

- Status: Accepted
- Date: 2026-06-19

## Context

UI（mypage）にログイン/ログアウトの導線を入れ、会員を認証したい。[[0001]] のサービスベース
構成では認証の所有者を 1 つに定める必要がある。会員情報は member ドメインが持つので、
パスワードとセッションも member が所有するのが自然。

[[0006]] で UI のデータ取得は **サーバ側ローダ → 各サービス HTTP 呼び出し**に閉じている。
ブラウザは UI オリジンだけを叩き、サービス URL やトークンはサーバ側に閉じる方針。認証もこの
形に合わせ、ブラウザに JWT を持たせる SPA 的な方式ではなく **サーバ側セッション**を採る。

## Decision

### セッションの所有と保存

- 認証は member サービスが所有する。`member.sessions` テーブルと
  `member.members.password_hash` 列を migration で追加する。
- パスワードは **bcrypt**（`golang.org/x/crypto/bcrypt`）でハッシュ化して保存する。平文・可逆
  暗号は保存しない。`password_hash` は `DEFAULT ''` で追加し、空文字は bcrypt 照合が必ず失敗
  する＝「ログイン不可の会員」を表す（既存行の移行にダミーパスワードを撒かずに済む）。
- セッションは **不透明トークン**（32 byte 乱数の base64url）。トークンの **SHA-256 hex** を
  `member.sessions.id`（主キー）に保存する。生トークンは Cookie だけが保持し、DB には載せない。
  DB が流出しても生トークンを復元できず、既存セッションを乗っ取られないため。

### API（member）

- `POST /sessions`（ログイン）: email + password を検証し、成功時に 201 で `Session`
  （`id` = 生トークン / `memberId` / `expiresAt`）を返す。失敗は **401 `unauthorized`**。
  「メール無し」と「パスワード不一致」で文言を変えない（user enumeration を避けるため）。
- `GET /sessions/{id}`（検証）: `{id}` は生トークン。サーバは SHA-256 して引き、`expires_at >
  now()` の行があれば 200、無ければ 404。
- `DELETE /sessions/{id}`（ログアウト）: 生トークンをハッシュして該当行を削除。存在しない id
  でも **冪等に 204**。
- 既存エンドポイントは path id を `int64`（`IdPath`）に統一していたが、セッションの `{id}` は
  推測可能な連番にできない（連番だと総当たりでセッションを引ける）ため **string（不透明
  トークン）**とする。これは [[0015]] までの int64 規約から意図的に外れる箇所。
- エラーモデル（[[0014]]）に **401 `unauthorized`** を追加（`middleware.Unauthorized`）。

### UI（mypage / [[0011]] React Router v7）

- ログイン: `POST /sessions` 成功で生トークンを **HttpOnly Cookie**（`member_session`）に
  載せ、サーバ側ローダ/アクションだけが読む。XSS でトークンを盗まれないため HttpOnly。
- 保護ページのローダは Cookie のトークンで `GET /sessions/{id}` を叩いて検証し、得た `memberId`
  を **`X-Member-Id` ヘッダ**として下流サービス（order 等）へ受け渡す。未ログイン/失効なら
  `/login` へリダイレクト。
- ログアウト: `DELETE /sessions/{id}` でサーバ側セッションを破棄し、Cookie を失効させる。
  破棄が失敗しても Cookie は必ず消してログアウトを成立させる。
- Cookie 属性は `HttpOnly; SameSite=Lax; Path=/`。**`Secure` は付けない**。ローカル学習
  スタックは edge-proxy が http 終端で、Secure Cookie はブラウザに保存されずログインが成立
  しないため。TLS 終端を入れる段で `Secure` を足すこと（下記 Consequences）。
- セッション TTL は 7 日（member 側 `sessionTTL` と Cookie `Max-Age` を揃える）。

## Consequences

- トークンは HttpOnly Cookie に閉じ JS から読めない。DB には SHA-256 ハッシュしか無いので、
  DB 流出単体では生トークンを復元できない。失効・ログアウトはサーバ側削除で即時に効く
  （ステートレス JWT と違いブラックリスト不要）。
- `X-Member-Id` は現状 UI が「認証済みの会員 id」を下流へ伝える経路を作るだけで、order 等の
  下流サービスはまだ参照しない（Step 0 は直接呼び出し）。下流での認可（自分の注文だけ見える
  等）は別 ADR / 別イシューで、信頼境界（`X-Member-Id` を誰が付けてよいか）と併せて決める。
- **`Secure` 未設定は本番では不適**。公開時は TLS 終端の edge を前提に `Secure` を有効化する。
  この ADR の決定は「ローカル完結・費用ゼロ」（[[0001]]）の制約下での妥協であることを明記する。
- path id の型がエンドポイントで二系統（int64 / string）になる。セッションだけの例外として
  `SessionIdPath`（string）を OpenAPI に定義し、規約から外れる理由を本 ADR に固定する。

## Alternatives considered

- **ステートレス JWT をブラウザ（localStorage / 非 HttpOnly Cookie）に保持**: サーバ側ストア
  不要だが、失効が難しく（短命化 + リフレッシュが要る）、XSS でのトークン奪取に弱い。学習用に
  認可・失効の挙動を素直に観察したいので不採用。
- **セッション id を連番（int64）にして既存 `IdPath` 規約に合わせる**: 規約は揃うが、id が
  推測可能になり総当たりでセッションを引ける。セキュリティ上不可。
- **生トークンをそのまま DB 主キーにする**: 実装は単純だが、DB 流出で全セッションを即乗っ取
  られる。ハッシュ保存で被害を限定する。
- **専用の auth/identity サービスを新設**: 認証を 1 か所に集約できるが、会員を所有するのは
  member であり、Step 0 でのサービス追加は早すぎる複雑化（[[0001]] のロードマップ）。
