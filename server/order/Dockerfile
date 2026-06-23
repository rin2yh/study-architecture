# syntax=docker/dockerfile:1
#
# 単一ルート go.mod 構成。ビルドコンテキストはリポジトリルート:
#   docker build -f services/order/Dockerfile -t ec-order .
# 生成コードはコミット済み前提。未生成なら先に `mise gen` を実行する。
FROM golang:1.26-alpine AS build
WORKDIR /src

# 依存解決ファイルを先に COPY してレイヤキャッシュを活かす
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod go mod download

# ソース一式を COPY し、このサービスの cmd だけビルドする
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod --mount=type=cache,target=/root/.cache/go-build,id=gobuild CGO_ENABLED=0 go build -trimpath -o /out/order ./server/order

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/order /order
EXPOSE 80
ENTRYPOINT ["/order"]
