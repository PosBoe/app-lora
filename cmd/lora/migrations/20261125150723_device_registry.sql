-- The device registry: the device models and the devices of each organization.

-- +goose Up
-- pg_trgm indexes the device search, which matches any part of a name, a
-- DevEUI, or an address.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE device_models (
  id bigserial PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations (id),
  vendor text NOT NULL CHECK (btrim(vendor) <> ''),
  model text NOT NULL CHECK (btrim(model) <> ''),
  created_at timestamptz NOT NULL DEFAULT now(),
  -- devices refers to this pair, so a device cannot use the model of another
  -- organization.
  UNIQUE (organization_id, id)
);

CREATE UNIQUE INDEX device_models_name_key
  ON device_models (organization_id, lower(vendor), lower(model));

CREATE TABLE devices (
  id bigserial PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations (id),
  -- 16 lower-case hex characters. An archived device keeps its DevEUI.
  dev_eui text NOT NULL CHECK (dev_eui ~ '^[0-9a-f]{16}$'),
  name text NOT NULL CHECK (btrim(name) <> ''),
  device_model_id bigint NOT NULL,
  meter_type text CHECK (meter_type IN ('water', 'gas')),
  address text,
  note text,
  archived_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  search text GENERATED ALWAYS AS (
    lower(name || ' ' || dev_eui || ' ' || coalesce(address, ''))
  ) STORED,
  FOREIGN KEY (organization_id, device_model_id) REFERENCES device_models (organization_id, id),
  UNIQUE (organization_id, dev_eui)
);

CREATE INDEX devices_list_idx ON devices (organization_id, (archived_at IS NULL), lower(name), id);
CREATE INDEX devices_model_idx ON devices (organization_id, device_model_id);
CREATE INDEX devices_search_idx ON devices USING gin (search gin_trgm_ops);

-- +goose Down
DROP TABLE devices;
DROP TABLE device_models;
DROP EXTENSION IF EXISTS pg_trgm;
