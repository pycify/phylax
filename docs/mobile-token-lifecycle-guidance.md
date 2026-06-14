# Mobile Token Lifecycle Guidance

This guide describes production-safe lifecycle handling for Phylax session tokens in mobile clients.

## Objectives

- Keep protected API calls reliable without unnecessary attestation calls.
- Refresh before expiry to avoid user-facing failures.
- Avoid token leakage in logs, analytics, and crash reports.
- Handle offline and transient failures gracefully.

## Token Model

A Phylax session token is short-lived and should be treated as an in-memory credential with secure persistence fallback.

Recommended assumptions:

- The token can expire at any time close to ExpiresAt on device clocks with skew.
- The backend is the source of truth through introspection.
- The client should not trust local expiry checks alone for security decisions.

## Storage Strategy

Use secure platform storage.

- iOS: Keychain
- Android: EncryptedSharedPreferences or Keystore-backed storage

Do not store in plain local preferences.
Do not write the token to app logs.

Store:

- token
- expires_at
- issued_at
- app_id (optional for diagnostics)

## Lifecycle States

Maintain a simple state machine:

1. Empty: no token available.
2. Valid: token exists and has enough remaining lifetime.
3. Refreshing: refresh is in progress; callers wait for result.
4. Expired: token is unusable and requires re-attestation.
5. Invalid: backend rejected token; force re-attestation.

## Refresh Policy

Refresh proactively before expiry.

Recommended refresh window:

- refresh_at = expires_at - max(60s, 20% of token_ttl)

Examples:

- 15-minute token: refresh about 3 minutes before expiry.
- 5-minute token: refresh about 1 minute before expiry.

Apply jitter for large fleets:

- subtract an extra random 5 to 20 seconds to avoid synchronized refresh spikes.

## Request-Time Behavior

Before protected backend calls:

1. Check current token state.
2. If valid and not in refresh window, attach X-Phylax-Token and proceed.
3. If in refresh window, trigger a single refresh flow and wait.
4. If expired or missing, run full attestation flow.

Header to backend:

```http
X-Phylax-Token: PHYLAX_SESSION_TOKEN
```

## Single-Flight Refresh

Prevent parallel refresh storms in app process.

- Use one in-flight refresh future/promise.
- Additional requests await that same refresh result.
- On success, all queued requests continue with the new token.
- On failure, fail fast with typed error and let caller retry policy decide.

## Retry Policy

Use bounded exponential backoff for network and 5xx errors.

Recommended:

- max attempts: 3
- backoff: 300ms, 900ms, 1800ms
- add jitter

Do not retry on deterministic 4xx validation errors from Phylax verify endpoints.

## Invalidation Rules

Immediately invalidate local token and force re-attestation when:

- backend returns inactive token decision
- backend returns explicit attestation-required response
- app identity/config changed (package/bundle/environment)
- user signs out and app policy ties attestation to user session

## Offline Behavior

When offline:

- If token is still within local validity window, allow attempts and let backend enforce final policy.
- If expired and refresh/attest cannot run, return a typed temporary-unavailable error to caller.

Avoid silent fallback that bypasses protected-route requirements.

## Security and Privacy

- Never log full token values.
- Redact token-like strings in error reporting.
- Do not include token in analytics events.
- Clear token on app data reset, uninstall/reinstall flows, and security-sensitive account actions.

## Observability

Track lifecycle metrics without sensitive data:

- token_refresh_attempt
- token_refresh_success
- token_refresh_failure
- attest_attempt
- attest_success
- attest_failure
- protected_request_with_token
- protected_request_without_token

Include reason codes, not token values.

## Recommended Error Surface

Expose typed errors for SDK consumers:

- token_missing
- token_expired
- token_refresh_failed
- attest_failed
- backend_rejected_token
- temporary_unavailable

These should map cleanly to UI and retry behavior.

## Reference Request Flows

### Happy Path

1. Attest and receive token.
2. Cache token securely.
3. Attach token on protected requests.
4. Refresh before expiry.

### Rejection Path

1. Backend marks token inactive.
2. Client invalidates cache.
3. Client runs full attestation.
4. Client retries request with new token.

## Flutter-Specific Notes

- Wrap protected HTTP client with an interceptor that injects X-Phylax-Token.
- Keep token manager as a singleton service.
- Use a mutex/single-flight guard around refresh/attest.
- Keep storage and crypto handling in platform-secure libraries.
