#!/bin/sh
# usage: up.sh [repo-root] [work-dir]
#   → Grafana http://127.0.0.1:3000 (匿名 Admin)、Prometheus :9090、Alloy OTLP :4317
set -eu

repo=${1:-/home/user/study-architecture}
work=${2:-/tmp/grafana-shot}
mkdir -p "$work"

if ! docker info > /dev/null 2>&1; then
	nohup dockerd > "$work/dockerd.log" 2>&1 &
	i=0
	while ! docker info > /dev/null 2>&1; do
		i=$((i + 1))
		[ "$i" -lt 40 ] || { echo "dockerd did not start; see $work/dockerd.log" >&2; exit 1; }
		sleep 1
	done
fi

# Docker Hub 直は blob の CDN がネットワークポリシーに弾かれるので、pull-through cache から取って
# compose のイメージ名に付け替える。
for img in $(grep -oE 'image: (prom/prometheus|grafana/grafana|grafana/alloy):[^ ]+' "$repo/compose.yaml" | awk '{print $2}'); do
	docker image inspect "$img" > /dev/null 2>&1 && continue
	echo "pulling $img" >&2
	docker pull -q "mirror.gcr.io/$img" > /dev/null
	docker tag "mirror.gcr.io/$img" "$img"
done

# compose はホストへ Grafana しか公開しない。検証プロセスが直接叩く分だけ足す。
cat > "$work/ports.yml" <<'EOF'
services:
  alloy:
    ports: ["4317:4317"]
  prometheus:
    ports: ["9090:9090"]
EOF

docker compose -f "$repo/compose.yaml" -f "$work/ports.yml" --profile observability \
	up -d --no-build --no-deps prometheus grafana alloy >&2

i=0
while ! curl -sf http://127.0.0.1:3000/api/health > /dev/null; do
	i=$((i + 1))
	[ "$i" -lt 60 ] || { echo "grafana did not become healthy" >&2; exit 1; }
	sleep 1
done
curl -s http://127.0.0.1:3000/api/health
echo "compose file: $work/ports.yml" >&2
