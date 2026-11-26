-- Decoding: the decoder of a device model, the decoded values of an uplink,
-- and the alarms of a device.

-- +goose Up
-- The id of a decoder in device/decoder; null when the model has none.
ALTER TABLE device_models ADD COLUMN decoder text;

-- decoder names the decoder that wrote the decoded values. decode_pending
-- marks a row that the decode worker must decode again: a new uplink of a
-- device, an attached one, or every uplink of a device whose decoder changed.
ALTER TABLE uplinks
  ADD COLUMN decoder text,
  ADD COLUMN pulse_count bigint,
  ADD COLUMN alarms_reported text[],
  ADD COLUMN alarms_raised text[],
  ADD COLUMN decode_error text,
  ADD COLUMN decode_pending boolean NOT NULL DEFAULT false;

UPDATE uplinks SET decode_pending = true WHERE device_id IS NOT NULL;

-- The receive time up to which the alarms of a device are computed. Null
-- means the alarms must be computed again from all decoded uplinks.
ALTER TABLE devices ADD COLUMN alarms_through timestamptz;

CREATE TABLE alarms (
  id bigserial PRIMARY KEY,
  organization_id bigint NOT NULL,
  device_id bigint NOT NULL,
  kind text NOT NULL,
  raised_at timestamptz NOT NULL,
  -- Null while the alarm is active.
  cleared_at timestamptz,
  raised_uplink_id bigint REFERENCES uplinks (id) ON DELETE SET NULL,
  cleared_uplink_id bigint REFERENCES uplinks (id) ON DELETE SET NULL,
  FOREIGN KEY (organization_id, device_id) REFERENCES devices (organization_id, id)
);

CREATE INDEX alarms_device_idx ON alarms (organization_id, device_id, raised_at DESC);
CREATE INDEX alarms_active_idx ON alarms (organization_id, device_id) WHERE cleared_at IS NULL;
CREATE INDEX uplinks_pending_idx ON uplinks (id) WHERE decode_pending;
CREATE INDEX uplinks_decoded_idx ON uplinks (organization_id, device_id, received_at)
  WHERE pulse_count IS NOT NULL;

-- +goose Down
DROP INDEX uplinks_decoded_idx;
DROP INDEX uplinks_pending_idx;
DROP TABLE alarms;
ALTER TABLE devices DROP COLUMN alarms_through;
ALTER TABLE uplinks
  DROP COLUMN decoder,
  DROP COLUMN pulse_count,
  DROP COLUMN alarms_reported,
  DROP COLUMN alarms_raised,
  DROP COLUMN decode_error,
  DROP COLUMN decode_pending;
ALTER TABLE device_models DROP COLUMN decoder;
