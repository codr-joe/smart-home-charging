-- TimescaleDB initial schema for Smart Charging Phase 1

CREATE TABLE IF NOT EXISTS energy_readings (
  time    TIMESTAMPTZ      NOT NULL,
  power_w DOUBLE PRECISION NOT NULL,
  solar_w DOUBLE PRECISION,
  tariff  CHAR(2)
);

SELECT create_hypertable('energy_readings', 'time', if_not_exists => TRUE);

-- Retain raw data for 90 days; continuous aggregates keep longer history.
SELECT add_retention_policy('energy_readings', INTERVAL '90 days', if_not_exists => TRUE);

-- Hourly continuous aggregate for fast historical queries.
CREATE MATERIALIZED VIEW IF NOT EXISTS energy_hourly
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1 hour', time) AS bucket,
  AVG(power_w)                AS avg_power_w,
  MIN(power_w)                AS min_power_w,
  MAX(power_w)                AS max_power_w,
  AVG(solar_w)                AS avg_solar_w
FROM energy_readings
GROUP BY bucket;

SELECT add_continuous_aggregate_policy('energy_hourly',
  start_offset => INTERVAL '3 hours',
  end_offset   => INTERVAL '1 hour',
  schedule_interval => INTERVAL '1 hour',
  if_not_exists => TRUE);
