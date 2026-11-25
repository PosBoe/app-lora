-- The core context: organizations, users, and login sessions.

-- +goose Up
CREATE TABLE organizations (
  id bigserial PRIMARY KEY,
  name text NOT NULL CHECK (btrim(name) <> ''),
  -- A deactivated organization has a time here. Its users cannot log in.
  deactivated_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id bigserial PRIMARY KEY,
  -- An admin is a member of no organization.
  organization_id bigint REFERENCES organizations (id),
  is_admin boolean NOT NULL DEFAULT false,
  -- The login is stored in lower case, so the unique index ignores case.
  login text NOT NULL CHECK (login = lower(login) AND login <> ''),
  password_hash text NOT NULL,
  password_temporary boolean NOT NULL,
  failed_logins integer NOT NULL DEFAULT 0,
  locked_until timestamptz,
  last_login_at timestamptz,
  first_name text,
  last_name text,
  email text,
  phone text,
  language text NOT NULL DEFAULT 'ro' CHECK (language IN ('ro', 'ru')),
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (is_admin = (organization_id IS NULL))
);

CREATE UNIQUE INDEX users_login_key ON users (login);
CREATE INDEX users_organization_id_idx ON users (organization_id);

CREATE TABLE sessions (
  -- The SHA-256 of the cookie token. The token itself is never stored.
  token_hash bytea PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL,
  expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- +goose Down
DROP TABLE sessions;
DROP TABLE users;
DROP TABLE organizations;
