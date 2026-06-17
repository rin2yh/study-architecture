package api

//go:generate go tool sqlc generate -f ../sqlc.yaml
//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml
