ALTER TABLE users ADD COLUMN organization_id UUID NOT NULL;

DROP INDEX ux_users_email;

CREATE UNIQUE INDEX ux_users_organization_email ON users(organization_id, email);
CREATE INDEX ix_users_organization_created_at ON users(organization_id, created_at DESC);
