DROP INDEX ix_users_organization_created_at;
DROP INDEX ux_users_organization_email;

ALTER TABLE users DROP COLUMN organization_id;

CREATE UNIQUE INDEX ux_users_email ON users(email);
