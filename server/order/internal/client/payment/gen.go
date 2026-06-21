// Package payment は order が payment サービスを呼ぶための生成 HTTP クライアント。
package payment

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../payment/api/openapi.yaml
