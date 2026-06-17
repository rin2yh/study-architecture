---
paths:
  - "server/**/*.go"
---

# server 規約

server 配下の Go コードを編集するときに守る方針。

- **`server/internal/httperror` はなるべく使わない方向に寄せる**
  - 新しい `main.go` を増やす・既存 `main.go` を編集するときは、まず `api.NewStrictHandler(h, nil)` のデフォルト動作で済まないか考える
  - Step 0 の Get-only 骨格や、内部詳細の露出を気にしなくてよい開発段階ではデフォルトで十分
  - 内部エラー文言を本番でクライアントに見せたくない場面で初めて httperror に切り替える。または handler 内で `api.ListXxx500JSONResponse{...}` のような **型付きエラーレスポンス** を直接返す
  - 理由: 共通基盤を増やすほど main.go との結合が増え、サービス境界がぼやける。サービスを「素直に立ち上げる」Step 0 方針 (ADR 0001) と整合させる
  - 既存 5 サービスの httperror 経由は無理に剥がさない。次に main.go を編集する機会に「使わない方向で書けるか」を再検討する程度でよい
