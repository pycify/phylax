# Phylax Roadmap

## Product Goal

Phylax is a developer tool for mobile app attestation.

The developer experience should be:

A developer creates a project and app in Phylax.

The developer configures Play Integrity, App Attest, or Firebase App Check credentials.

The mobile SDK attests the running app and receives a short-lived Phylax session token.

The mobile app sends that Phylax token to the developer's backend with protected API requests.

The developer's backend uses a Phylax server SDK or middleware to validate the token before serving sensitive routes.

The key product boundary:

```text
Mobile SDK -> Phylax service -> Phylax session token -> Customer backend SDK/middleware
```

The mobile SDK proves the app instance is valid. The backend SDK makes that proof easy to enforce on customer APIs.

## Current State

Working:

- [x] Project creation
- [x] Android app creation with Play Integrity provider
- [x] Play Integrity setup and credential flow
- [x] Flutter client can initialize Phylax
- [x] Flutter client can call attest
- [x] Phylax can create a challenge
- [x] Phylax can verify Play Integrity attestation
- [x] Phylax can issue a session token
- [x] Phylax can introspect a session token

Missing for the full developer story:

- [x] Backend service-to-service authentication
- [x] Official backend SDK/middleware
- [x] Clear protected-route integration docs
- [x] Production-grade token lifecycle guidance for mobile clients
- [ ] iOS/App Attest real verification
- [ ] Firebase App Check real verification

## Phase 0 - Harden The Current Attestation Contract

Before building SDKs around the API, make sure the core contract is stable.

- [ ] Confirm the exact header name for protected requests: `X-Phylax-Token`
- [ ] Confirm session token TTL defaults and renewal behavior
- [ ] Confirm whether session tokens are app-scoped, project-scoped, or both
- [ ] Confirm introspection response fields are final enough for SDK consumption
- [ ] Add tests around expired sessions, wrong app IDs, and missing/invalid Phylax tokens
- [ ] Document the end-to-end mobile flow:
  - [ ] create challenge
  - [ ] request provider token on device
  - [ ] verify attestation
  - [ ] store Phylax session token
  - [ ] attach token to backend API calls

Deliverable: the attestation API is stable enough for SDKs.

## Phase 1 - Project API Keys

Backend developers should not authenticate to Phylax as dashboard users.

Add project-scoped API keys for service-to-service access.

- [x] DB migration: `project_api_keys`
- [x] Store only hashed API keys
- [x] Automatically issue the first API key when a project is created
- [x] Show the plaintext API key only once on project creation or rotation
- [x] Allow only one active API key per project
- [x] `GET /v1/projects/{project_id}/api-key` to view active key metadata
- [x] `POST /v1/projects/{project_id}/api-key/rotate` to revoke the old key and issue a new one
- [x] `DELETE /v1/projects/{project_id}/api-key` to revoke the active key
- [x] API key middleware for service endpoints
- [x] Require API keys on `POST /v1/sessions/introspect`

Suggested auth header:

```http
Authorization: Bearer phylax_project_api_key
```

Deliverable: customer backends can securely call Phylax without user auth.

## Phase 2 - `phylax-go` Backend SDK

This is the primary backend integration path.

Spec: `1.md`

Target module:

```text
github.com/pycify/phylax-go
```

The SDK should let backend developers write:

```go
client, err := phylax.New(phylax.Config{
    BaseURL:   "https://api.phylax.dev",
    ProjectID: "project_123",
    AppID:     "app_123",
    APIKey:    os.Getenv("PHYLAX_API_KEY"),
})

r.Use(client.Middleware(phylax.MiddlewareConfig{
    Required: true,
}))
```

The SDK should primarily call:

```text
POST /v1/sessions/introspect
```

It should:

- Read `X-Phylax-Token`
- Introspect the token with Phylax
- Reject missing tokens when required
- Reject inactive tokens
- Attach the active attestation session to request context
- Avoid logging API keys or Phylax tokens

Deliverable: Go backend developers can protect routes with one middleware.

## Phase 3 - Backend Integration Docs

Developers need a short path from "I installed Phylax" to "my API is protected."

Write docs for:

- [x] What the mobile app sends to the backend
- [x] What the backend validates
- [x] How to create a project API key
- [x] How to install and configure `phylax-go`
- [x] How to protect one route
- [x] How to access the attestation session in a handler
- [x] Recommended failure behavior for missing/inactive tokens

Minimum docs should include:

```http
X-Phylax-Token: PHYLAX_SESSION_TOKEN
```

and a complete backend example.

Deliverable: a developer can follow docs without reading Phylax internals.

Reference implementation and examples are in:

- [docs/protected-route-integration.md](docs/protected-route-integration.md)

## Phase 4 - Flutter SDK Productization

You have already proven the Flutter flow works. Now turn that into the official mobile developer experience.

- Clean initialization API
- `attest()` or `getToken()` public method
- Token caching
- Token refresh before expiry
- Clear error types
- Retry policy for network errors
- No token logging
- Android Play Integrity setup guide
- Example Flutter app
- Optional HTTP interceptor that adds `X-Phylax-Token`

Deliverable: Flutter developers can add Phylax to a client app without manually wiring challenge and verify calls.

Reference guidance is in:

- [docs/mobile-token-lifecycle-guidance.md](docs/mobile-token-lifecycle-guidance.md)

## Phase 5 - Provider Completeness

Android Play Integrity works first, but the product promise includes iOS and Firebase-backed apps.

- Apple App Attest real verification
  - verify attestation object
  - verify assertion
  - validate nonce/challenge
  - validate app/team identity
- Firebase App Check real verification
  - verify token
  - validate project/app identity
  - map Firebase claims into Phylax verdict/session

Deliverable: production support for iOS and Firebase App Check.

## Recommended Next Step

Start with Phase 1: Project API Keys.

Reason: your mobile attestation flow already works, but customer backends need a secure way to call Phylax introspection. Without API keys, the backend SDK cannot be production-ready.

After API keys:

1. Build `phylax-go`.
2. Document backend route protection.
3. Productize the Flutter SDK around the working attestation flow.
