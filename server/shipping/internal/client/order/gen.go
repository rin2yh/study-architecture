// Package order は shipping が order サービスを呼ぶための生成 HTTP クライアント。
package order

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../order/api/openapi.yaml
