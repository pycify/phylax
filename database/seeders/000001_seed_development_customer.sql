do $$
declare
  dev_user_id uuid;
begin
  select id into dev_user_id
  from users
  where email = 'customer@phylax.local' and deleted_at is null;

  if dev_user_id is null then
    dev_user_id := gen_random_uuid();
    insert into users (
      id, email, full_name, password_hash, email_verified_at,
      user_type, status, created_at, updated_at
    )
    values (
      dev_user_id, 'customer@phylax.local', 'Development Customer',
      '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
      now(), 'customer', 'active', now(), now()
    );
  end if;

  if not exists (
    select 1 from user_identities
    where provider = 'standard' and provider_user_id = 'customer@phylax.local' and deleted_at is null
  ) then
    insert into user_identities (
      id, user_id, provider, provider_user_id,
      email, email_verified, created_at, updated_at
    )
    values (
      gen_random_uuid(), dev_user_id, 'standard', 'customer@phylax.local',
      'customer@phylax.local', true, now(), now()
    );
  end if;
end;
$$;
