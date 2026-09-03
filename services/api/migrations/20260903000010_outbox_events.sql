-- adr/0022: transactional outbox feeding services/api/searchindex's poller,
-- matching docs/02-architecture/data-architecture.md's already-decided
-- "transactional outbox + indexer worker" design for OpenSearch projection.
-- payload is the exact document body the poller PUTs to OpenSearch —
-- computed once, in the same transaction as the domain write, so the
-- projection can never observe a different value than what was committed.
create table outbox_events (
  id           text primary key,
  tenant_id    text not null,
  entity_type  text not null,
  entity_id    text not null,
  payload      jsonb not null,
  created_at   timestamptz not null default now(),
  indexed_at   timestamptz
);

create index outbox_events_unindexed on outbox_events (created_at) where indexed_at is null;
