# Go Learner Kit Agent Guide

## Workspace

- This is a multi-module workspace: `src/user-svc` and `src/authz-svc` are independent Go modules; `src/identity-svc` is a .NET 8 project. Run language commands from the relevant service directory unless using the root Makefile.
- Start the integrated stack with `cp .env.example .env` followed by `make up`. Root Compose owns service dependencies, one-shot migrations, OpenFGA, Prometheus, and Grafana.
- `user-service` (`:8080`) and `identity-service` (`:8081`) are host-exposed. `authz-service`, OpenFGA, and databases are Compose-network-only.

## Verification

- Run all suites with `make test`; run focused Go tests from a service with `go test ./internal/transport/http -run TestName` (adjust package/path), and identity tests with `dotnet test` in `src/identity-svc`.
- Run `make generate` after changing either OpenAPI contract. `user-svc` generated bindings are in `api/generated/`; never edit generated files manually.
- Build Tailwind output after changing identity UI styles: `npm run css:build` in `src/identity-svc`. The generated file is `wwwroot/lib/tailwind/tailwind-output.css`.

## Service Boundaries

- Keep `user-svc` changes layered: OpenAPI/HTTP transport -> `internal/service` -> `internal/repository`; pass the request context through each layer. Organization scoping must remain in repository queries.
- `user-svc` currently trusts `X-User-ID` and `X-Organization-ID` from a gateway or trusted caller; it does not validate bearer tokens. It calls `authz-svc` before user operations and must fail closed on denied or unavailable authorization.
- `authz-svc` is the only service that writes/checks OpenFGA tuples. Its embedded model and `Engine` interface live in `src/authz-svc/internal/authz`; custom roles may grant only `list_users` and `create_user`.
- `identity-svc` owns accounts, organization memberships, OIDC tokens, and organization selection. New organizations are provisioned as OpenFGA admins through the private authz endpoint using `Authz__ProvisioningToken`.

## Data And Configuration

- User schema changes use new ordered SQL migrations in `src/user-svc/migrations`; do not rewrite existing migrations. Identity schema changes use EF Core migrations in `src/identity-svc/Data/Migrations` and are applied by `identity-migrate` with `--migrate`.
- Compose configuration uses service DNS names and port `5432`. For local identity development, `src/identity-svc/compose.yaml` publishes PostgreSQL on `5444`; its Makefile exports `ConnectionStrings__Default` accordingly. Do not use the Compose hostname/port when running the app directly from the host.
- .NET hierarchical environment variables use double underscores, for example `ConnectionStrings__Default`, `OpenIddict__Issuer`, and `Authz__ServiceUrl`. Root `.env` supplies bootstrap IDs, admin credentials, and the provisioning token.
- `tmp/`, `src/identity-svc/node_modules/`, `bin/`, and `obj/` are ignored local artifacts; do not treat them as project source.
