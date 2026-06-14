# Phylax

Phylax is a server-side mobile app attestation service.

It verifies proofs from Android and Apple clients, then issues short-lived
Phylax session tokens that backend APIs can require on sensitive routes.

Current scope:

- Standalone HTTP service
- Environment-based configuration
- Multi-app registry
- Provider readiness checks
- Challenge issuance
- Mock Android and Apple verification for local development
- Signed Phylax token issuance
- Token introspection endpoint
- Standard email/password auth with database-backed bearer tokens
- Redis-backed email verification OTPs

Production provider verification is intentionally isolated behind provider
interfaces so Google Play Integrity, Firebase App Check, and Apple App Attest
can be added without changing the API surface.

## Environments

Phylax uses a flavour-based setup:

```text
development -> local service + Postgres via DATABASE_URL
staging     -> Render + Postgres via DATABASE_URL
production  -> Render + Postgres via DATABASE_URL
```

Configuration is loaded from `.env` locally and environment variables in
deployment. Start from the example file:

```sh
cp .env.example .env
```

Required values are validated on startup:

```text
PHYLAX_ENV
DATABASE_URL
PHYLAX_TOKEN_SECRET
PHYLAX_CREDENTIAL_ENCRYPTION_KEY
```

When `MAIL_DRIVER=smtp`, `SMTP_HOST`, `SMTP_PORT`, and `MAIL_FROM_EMAIL` are
also required. `SMTP_PASS` is optional.

## Run Locally

```sh
cp .env.example .env
make run
```

## Quality Gates

```sh
make fmt
make test
make coverage
make build
```

The project is structured so every provider, store, handler, and policy can be
tested independently. Coverage starts with the core flow and should be ratcheted
toward 100% as real Google, Apple, Firebase, and adapter implementations land.

## Package Stack

Phylax uses standard Go packages instead of framework-specific magic:

```text
HTTP router:      chi
ORM/query layer:  gorm
Migrations:       golang-migrate
Validation:       go-playground/validator
Config/env:       godotenv + typed validation
OpenAPI:          swaggo annotations + generated docs
DI:               manual constructors
Testing:          standard testing + testify
```

## Project Structure

Application-facing code follows the NexBox-style layered layout:

```text
internal/app/
  controllers/
  exceptions/
  listeners/
  middleware/
  models/
  routes/
  services/
  types/
  validators/
```

Domain grouping lives inside each layer:

```text
internal/app/models/identity/
internal/app/models/projects/
internal/app/services/identity/
internal/app/services/projects/
internal/app/types/dto/identity/
internal/app/types/dto/projects/
internal/app/validators/identity/
internal/app/validators/projects/
```

Current model foundations:

```text
identity -> users, roles, user_roles, user_identities, auth_access_tokens
projects -> projects, project_members, project_apps
```

User emails are intentionally not unique at the database layer. Standard
email/password signups are currently guarded in the service layer. Users support
soft deletes through `deleted_at`.

Standard auth uses:

```text
POST /v1/auth/accounts      -> create account and send email OTP
POST /v1/auth/verify-email -> verify OTP and receive bearer token
POST /v1/auth/login        -> login with email/password
POST /v1/auth/logout       -> revoke current bearer token
```

Bearer tokens are returned once to the caller. Phylax stores only the token hash
in `auth_access_tokens`, so tokens are stateless on each HTTP request but still
revocable in the database.

Email verification OTPs are stored in Redis for 5 minutes when `REDIS_URL` is
configured. Local development falls back to an in-memory OTP store. Local mail
uses the `log` driver; staging and production should use SMTP.

Project apps are created inside projects and currently support:

```text
android -> package_name + Play Integrity/Firebase App Check
ios     -> bundle_id + App Attest/Firebase App Check
```

Project apps are always nested under projects:

```text
POST   /v1/projects
GET    /v1/projects
GET    /v1/projects/{project_id}
PATCH  /v1/projects/{project_id}
DELETE /v1/projects/{project_id}

POST   /v1/projects/{project_id}/apps
GET    /v1/projects/{project_id}/apps
GET    /v1/projects/{project_id}/apps/{app_id}
PATCH  /v1/projects/{project_id}/apps/{app_id}
DELETE /v1/projects/{project_id}/apps/{app_id}
```

Project routes require:

```http
Authorization: Bearer USER_TOKEN
```

## Endpoints

```text
GET  /v1/health
GET  /swagger/index.html
GET  /swagger/doc.json
POST /v1/auth/accounts
POST /v1/auth/verify-email
POST /v1/auth/login
POST /v1/auth/logout
POST /v1/apps/{app_id}/challenges
POST /v1/apps/{app_id}/attestations/verify
GET  /v1/apps/{app_id}/setup-checks
POST /v1/apps/{app_id}/credentials
POST /v1/sessions/introspect
POST /v1/projects
GET  /v1/projects
GET  /v1/projects/{project_id}
PATCH /v1/projects/{project_id}
DELETE /v1/projects/{project_id}
```

## Persistence

Storage is configured through `PHYLAX_STORAGE_DRIVER`.

Runtime uses Postgres through `DATABASE_URL`. The in-memory store remains
available for tests and explicit local experiments, but `DATABASE_URL` is still
validated as a required environment variable.

Uploaded provider credentials are encrypted before storage. Set a strong
`PHYLAX_CREDENTIAL_ENCRYPTION_KEY` value, at least 32 characters, before using
`POST /v1/apps/{app_id}/credentials`.

Postgres schema files live in:

```text
database/migrations
```

Run migrations with:

```sh
make migrate-up
make migrate-down
```

Run seeders with:

```sh
make seed
```

The development seeder creates `customer@phylax.local` with password
`password`, already email-verified.

Seed SQL files live in:

```text
database/seeders
```

## Swagger Docs

Swagger documentation is generated from Go annotations.

```sh
make swagger
```

The generated files live in:

```text
docs/
```

## Integration Shape

Customer backends can use Phylax as an internal service:

```text
Mobile app -> Customer API -> Phylax -> Google/Apple/Firebase
```

Protected requests carry:

```http
Authorization: Bearer USER_TOKEN
X-Phylax-Token: PHYLAX_SESSION_TOKEN
```
