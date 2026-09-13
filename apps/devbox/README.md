# Aspire Development AppHost

`apphost.cs` replaces the host-watch orchestration previously implemented by
`scripts/watch.ps1`. It keeps the existing public local endpoints so the OIDC
issuer and BFF callback registrations remain valid.

Run it from the repository root:

```powershell
make watch
```

The Aspire dashboard shows resource status and logs. The BFF is available at
`http://localhost:3001`, Mailpit at `http://localhost:8025`, Prometheus at
`http://localhost:9090`, and Grafana at `http://localhost:3000`.

The AppHost reads optional bootstrap overrides from the repository `.env` file.
Install portal dependencies with `npm ci` in `apps/saas-admin-portal` before
starting it when `node_modules` is absent.
