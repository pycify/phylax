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

Production provider verification is intentionally isolated behind provider
interfaces so Google Play Integrity, Firebase App Check, and Apple App Attest
can be added without changing the API surface.

## Run

```sh
cp phylax.example.json phylax.json
go run ./cmd/phylax -config phylax.json
```

## Endpoints

```text
GET  /v1/health
GET  /v1/apps
GET  /v1/apps/{app_id}/providers
POST /v1/challenges
POST /v1/attestations/verify
POST /v1/sessions/introspect
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

