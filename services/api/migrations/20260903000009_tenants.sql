-- adr/0022: a lab-appropriate tenant *selector*, not real auth/RBAC (that
-- stays Keycloak/OIDC per adr/0005, not started). api_key_hash is nullable
-- because the seeded "lab" tenant is the implicit default (services/api's
-- TenantKeyMiddleware falls back to it when no X-Tenant-Key header is
-- sent) and was never issued a key of its own.
create table tenants (
  id            text primary key,
  name          text not null,
  api_key_hash  text unique,
  created_at    timestamptz not null default now()
);

insert into tenants (id, name, api_key_hash, created_at)
values ('lab', 'Lab (default)', null, now());
