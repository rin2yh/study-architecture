# syntax=docker/dockerfile:1
#
# 単一ルート go.mod 構成。ビルドコンテキストはリポジトリルート:
#   docker build -f server/inventory/docker/server.Dockerfile -t ec-inventory .
# 生成コードはコミット済み前提。未生成なら先に `mise gen` を実行する。
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod --mount=type=cache,target=/root/.cache/go-build,id=gobuild CGO_ENABLED=0 go build -trimpath -o /out/inventory ./server/inventory/cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/inventory /inventory
EXPOSE 80
ENTRYPOINT ["/inventory"]
