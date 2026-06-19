// Package shipping は order が shipping サービスを呼ぶための生成 HTTP クライアント。
package shipping

//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../../../shipping/api/openapi.yaml
