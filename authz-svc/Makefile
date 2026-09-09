.PHONY: generate run test fmt

generate:
	go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml

run:
	go run ./cmd/api

test:
	go test ./...

fmt:
	go fmt ./...
