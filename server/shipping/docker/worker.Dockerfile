# syntax=docker/dockerfile:1
#
# 決済確定イベントを購読する worker。HTTP の shipping (Dockerfile) とは別バイナリ・別デプロイ
# で web/worker を独立させる (ADR-[[202606211200]])。
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/shipping-worker ./server/shipping/cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/shipping-worker /shipping-worker
ENTRYPOINT ["/shipping-worker"]
