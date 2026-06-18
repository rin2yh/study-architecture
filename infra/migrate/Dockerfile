# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/goose github.com/pressly/goose/v3/cmd/goose

FROM alpine:3.20
COPY --from=build /out/goose /usr/local/bin/goose
COPY server/order/db/migration    /migrations/order
COPY server/payment/db/migration  /migrations/payment
COPY server/member/db/migration   /migrations/member
COPY server/product/db/migration  /migrations/product
COPY server/shipping/db/migration /migrations/shipping
ENTRYPOINT ["goose"]
