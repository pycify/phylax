create table if not exists app_credentials (
  id uuid primary key default gen_random_uuid(),
  app_id uuid not null references project_apps (id) on delete cascade,
  provider text not null,
  credential_type text not null,
  encrypted_secret text not null,
  secret_nonce text not null,
  secret_sha256 text not null,
  active boolean not null default true,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  deleted_at timestamptz,
  constraint app_credentials_provider_check check (provider in ('play_integrity', 'app_attest', 'firebase_app_check')),
  constraint app_credentials_type_check check (credential_type in ('google_service_account', 'firebase_service_account', 'apple_team'))
);

create unique index if not exists app_credentials_active_idx
  on app_credentials (app_id, provider, credential_type)
  where active = true and deleted_at is null;

create index if not exists app_credentials_app_id_idx on app_credentials (app_id);
create index if not exists app_credentials_deleted_at_idx on app_credentials (deleted_at);
