# ADR-202606250141: テレメトリの秘匿情報は計装段と Alloy 段の二重でマスキングする

- Status: Accepted
- Date: 2026-06-25
- Relates to: ADR-[[202606241356]] (可観測性スタック), ADR-[[202606211100]] (Cookie セッション), ADR-[[202606230930]] (X-Member-Id)

## Context

テレメトリは payment (金額) / member (セッション・認証) を通る。無対策だと span 属性・ログ・メトリクス
のラベルに `Cookie` / `Authorization` / `X-Member-Id` / 金額 / email が混入し、Tempo/Loki に保存される。

## Decision

**計装段で入れない + Alloy 段で落とす**の二重。

- 計装段: `otelgin` / `otelhttp` の「ヘッダ・ボディを拾わない」既定を崩さない。span 属性・ログに秘匿情報を
  手で入れない。span 名はルートテンプレート。
- Alloy 段: deny-list (`Cookie` / `Set-Cookie` / `Authorization` / `X-Member-Id` / session / email / 金額 /
  カード) を drop / ハッシュ。設定は 1 箇所。
- メトリクスのラベルに PII・高カーディナリティ (会員 id 等) を使わない。

## Consequences

- 片方が漏れてももう片方で止まる。
- deny-list の維持コスト (秘匿情報が増えたら Alloy 設定に追記)。
- 自由記述ログに埋め込むと取りこぼしうる。

## Alternatives considered

- 計装段だけ → 新しい計装が秘匿情報を拾い始めると素通し。安全網が無い。
- Alloy 段だけ → deny-list の漏れがそのまま流出。
