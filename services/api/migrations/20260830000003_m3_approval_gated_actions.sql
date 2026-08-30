-- M3: approval/deny/expiry state machine for a mock network-block action.
-- See adr/0011-approval-gated-mock-network-block.md.

create table actions (
  id                text primary key,
  tenant_id         text not null,
  case_id           text not null references cases(id) on delete cascade,
  intent            text not null check (intent in ('block_network_indicator')),
  target_type       text not null,
  target_value      text not null,
  risk              text not null check (risk in ('low','medium','high','critical')),
  mode              text not null check (mode in ('dry_run','execute')) default 'execute',
  requested_by      text not null,
  status            text not null check (status in
                      ('pending_approval','approved','rejected','expired','executed','released')),
  idempotency_key   text not null unique,
  policy_version    text not null default 'policy-1.0',
  block_ttl_seconds integer not null default 600,
  executed_at       timestamptz,
  released_at       timestamptz,
  created_at        timestamptz not null default now()
);

create table approvals (
  id          text primary key,
  action_id   text not null unique references actions(id) on delete cascade,
  status      text not null check (status in ('pending','approved','rejected','expired')) default 'pending',
  approved_by text,
  reason      text,
  decided_at  timestamptz,
  expires_at  timestamptz not null,
  created_at  timestamptz not null default now()
);

create table network_blocks (
  id           bigserial primary key,
  tenant_id    text not null,
  action_id    text not null unique references actions(id) on delete cascade,
  target_value text not null,
  status       text not null check (status in ('active','released')) default 'active',
  created_at   timestamptz not null default now(),
  released_at  timestamptz
);
