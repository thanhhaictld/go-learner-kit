# Environment variables
DOTNET_ENVIRONMENT ?= Development
ConnectionStrings__Default ?= Host=localhost;Port=5444;Database=identity;Username=app;Password=app

export DOTNET_ENVIRONMENT
export ConnectionStrings__Default

.PHONY: run migrate test build

watch:
	docker compose up -d
	dotnet watch run

run:
	docker compose up -d
	dotnet run

migrate:
	dotnet run -- --migrate

test:
	dotnet test

build:
	dotnet build
