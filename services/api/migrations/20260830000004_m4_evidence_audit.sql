-- M4: evidence metadata + append-only audit log.
-- See adr/0012-evidence-integrity-and-audit-export.md.

create table evidence (
  id                  text primary key,
  tenant_id           text not null,
  case_id             text not null references cases(id) on delete cascade,
  source_event_id     text references events(id),
  artifact_type       text not null,
  uri                 text not null,
  sha256              text not null,
  size                integer not null,
  source              text not null,
  acquisition_method  text not null,
  collector           text not null,
  collected_at        timestamptz not null,
  created_at          timestamptz not null default now(),
  received_at         timestamptz not null,
  timezone            text not null default 'UTC',
  tool_name           text not null,
  tool_version        text not null,
  command_ref         text not null,
  analyst             text not null,
  purpose             text not null,
  limitations         text[] not null default '{}',
  derived_from        text[] not null default '{}',
  sensitivity         text not null check (sensitivity in ('public','internal','confidential','restricted')),
  retention_class     text not null,
  integrity_status    text not null check (integrity_status in ('verified','pending','mismatch','unavailable')) default 'pending'
);

create index evidence_case_id_idx on evidence (case_id);

create table evidence_custody_events (
  id           bigserial primary key,
  evidence_id  text not null references evidence(id) on delete cascade,
  event        text not null,
  actor        text not null,
  at           timestamptz not null,
  from_hash    text,
  to_hash      text
);

create index evidence_custody_events_evidence_id_idx on evidence_custody_events (evidence_id);

create table audit_log (
  id                bigserial primary key,
  tenant_id         text not null,
  event_id          text,
  request_id        text,
  actor             text not null,
  actor_type        text not null,
  case_id           text,
  execution_id      text,
  action            text not null,
  target_hash       text,
  policy_version    text not null default 'policy-1.0',
  connector_version text,
  decision          text not null check (decision in ('allowed','denied','completed')),
  reason            text,
  "timestamp"       timestamptz not null default now(),
  result_hash       text not null
);

create index audit_log_tenant_timestamp_idx on audit_log (tenant_id, "timestamp" desc);
