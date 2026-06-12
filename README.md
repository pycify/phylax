# Phylax

Phylax is a server-side mobile app attestation service.

It verifies proofs from Android and Apple clients, then issues short-lived
Phylax session tokens that backend APIs can require on sensitive routes.

Current scope:

- Standalone HTTP service
- JSON configuration
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
development -> local service + local memory store by default
staging     -> Render + Postgres via DATABASE_URL
production  -> Render + Postgres via DATABASE_URL
```

Set the environment with:

```sh
PHYLAX_ENV=development
PHYLAX_ENV=staging
PHYLAX_ENV=production
```

The service automatically loads:

```text
configs/phylax.development.json
configs/phylax.staging.json
configs/phylax.production.json
```

You can override the path with:

```sh
PHYLAX_CONFIG=/path/to/phylax.json
```

Runtime secrets and deployment values belong in `.env` locally and Render
environment variables remotely. See `.env.example`.

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
Config/env:       viper + godotenv
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
POST /v1/challenges
POST /v1/attestations/verify
POST /v1/sessions/introspect
POST /v1/projects
GET  /v1/projects
GET  /v1/projects/{project_id}
PATCH /v1/projects/{project_id}
DELETE /v1/projects/{project_id}
```

## Persistence

Storage is configured through `storage.driver` or `PHYLAX_STORAGE_DRIVER`.

```text
memory   -> local development
postgres -> staging/production on Render
```

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
