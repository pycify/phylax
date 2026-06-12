create extension if not exists "pgcrypto";

create table if not exists users (
  id uuid primary key default gen_random_uuid(),
  email text not null,
  full_name text not null default '',
  user_type text not null,
  status text not null default 'active',
  created_at timestamptz not null,
  updated_at timestamptz not null,
  deleted_at timestamptz,
  constraint users_user_type_check check (user_type in ('customer', 'admin')),
  constraint users_status_check check (status in ('active', 'suspended'))
);

create index if not exists users_email_idx on users (email);
create index if not exists users_user_type_idx on users (user_type);
create index if not exists users_deleted_at_idx on users (deleted_at);

create table if not exists roles (
  id uuid primary key default gen_random_uuid(),
  name text not null unique,
  description text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null
);

insert into roles (id, name, description, created_at, updated_at)
values
  (gen_random_uuid(), 'super_admin', 'Full platform administration access', now(), now()),
  (gen_random_uuid(), 'admin', 'Platform administration access', now(), now()),
  (gen_random_uuid(), 'customer_service', 'Customer support access', now(), now())
on conflict (name) do nothing;

create table if not exists user_roles (
  user_id uuid not null references users (id) on delete cascade,
  role_id uuid not null references roles (id) on delete cascade,
  created_at timestamptz not null,
  primary key (user_id, role_id)
);

create index if not exists user_roles_role_id_idx on user_roles (role_id);

create table if not exists projects (
  id uuid primary key default gen_random_uuid(),
  owner_user_id uuid not null references users (id),
  name text not null,
  slug text not null unique,
  description text not null default '',
  created_at timestamptz not null,
  updated_at timestamptz not null,
  deleted_at timestamptz
);

create index if not exists projects_owner_user_id_idx on projects (owner_user_id);
create index if not exists projects_deleted_at_idx on projects (deleted_at);

create table if not exists project_members (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects (id) on delete cascade,
  user_id uuid not null references users (id) on delete cascade,
  role text not null,
  status text not null,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  deleted_at timestamptz,
  constraint project_members_role_check check (role in ('owner', 'admin', 'member', 'viewer')),
  constraint project_members_status_check check (status in ('active', 'invited', 'revoked'))
);

create unique index if not exists project_members_project_user_active_idx
  on project_members (project_id, user_id)
  where deleted_at is null;

create index if not exists project_members_user_id_idx on project_members (user_id);
create index if not exists project_members_deleted_at_idx on project_members (deleted_at);

create table if not exists project_apps (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects (id) on delete cascade,
  name text not null,
  platform text not null,
  provider text not null,
  environment text not null,
  package_name text not null default '',
  bundle_id text not null default '',
  enabled boolean not null default true,
  created_at timestamptz not null,
  updated_at timestamptz not null,
  deleted_at timestamptz,
  constraint project_apps_platform_check check (platform in ('android', 'ios')),
  constraint project_apps_provider_check check (provider in ('play_integrity', 'app_attest', 'firebase_app_check')),
  constraint project_apps_environment_check check (environment in ('development', 'staging', 'production')),
  constraint project_apps_identifier_check check (
    (platform = 'android' and package_name <> '' and bundle_id = '')
    or
    (platform = 'ios' and bundle_id <> '' and package_name = '')
  )
);

create unique index if not exists project_apps_android_identifier_idx
  on project_apps (project_id, environment, package_name)
  where platform = 'android' and deleted_at is null;

create unique index if not exists project_apps_ios_identifier_idx
  on project_apps (project_id, environment, bundle_id)
  where platform = 'ios' and deleted_at is null;

create index if not exists project_apps_project_id_idx on project_apps (project_id);
create index if not exists project_apps_deleted_at_idx on project_apps (deleted_at);
