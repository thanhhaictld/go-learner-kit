# Identity Service

`identity-svc` is the OpenIddict and ASP.NET Core Identity provider for this workspace. It owns authentication, organizations, organization membership, and OpenID Connect tokens; authorization decisions remain in `authz-svc`/OpenFGA.

## Local endpoints

- Login UI: `http://localhost:8081/Identity/Account/Login`
- Discovery: `http://localhost:8081/.well-known/openid-configuration`
- Health: `/health/live`, `/health/ready`
- Metrics: `/metrics`

The development seed creates the configured bootstrap organization and admin. The `saas-web` client is seeded with authorization-code + PKCE and refresh-token permissions. Its development redirect URI is `http://localhost:3001/auth/callback`.

## Configuration

Use `ConnectionStrings__Default`, `OpenIddict__Issuer`, `Authz__ServiceUrl`, `Authz__ProvisioningToken`, and `Bootstrap__*` environment variables. `Bootstrap__AdminPassword` and `Authz__ProvisioningToken` must be replaced outside local development.

Run migrations and the seed:

```powershell
dotnet run --project src/identity-svc -- --migrate
```

Then start the host:

```powershell
dotnet run --project src/identity-svc
```
