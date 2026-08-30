-- M1 schema: events, alerts, cases and their join tables, plus the
-- idempotency-key table shared by every mutating M1 endpoint.
--
-- Applied via `atlas migrate apply --dir file://migrations --url "$DATABASE_URL"`
-- once atlas is installed, or directly with `psql "$DATABASE_URL" -f
-- migrations/20260830000001_initial_schema.sql` for local development.

create table events (
  id                text primary key,
  tenant_id         text not null,
  source            text not null,
  source_event_id   text,
  observed_at       timestamptz not null,
  received_at       timestamptz not null,
  event_class       text not null check (event_class in
                     ('authentication','network','process','file','email','vulnerability','cloud','malware','other')),
  severity          text not null check (severity in
                     ('informational','low','medium','high','critical')),
  confidence        numeric not null check (confidence >= 0 and confidence <= 1),
  raw_payload       jsonb not null,
  raw_ref_uri       text not null,
  raw_ref_sha256    text not null,
  labels            text[] not null default '{}',
  extensions        jsonb not null default '{}',
  schema_version    text not null default 'event-1.0',
  created_at        timestamptz not null default now()
);

create unique index events_tenant_source_event_uq
  on events (tenant_id, source, source_event_id)
  where source_event_id is not null;

create index events_tenant_id_idx on events (tenant_id, id desc);

create table observables (
  id             bigserial primary key,
  tenant_id      text not null,
  type           text not null check (type in
                  ('ipv4','ipv6','domain','url','hash','email','user','host','process','file','vulnerability')),
  value          text not null,
  algorithm      text,
  normalization  text not null,
  first_seen     timestamptz not null,
  last_seen      timestamptz not null,
  confidence     numeric not null check (confidence >= 0 and confidence <= 1),
  sensitivity    text not null check (sensitivity in ('public','internal','confidential','restricted')),
  labels         text[] not null default '{}',
  extensions     jsonb not null default '{}'
);

create table event_observables (
  event_id       text not null references events(id) on delete cascade,
  observable_id  bigint not null references observables(id) on delete cascade,
  primary key (event_id, observable_id)
);

create table observable_source_refs (
  observable_id  bigint not null references observables(id) on delete cascade,
  source_ref     text not null,
  primary key (observable_id, source_ref)
);

create table alerts (
  id                          text primary key,
  tenant_id                   text not null,
  rule_id                     text not null,
  rule_version                text not null,
  detection_source            text not null,
  status                      text not null check (status in
                               ('new','acknowledged','investigating','escalated','resolved','false_positive')),
  severity                    text not null check (severity in
                               ('informational','low','medium','high','critical')),
  confidence                  numeric not null check (confidence >= 0 and confidence <= 1),
  correlation_key             text,
  correlation_window_seconds  int,
  case_ref                    text,
  created_at                  timestamptz not null default now()
);

create table alert_events (
  alert_id  text not null references alerts(id) on delete cascade,
  event_id  text not null references events(id) on delete cascade,
  primary key (alert_id, event_id)
);

create table alert_observables (
  alert_id       text not null references alerts(id) on delete cascade,
  observable_id  bigint not null references observables(id) on delete cascade,
  primary key (alert_id, observable_id)
);

create table cases (
  id                    text primary key,
  tenant_id             text not null,
  title                 text not null check (char_length(title) between 1 and 512),
  category              text not null,
  severity              text not null check (severity in
                         ('informational','low','medium','high','critical')),
  confidence            numeric not null check (confidence >= 0 and confidence <= 1),
  status                text not null check (status in
                         ('new','triage','investigating','contained','eradication','recovery',
                          'escalated','dismissed','closed')),
  opened_at             timestamptz not null default now(),
  updated_at            timestamptz not null default now(),
  closed_at             timestamptz,
  owner                 text not null,
  incident_commander    text,
  service_owner         text,
  hypothesis            text,
  facts                 text[] not null default '{}',
  limitations           text[] not null default '{}',
  closure_reason        text,
  correlation_key       text,
  root_cause            text,
  lessons_learned       text
);

create index cases_tenant_id_idx on cases (tenant_id, id desc);

create table case_alerts (
  case_id   text not null references cases(id) on delete cascade,
  alert_id  text not null references alerts(id) on delete cascade,
  primary key (case_id, alert_id)
);

create table case_observables (
  case_id        text not null references cases(id) on delete cascade,
  observable_id  bigint not null references observables(id) on delete cascade,
  primary key (case_id, observable_id)
);

create table case_timeline_entries (
  id          bigserial primary key,
  case_id     text not null references cases(id) on delete cascade,
  tenant_id   text not null,
  occurred_at timestamptz not null default now(),
  kind        text not null,
  actor       text not null default 'system',
  detail      jsonb not null default '{}'
);

create index case_timeline_case_idx on case_timeline_entries (case_id, occurred_at);

create table idempotency_keys (
  tenant_id       text not null,
  http_method     text not null,
  route           text not null,
  idempotency_key text not null,
  request_hash    text not null,
  status_code     int not null,
  response_body   jsonb not null,
  created_at      timestamptz not null default now(),
  primary key (tenant_id, http_method, route, idempotency_key)
);
