drop table if exists auth_access_tokens;
drop table if exists user_identities;

alter table users
  drop column if exists email_verified_at,
  drop column if exists password_hash;

