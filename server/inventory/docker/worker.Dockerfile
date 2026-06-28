# syntax=docker/dockerfile:1
#
# 決済確定イベントを購読し TTL 回収も回す worker。HTTP の inventory とは別バイナリ・別デプロイで
# web/worker を独立させる (ADR-[[202606211200]] / ADR-[[202606262000]])。
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod,id=gomod --mount=type=cache,target=/root/.cache/go-build,id=gobuild CGO_ENABLED=0 go build -trimpath -o /out/inventory-worker ./server/inventory/cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/inventory-worker /inventory-worker
ENTRYPOINT ["/inventory-worker"]
