#!/bin/sh
# usage: start-grafana.sh <rootfs> <repo-root> [prometheus-url]
#   例: start-grafana.sh /tmp/.../grafana-root /home/user/study-architecture http://127.0.0.1:9090
#
# 検証したいのは「compose が配るのと同じ provisioning」なので、マウント構成と環境変数は compose から写す。
# 公式イメージは Alpine (musl) 前提のバイナリで、ホストの glibc では動かないため chroot で走らせる。
set -eu

rootfs=$1
repo=$2
prom=${3:-http://127.0.0.1:9090}

mkdir -p "$rootfs/etc/grafana/provisioning/datasources" \
	"$rootfs/etc/grafana/provisioning/alerting" \
	"$rootfs/etc/grafana/provisioning/dashboards" \
	"$rootfs/var/lib/grafana/dashboards" \
	"$rootfs/var/log/grafana"

# datasource だけは compose のサービス名で解決できないのでローカルへ差し替える。
sed "s#http://prometheus:9090#$prom#" "$repo/infra/o11y/grafana-datasources.yaml" \
	> "$rootfs/etc/grafana/provisioning/datasources/datasources.yaml"
cp "$repo"/infra/o11y/alerting/*.yaml "$rootfs/etc/grafana/provisioning/alerting/"
cp "$repo/infra/o11y/grafana-dashboards.yaml" "$rootfs/etc/grafana/provisioning/dashboards/dashboards.yaml"
cp "$repo"/infra/o11y/dashboards/*.json "$rootfs/var/lib/grafana/dashboards/"

cat > "$rootfs/run-grafana.sh" <<'EOF'
#!/bin/sh
export GF_PATHS_DATA=/var/lib/grafana
export GF_PATHS_LOGS=/var/log/grafana
export GF_PATHS_PLUGINS=/var/lib/grafana/plugins
export GF_PATHS_PROVISIONING=/etc/grafana/provisioning
# compose と同じ匿名 Admin。ログイン画面を挟まないのでスクショが撮れる。
export GF_AUTH_ANONYMOUS_ENABLED=true
export GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
export GF_AUTH_DISABLE_LOGIN_FORM=true
export GF_SERVER_HTTP_ADDR=127.0.0.1
cd /usr/share/grafana
exec /usr/share/grafana/bin/grafana server --homepath=/usr/share/grafana --config=/usr/share/grafana/conf/defaults.ini
EOF
chmod +x "$rootfs/run-grafana.sh"

nohup chroot "$rootfs" /run-grafana.sh > "$rootfs/../grafana.log" 2>&1 &
echo "grafana pid: $!" >&2

i=0
while [ "$i" -lt 40 ]; do
	if curl -sf http://127.0.0.1:3000/api/health > /dev/null; then
		curl -s http://127.0.0.1:3000/api/health
		exit 0
	fi
	i=$((i + 1))
	sleep 1
done
echo "grafana did not become healthy; see grafana.log" >&2
exit 1
