-- Ingestion: the integrations of an organization and the uplinks they deliver.

-- +goose Up
CREATE TABLE integrations (
  id bigserial PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations (id),
  name text NOT NULL CHECK (btrim(name) <> ''),
  kind text NOT NULL CHECK (kind IN ('chirpstack', 'ttn')),
  -- The SHA-256 of the secret. The secret itself is never stored.
  secret_hash bytea NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_uplink_at timestamptz,
  last_error_at timestamptz,
  last_error_code text,
  last_error_message text,
  UNIQUE (organization_id, id)
);

CREATE TABLE uplinks (
  id bigserial PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations (id),
  -- Null after the integration is deleted; the uplink stays.
  integration_id bigint REFERENCES integrations (id) ON DELETE SET NULL,
  -- Null for an unmatched uplink.
  device_id bigint,
  dev_eui text NOT NULL CHECK (dev_eui ~ '^[0-9a-f]{16}$'),
  f_cnt bigint NOT NULL,
  payload bytea NOT NULL,
  -- The time the network server received the uplink, else the arrival time.
  received_at timestamptz NOT NULL,
  stored_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (organization_id, device_id) REFERENCES devices (organization_id, id)
);

-- A network server that sends an uplink again sends the same counter and
-- time, so the copy hits this key and is not stored.
CREATE UNIQUE INDEX uplinks_delivery_key ON uplinks (organization_id, dev_eui, f_cnt, received_at);
CREATE INDEX uplinks_device_idx ON uplinks (organization_id, device_id, received_at DESC);
CREATE INDEX uplinks_unmatched_idx ON uplinks (organization_id, dev_eui) WHERE device_id IS NULL;
CREATE INDEX uplinks_integration_idx ON uplinks (integration_id, stored_at);
CREATE INDEX uplinks_purge_idx ON uplinks (stored_at) WHERE device_id IS NULL;

-- +goose Down
DROP TABLE uplinks;
DROP TABLE integrations;
