-- Device groups: named sets of devices in one organization.

-- +goose Up
-- The membership key below refers to this pair, so a membership never joins
-- a device and a group of two organizations.
ALTER TABLE devices ADD CONSTRAINT devices_organization_id_id_key UNIQUE (organization_id, id);

CREATE TABLE device_groups (
  id bigserial PRIMARY KEY,
  organization_id bigint NOT NULL REFERENCES organizations (id),
  name text NOT NULL CHECK (btrim(name) <> ''),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id, id)
);

CREATE UNIQUE INDEX device_groups_name_key ON device_groups (organization_id, lower(name));

CREATE TABLE device_group_members (
  organization_id bigint NOT NULL,
  group_id bigint NOT NULL,
  device_id bigint NOT NULL,
  PRIMARY KEY (group_id, device_id),
  FOREIGN KEY (organization_id, group_id)
    REFERENCES device_groups (organization_id, id) ON DELETE CASCADE,
  FOREIGN KEY (organization_id, device_id) REFERENCES devices (organization_id, id)
);

CREATE INDEX device_group_members_device_idx ON device_group_members (organization_id, device_id);

-- +goose Down
DROP TABLE device_group_members;
DROP TABLE device_groups;
ALTER TABLE devices DROP CONSTRAINT devices_organization_id_id_key;
