.PHONY: generate run watch test fmt

generate:
	go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml

run:
	go run ./cmd/api

watch:
	go tool air -c .air.toml

test:
	go test ./...

fmt:
	go fmt ./...
