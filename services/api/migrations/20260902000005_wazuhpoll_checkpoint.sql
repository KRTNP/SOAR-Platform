-- Persists services/api/wazuhpoll's poll checkpoint (adr/0017 disclosed this
-- as in-memory-only, reset to "now" on every restart, silently dropping any
-- alert that arrived during the restart window). The boolean-primary-key
-- trick keeps this to exactly one row, a settings-singleton pattern.
create table wazuhpoll_checkpoint (
  id              boolean primary key default true check (id),
  last_polled_at  timestamptz not null
);
