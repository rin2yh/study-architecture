---
paths:
  - "compose.yaml"
  - "**/Dockerfile"
  - "Dockerfile*"
  - "infra/**/*"
---

# docker 規約

`docker compose` をローカル (OrbStack 含む) で動かすときの方針。

## 全イメージを 1 ショットで build しない

`docker compose --profile external --profile internal up -d --build` を一発で叩くと、
buildkit が 8 イメージ (backend 5 + UI 3) を並列に build しはじめて、OrbStack の docker
daemon が捌ききれなくなる:

- daemon socket が `use of closed network connection` で切断される
- buildkit との RPC が `DeadlineExceeded` / `unexpected EOF` で落ちる
- 結果としてビルドが終わらず時間切れになる

**対策**: 必ず **サービスを 1 つずつ build → まとめて up** の順で扱う。

```bash
# 個別 build (1 つずつ)
for svc in product order payment member shipping; do
  docker compose build "$svc"
done
docker compose --profile external build store mypage
docker compose --profile internal build backoffice

# 全部 build できたら image だけで起動 (--no-build で並列 build を抑止)
docker compose --profile external --profile internal up -d --no-build
```

`mise run up` で `up -d --build` を呼ぶ前提を捨てて、build が必要なときは個別実行する。
これにより daemon の I/O 競合が解消される。

## daemon が詰まった後のリカバリ

- `docker ps` が空 / hang する → OrbStack が応答していない
- まず `docker ps` が返るか確認 (返らないなら OrbStack を再起動)
- その後、上の「1 つずつ build」を順に実行する
- それでも特定サービスだけ詰まる場合は、そのサービスに `--no-cache` を付けて再 build
