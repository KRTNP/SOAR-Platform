-- docs/08-testing/lab-scenario-ssh-bruteforce.md: "Escalation: Medium
-- becomes higher priority only when source targets a privileged user or
-- critical asset." Same shape as trusted_sources (adr/0018) — a distinct
-- table because "this source should never be blocked" and "this target
-- deserves higher priority" are unrelated concerns that happen to share a
-- CRUD shape. See adr/0019.
create table critical_targets (
  id          text primary key,
  tenant_id   text not null,
  value       text not null,
  reason      text not null,
  created_by  text not null,
  created_at  timestamptz not null default now()
);

create unique index critical_targets_tenant_value_uq on critical_targets (tenant_id, value);
