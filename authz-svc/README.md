# Authorization Service

`authz-svc` stores organization-admin tuples in OpenFGA and exposes internal permission and role-management APIs.

Required configuration:

- `FGA_API_URL`, for example `http://openfga:8080`
- `FGA_STORE_NAME`
- `BOOTSTRAP_ORGANIZATION_ID` and `BOOTSTRAP_ADMIN_USER_ID`, both UUIDs

On startup it creates or reuses the named OpenFGA store, writes the service model, and seeds the configured administrator. Role endpoints trust the internal caller's `X-User-ID` header and require that user to be an administrator of the organization.
