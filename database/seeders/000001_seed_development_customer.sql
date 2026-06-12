with dev_user as (
  insert into users (
    id,
    email,
    full_name,
    password_hash,
    email_verified_at,
    user_type,
    status,
    created_at,
    updated_at
  )
  values (
    gen_random_uuid(),
    'customer@phylax.local',
    'Development Customer',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
    now(),
    'customer',
    'active',
    now(),
    now()
  )
  on conflict (email) do update set
    full_name = excluded.full_name,
    password_hash = excluded.password_hash,
    email_verified_at = excluded.email_verified_at,
    user_type = excluded.user_type,
    status = excluded.status,
    updated_at = now(),
    deleted_at = null
  returning id
)
insert into user_identities (
  id,
  user_id,
  provider,
  provider_user_id,
  email,
  email_verified,
  created_at,
  updated_at
)
select
  gen_random_uuid(),
  dev_user.id,
  'standard',
  'customer@phylax.local',
  'customer@phylax.local',
  true,
  now(),
  now()
from dev_user
on conflict (provider, provider_user_id) do update set
  user_id = excluded.user_id,
  email = excluded.email,
  email_verified = excluded.email_verified,
  updated_at = now(),
  deleted_at = null;
