
-- +goose Up

CREATE TYPE SPANKIND AS ENUM ('server', 'client', 'unspecified', 'producer', 'consumer', 'ephemeral', 'internal');

CREATE TABLE services (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
CREATE INDEX idx_services_name ON services(name);

CREATE TABLE operations (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  service_id BIGINT REFERENCES services(id) NOT NULL,
  kind SPANKIND NOT NULL,

  UNIQUE (name, kind, service_id)
);
CREATE INDEX idx_operations_name ON operations(name);

-- here our primary key is not named id as is customary. Instead we have named
-- it hack_id. This is to help us differentiate it from span_id. We can't use
-- span_id as our primary key because its not guaranteed to be unique as a
-- compatability measure with zipkin.
CREATE TABLE spans (
  hack_id BIGSERIAL PRIMARY KEY,
  span_id BYTEA NOT NULL,
  trace_id BYTEA NOT NULL,
  operation_id BIGINT REFERENCES operations(id) NOT NULL,
  service_id BIGINT REFERENCES services(id) NOT NULL,
  flags BIGINT NOT NULL,
  start_time TIMESTAMP NOT NULL,
  duration INTERVAL NOT NULL,
  tags JSONB,
  process_id TEXT NOT NULL,
  process_tags JSONB NOT NULL,
  warnings TEXT[],
  logs JSONB,
  kind SPANKIND NOT NULL,
  refs JSONB NOT NULL
);

CREATE INDEX idx_trace_id ON spans (trace_id);
CREATE INDEX idx_spans_operation_service ON spans(operation_id, service_id);
CREATE INDEX idx_spans_start_duration ON spans(start_time, duration);
CREATE INDEX idx_spans_start_time ON spans(start_time);
CREATE INDEX idx_spans_duration ON spans(duration);
CREATE INDEX idx_spans_tags ON spans USING GIN (tags);
CREATE INDEX idx_spans_process_tags ON spans USING GIN (process_tags);

-- +goose Down

DROP TABLE spans;
DROP TABLE operations;
DROP TABLE services;

DROP TYPE SPANKIND;