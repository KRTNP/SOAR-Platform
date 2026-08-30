-- M2: mock IOC enrichment and mock ticketing connectors.
-- See adr/0010-mock-enrichment-and-ticketing-connectors.md.

create table ioc_indicators (
  id          bigserial primary key,
  type        text not null check (type in ('ipv4','ipv6','domain','url','hash','email')),
  value       text not null,
  verdict     text not null check (verdict in ('malicious','suspicious','benign')),
  reliability text not null check (reliability in ('a','b','c','d','e','f')),
  labels      text[] not null default '{}',
  unique (type, value)
);

-- Deterministic lab data: RFC 5737/6761/3849 test ranges only, never real IOCs.
insert into ioc_indicators (type, value, verdict, reliability, labels) values
  ('ipv4', '198.51.100.10', 'malicious', 'b', '{"known-scanner"}'),
  ('ipv4', '203.0.113.55', 'suspicious', 'c', '{"tor-exit-lab"}'),
  ('domain', 'malicious-lab.test', 'malicious', 'a', '{"phishing"}'),
  ('domain', 'benign-lab.test', 'benign', 'a', '{}');

create table enrichment_results (
  id                 bigserial primary key,
  tenant_id          text not null,
  observable_id      bigint not null references observables(id) on delete cascade,
  provider           text not null default 'mock-local-ti',
  verdict            text not null check (verdict in ('malicious','suspicious','benign','unknown')),
  confidence         numeric not null check (confidence >= 0 and confidence <= 1),
  source_reliability text,
  sighting_time      timestamptz,
  expiration         timestamptz,
  raw_reference      text not null,
  queried_at         timestamptz not null default now(),
  unique (observable_id, provider)
);

create table tickets (
  id             text primary key,
  tenant_id      text not null,
  case_id        text not null references cases(id) on delete cascade,
  provider       text not null default 'mock-ticketing',
  external_ref   text not null,
  status         text not null check (status in ('open','closed')) default 'open',
  summary        text not null,
  evidence_refs  text[] not null default '{}',
  created_at     timestamptz not null default now(),
  closed_at      timestamptz,
  closure_reason text,
  unique (tenant_id, case_id)
);
