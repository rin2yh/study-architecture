// Package api はこのサービスの OpenAPI 由来の生成コード（api.gen.go）を含む。
//
// コード生成は依存順序があるため mise の `gen` タスクで段階実行する想定:
//
//  1. sqlc        : DB アクセスコード（internal/db）
//  2. oapi-codegen: API サーバ/型（このパッケージ）
//  3. kessoku     : DI 配線（internal/di）。handler が oapi の interface を実装するため最後。
//
// `go generate -run <tool> ./...` で個別に順序実行できるよう、各ツールの
// ディレクティブを分離している。
package api

//go:generate go tool sqlc generate -f ../sqlc.yaml
//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
