// Package product は order が product サービスを呼ぶための生成 HTTP クライアント。
package product

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../product/api/openapi.yaml
