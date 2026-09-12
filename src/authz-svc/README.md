# Authorization Service

`authz-svc` stores organization-admin tuples in OpenFGA and exposes internal permission and role-management APIs.

Required configuration:

- `FGA_API_URL`, for example `http://openfga:8080`
- `FGA_STORE_NAME`
- `BOOTSTRAP_ORGANIZATION_ID` and `BOOTSTRAP_ADMIN_USER_ID`, both UUIDs

On startup it creates or reuses the named OpenFGA store, writes the service model, and seeds the configured administrator. Role endpoints trust the internal caller's `X-User-ID` header and require that user to be an administrator of the organization.

## Custom roles

An organization administrator can create a custom role, grant it `list_users` or `create_user`, and assign it to users. Roles are identified by an API-generated UUID and are scoped to the organization that created them.

```text
POST   /v1/organizations/{organizationId}/roles
PUT    /v1/organizations/{organizationId}/roles/{roleId}/permissions/{permission}
PUT    /v1/organizations/{organizationId}/roles/{roleId}/users/{userId}
DELETE /v1/organizations/{organizationId}/roles/{roleId}/permissions/{permission}
DELETE /v1/organizations/{organizationId}/roles/{roleId}/users/{userId}
```

All role-management requests require the caller's `X-User-ID` header. Only direct organization admins can manage roles; custom roles cannot grant role-management access.
