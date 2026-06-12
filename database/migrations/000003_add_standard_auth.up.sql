alter table users
  add column if not exists password_hash text not null default '',
  add column if not exists email_verified_at timestamptz;

create table if not exists user_identities (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users (id) on delete cascade,
  provider text not null,
  provider_user_id text not null,
  email text not null default '',
  email_verified boolean not null default false,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  deleted_at timestamptz,
  constraint user_identities_provider_check check (provider in ('standard', 'google', 'github', 'apple'))
);

create unique index if not exists user_identities_provider_user_id_idx
  on user_identities (provider, provider_user_id)
  where deleted_at is null;

create index if not exists user_identities_user_id_idx on user_identities (user_id);
create index if not exists user_identities_deleted_at_idx on user_identities (deleted_at);

create table if not exists auth_access_tokens (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users (id) on delete cascade,
  name text not null default '',
  hash text not null unique,
  abilities jsonb not null default '[]',
  created_at timestamptz not null,
  updated_at timestamptz not null,
  last_used_at timestamptz,
  expires_at timestamptz,
  revoked_at timestamptz
);

create index if not exists auth_access_tokens_user_id_idx on auth_access_tokens (user_id);
create index if not exists auth_access_tokens_revoked_at_idx on auth_access_tokens (revoked_at);
create index if not exists auth_access_tokens_expires_at_idx on auth_access_tokens (expires_at);
