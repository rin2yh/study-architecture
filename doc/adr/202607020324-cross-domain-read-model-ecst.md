# ADR-202607020324: 横断参照データの既定を ECST + ローカル read model にする (役割で snapshot / pull と使い分け)

- Status: Accepted
- Date: 2026-07-02
- Relates to: ADR-[[202607011621]] (マイクロサービス移行。本 ADR は具体化タスク 3), ADR-[[202606190900]] (注文時 snapshot), ADR-[[202606301000]] (shipping←order の同期 pull), ADR-[[202606301100]] (BFF→order の値渡し), ADR-[[202606261212]] (Outbox = 更新経路), ADR-[[202606261214]] (冪等投影), GitHub #98 (親)

## Context

- 横断データの持ち方が 3 パターン併存する: snapshot (注文時に商品名/単価を order_items へ。ADR-[[202606190900]]) / 同期 pull (shipping が order から宛先を引く。ADR-[[202606301000]]) / 値渡し (BFF が住所を解決し order へ。ADR-[[202606301100]])。加えて checkout の order→product は同期 HTTP 参照 (`FetchProduct`)。
- 方向 ADR-[[202607011621]] は「各サービスが自データを所有し他ドメインの必要分は自前で持つ。持ち方の既定は後続 ADR で確定」とし、event-carried state transfer (ECST) + ローカル read model を候補に挙げた。今は横串の原則が無く、持ち方が ADR ごとに別解になる。
- checkout の order→product は前進フローに残る同期結合で、product 障害が checkout を BadGateway に落とす (量子の可用性が独立しない)。

## Decision

横断参照データの既定を **ECST + 各サービスのローカル read model** にする。ただし全部を read model にせず、**役割で使い分ける判断軸**を定める。

- **判断軸 (ハイブリッド)**:
  - **時点事実 → snapshot 据置**: 注文時の単価など固定すべき事実は凍結コピー (ADR-[[202606190900]])。現在値へ追従させない。
  - **現在の共有参照 → ECST + read model (既定)**: read-mostly のカタログ等、他サービスが「今の値」を読みたい参照データは、所有者のイベントを購読してローカル read model に追従する。同期参照を置かない。
  - **強整合 / PII 回避 → 同期 pull 限定**: 直近の強整合が要る、または payload に PII を載せたくない場合だけ同期 pull を残す (shipping←order 宛先 ADR-[[202606301000]] はこの枠)。
- **read model は読み取り専用の投影**: 書けるのは所有者だけで所有権は割らない (横断 JOIN 禁止は不変)。更新は所有者の Outbox イベント (ADR-[[202606261212]]) 経由で、冪等に投影する (ADR-[[202606261214]] 同型)。結果整合を受け入れる。
- **最初の変換対象は order→product**: order が product のローカル read model (id / name / price 等) を持ち、checkout の同期 `FetchProduct` を外す。注文明細への時点 snapshot は維持し、両パターンを共存させる。product 障害が checkout の価格解決を止めない (前進フロー非同期化のタスク 2 と噛む)。以降の対象は別 issue。
- **既存パターンの位置づけ**: snapshot (ADR-[[202606190900]]) = 時点事実で据置、pull (ADR-[[202606301000]]) = PII 回避で同期維持、値渡し (ADR-[[202606301100]]) = edge 合成で不変。read model は新設の「現在の共有参照」枠。

## Consequences

- 横断の持ち方に単一の判断軸ができ、以降は「時点事実 / 現在参照 / 強整合・PII」の 3 択で裁ける。
- read model 化した参照は結果整合になる (所有者から遅延)。請求価格など時点固定すべき値は引き続き snapshot が担う (使い分けの要)。
- 複製が増える (product データが order にも載る)。マイクロサービスの独立と引き換えの想定内。
- read model を持つ側にコンシューマと投影表・その冪等更新が要る。
- 最初の対象 order→product で checkout の同期結合が 1 つ減り、主駆動特性 (独立デプロイ / 可用性) に効く。

## Alternatives considered

- **すべて ECST へ寄せる (snapshot も read model 化)**: 統一は美しいが、注文時価格などの時点事実の再現性を失う。時点固定が要る値に read model は不向き。役割で分ける。
- **case-by-case 維持**: 統一原則を作らず都度選ぶ。判断が属人化し、横断の持ち方が ADR ごとに割れる現状を温存する。
- **同期 pull を既定に (共有 read API)**: 実装は単純だが同期結合と可用性の巻き込みが残り、方向 ADR-[[202607011621]] の「前進フローは非同期を既定」に反する。
- **TTL キャッシュ (同期 pull のキャッシュ)**: read model に似るが、所有者イベントでなく TTL で陳腐化を許すため無効化の契機を持てず、イベント追従に劣る。
