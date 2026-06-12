create table if not exists phylax_challenges (
  id text primary key,
  app_id text not null,
  platform text not null,
  action text not null,
  install_id text not null,
  nonce text not null,
  expires_at timestamptz not null,
  created_at timestamptz not null,
  consumed_at timestamptz
);

create index if not exists phylax_challenges_app_install_idx
  on phylax_challenges (app_id, install_id);

create index if not exists phylax_challenges_expires_at_idx
  on phylax_challenges (expires_at);

create table if not exists phylax_sessions (
  id text primary key,
  app_id text not null,
  platform text not null,
  action text not null,
  install_id text not null,
  subject text,
  risk_level text not null,
  issued_at timestamptz not null,
  expires_at timestamptz not null,
  revoked_at timestamptz
);

create index if not exists phylax_sessions_app_install_idx
  on phylax_sessions (app_id, install_id);

create index if not exists phylax_sessions_expires_at_idx
  on phylax_sessions (expires_at);

create table if not exists phylax_apple_keys (
  key_id text primary key,
  app_id text not null,
  install_id text not null,
  public_key_pem text not null,
  sign_count bigint not null default 0,
  created_at timestamptz not null,
  last_used_at timestamptz
);

create index if not exists phylax_apple_keys_app_install_idx
  on phylax_apple_keys (app_id, install_id);

create table if not exists phylax_verdict_logs (
  id bigserial primary key,
  app_id text not null,
  platform text not null,
  provider text not null,
  install_id text not null,
  action text not null,
  risk_level text not null,
  claims jsonb not null default '{}',
  created_at timestamptz not null
);

create index if not exists phylax_verdict_logs_app_created_idx
  on phylax_verdict_logs (app_id, created_at desc);

