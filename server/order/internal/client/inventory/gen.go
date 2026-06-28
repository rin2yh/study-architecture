// Package inventory は order が inventory サービスを呼ぶための生成 HTTP クライアント。
package inventory

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../inventory/api/openapi.yaml
