# Protected Route Integration Guide

This guide is the shortest production path from mobile attestation to backend route protection.

## 1. What the mobile app sends

After successful attestation, the mobile app receives a Phylax session token.
For protected API calls to the customer backend, include:

```http
X-Phylax-Token: PHYLAX_SESSION_TOKEN
```

The app can also send its normal user auth token if the backend route requires user identity:

```http
Authorization: Bearer USER_TOKEN
```

## 2. What the backend validates

For each protected request, the backend must:

1. Read X-Phylax-Token.
2. Call Phylax introspection endpoint.
3. Reject if token is missing (when route requires attestation).
4. Reject if token is inactive.
5. Accept when token is active and attach attestation session to request context.

Introspection call:

```text
POST /v1/sessions/introspect
```

Service-to-service auth to Phylax:

```http
Authorization: Bearer PHYLAX_API_KEY
```

## 3. How to create and manage project API keys

Project API key lifecycle endpoints:

```text
GET    /v1/projects/{project_id}/api-key
POST   /v1/projects/{project_id}/api-key/rotate
DELETE /v1/projects/{project_id}/api-key
```

Rules:

1. The first key is issued automatically when project is created.
2. Plaintext key is shown only once (creation or rotation).
3. Store it securely in backend secrets as PHYLAX_API_KEY.
4. Only one active API key exists per project.

## 4. Backend middleware usage with phylax-go

Install and configure client:

```go
phylaxClient, err := phylax.New(phylax.Config{
    BaseURL:   "https://api.phylax.dev",
    ProjectID: "project_123",
    AppID:     "app_123",
    APIKey:    os.Getenv("PHYLAX_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}
```

Protect one route:

```go
protected := phylaxClient.Middleware(phylax.MiddlewareConfig{
    Required: true,
})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    session, ok := phylax.SessionFromContext(r.Context())
    if !ok || session == nil {
        w.WriteHeader(http.StatusForbidden)
        return
    }

    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("trusted request"))
}))

http.Handle("/sensitive", protected)
```

## 5. Recommended failure behavior

1. Missing X-Phylax-Token on required route: return 401.
2. Invalid or inactive Phylax token: return 403.
3. Phylax unavailable or timeout: return 503 for strictly protected routes.
4. Never log raw PHYLAX_API_KEY.
5. Never log raw X-Phylax-Token.

## 6. End-to-end request shape

Client to customer backend:

```http
GET /v1/sensitive-resource
Authorization: Bearer USER_TOKEN
X-Phylax-Token: PHYLAX_SESSION_TOKEN
```

Customer backend to Phylax:

```http
POST /v1/sessions/introspect
Authorization: Bearer PHYLAX_API_KEY
Content-Type: application/json

{"token":"PHYLAX_SESSION_TOKEN"}
```
