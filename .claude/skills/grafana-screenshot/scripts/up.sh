#!/bin/sh
# usage: up.sh <work-dir> [service...]   (リポジトリルートで実行する)
#   → Grafana http://127.0.0.1:3000 (匿名 Admin)、Prometheus :9090、Alloy OTLP :4317
set -eu

work=${1:?usage: up.sh <work-dir> [service...]}
shift
services=${*:-prometheus grafana alloy}
mkdir -p "$work"

wait_for() {
	limit=$1
	shift
	i=0
	while ! "$@" > /dev/null 2>&1; do
		i=$((i + 1))
		[ "$i" -lt "$limit" ] || return 1
		sleep 1
	done
}

if ! docker info > /dev/null 2>&1; then
	nohup dockerd > "$work/dockerd.log" 2>&1 &
	wait_for 40 docker info || {
		echo "dockerd did not start; see $work/dockerd.log" >&2
		exit 1
	}
fi

# Docker Hub 直は blob の CDN がネットワークポリシーに弾かれる。
conf=$(docker compose --profile observability config --format json)
for svc in $services; do
	img=$(echo "$conf" | jq -r --arg s "$svc" '.services[$s].image')
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

# shellcheck disable=SC2086
docker compose -f compose.yaml -f "$work/ports.yml" --profile observability up -d --no-deps $services >&2
wait_for 60 curl -sf --max-time 2 http://127.0.0.1:3000/api/health || {
	echo "grafana did not become healthy" >&2
	exit 1
}
