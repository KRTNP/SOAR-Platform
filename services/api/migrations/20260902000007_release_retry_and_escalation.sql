-- docs/08-testing/lab-scenario-ssh-bruteforce.md: "retry rollback at 5, 15
-- and 30 seconds; retain the block and escalate if all fail" — the
-- scheduler previously just retried every tick indefinitely
-- (adr/0014's disclosed "no backoff/alerting" gap). See adr/0019.
alter table actions
  add column release_attempts  int not null default 0,
  add column next_retry_at     timestamptz,
  add column escalated         boolean not null default false;
