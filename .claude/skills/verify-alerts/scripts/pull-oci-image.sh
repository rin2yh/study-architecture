#!/bin/sh
# usage: pull-oci-image.sh <repo> <tag> <dest-rootfs>
#   例: pull-oci-image.sh grafana/grafana 12.2.0 /tmp/.../grafana-root
#
# Docker Hub から直接引くと blob が CDN (production.cloudfront.docker.com) へリダイレクトされ、
# エージェント環境のネットワークポリシーに 403 で弾かれる。mirror.gcr.io は Docker Hub の
# pull-through cache で、manifest も blob も同一ホストから認証なしで返るためここを使う。
set -eu

repo=$1
tag=$2
dest=$3
reg=https://mirror.gcr.io/v2
accept_index='application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json'
accept_manifest='application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json'

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

curl -sSf -H "Accept: $accept_index, $accept_manifest" "$reg/$repo/manifests/$tag" > "$work/index.json"
digest=$(jq -r '.manifests[]? | select(.platform.os == "linux" and .platform.architecture == "amd64") | .digest' "$work/index.json" | head -1)
if [ -n "$digest" ] && [ "$digest" != "null" ]; then
	curl -sSf -H "Accept: $accept_manifest" "$reg/$repo/manifests/$digest" > "$work/manifest.json"
else
	# multi-arch でないタグはインデックスを挟まず manifest がそのまま返る。
	cp "$work/index.json" "$work/manifest.json"
fi

mkdir -p "$dest"
i=0
for d in $(jq -r '.layers[].digest' "$work/manifest.json"); do
	i=$((i + 1))
	curl -sSfL -o "$work/layer.tgz" "$reg/$repo/blobs/$d"
	tar xzf "$work/layer.tgz" -C "$dest" 2>/dev/null || tar xf "$work/layer.tgz" -C "$dest"
	echo "layer $i extracted" >&2
done
echo "$dest"
