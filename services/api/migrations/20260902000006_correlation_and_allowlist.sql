-- adr/0018: activates cases.correlation_key (present since the initial
-- schema, never written by any Go code until now) for one signature —
-- repeated authentication-class events sharing a source IP land on the same
-- case instead of each opening a new one. correlation_window_seconds
-- mirrors the column alerts already has, so a future second signature can
-- pick its own window instead of a shared constant. repeat_count is what
-- the SSH brute-force auto-trigger threshold reads.
alter table cases
  add column correlation_window_seconds  int,
  add column repeat_count                int not null default 1;

-- Trusted/management sources that must never be auto-blocked
-- (docs/08-testing/lab-scenario-ssh-bruteforce.md: "allowlist changes are
-- audited and expire"). No index beyond the primary key at lab scale.
create table trusted_sources (
  id          text primary key,
  tenant_id   text not null,
  value       text not null,
  reason      text not null,
  created_by  text not null,
  created_at  timestamptz not null default now(),
  expires_at  timestamptz
);

create unique index trusted_sources_tenant_value_uq on trusted_sources (tenant_id, value);
