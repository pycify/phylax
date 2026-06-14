package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateLimitBucket struct {
	limit  int
	window time.Duration
}

type rateLimitRule struct {
	key    string
	scope  string
	bucket rateLimitBucket
}

type rateLimitDecision struct {
	allowed    bool
	scope      string
	limit      int
	remaining  int
	retryAfter int
	resetIn    int
}

type fixedWindowState struct {
	windowStart time.Time
	count       int
	lastSeen    time.Time
}

type fixedWindowLimiter struct {
	mu     sync.Mutex
	states map[string]fixedWindowState
}

func newFixedWindowLimiter() *fixedWindowLimiter {
	return &fixedWindowLimiter{states: map[string]fixedWindowState{}}
}

func (l *fixedWindowLimiter) allow(now time.Time, rule rateLimitRule) rateLimitDecision {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanup(now)

	state, ok := l.states[rule.key]
	if !ok || now.Sub(state.windowStart) >= rule.bucket.window {
		state = fixedWindowState{windowStart: now, count: 0, lastSeen: now}
	}

	state.count++
	state.lastSeen = now
	l.states[rule.key] = state

	windowEndsAt := state.windowStart.Add(rule.bucket.window)
	resetIn := int(math.Ceil(windowEndsAt.Sub(now).Seconds()))
	if resetIn < 0 {
		resetIn = 0
	}

	remaining := rule.bucket.limit - state.count
	if remaining < 0 {
		remaining = 0
	}

	decision := rateLimitDecision{
		allowed:    state.count <= rule.bucket.limit,
		scope:      rule.scope,
		limit:      rule.bucket.limit,
		remaining:  remaining,
		retryAfter: resetIn,
		resetIn:    resetIn,
	}
	return decision
}

func (l *fixedWindowLimiter) cleanup(now time.Time) {
	for key, state := range l.states {
		if now.Sub(state.lastSeen) > 15*time.Minute {
			delete(l.states, key)
		}
	}
}

var attestationLimiter = newFixedWindowLimiter()

// AttestationRateLimit protects attestation endpoints against burst abuse.
// Scopes: shared IP, route+IP, and app+install.
func AttestationRateLimit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			now := time.Now().UTC()
			clientIP := requestIP(req)
			appID := strings.TrimSpace(strings.ToLower(pathParam(req.URL.Path, 2)))
			installID := extractInstallID(req)
			routeKey := strings.ToLower(req.Method) + ":" + normalizePath(req.URL.Path)

			rules := []rateLimitRule{
				{
					key:    "rl:attestation:shared:ip:" + clientIP,
					scope:  "shared-ip",
					bucket: rateLimitBucket{limit: 240, window: time.Minute},
				},
				{
					key:    "rl:attestation:route:" + routeKey + ":ip:" + clientIP,
					scope:  "route-ip",
					bucket: rateLimitBucket{limit: 40, window: time.Minute},
				},
			}

			if appID != "" && installID != "" {
				rules = append(rules, rateLimitRule{
					key:    "rl:attestation:install:app:" + appID + ":install:" + installID,
					scope:  "app-install",
					bucket: rateLimitBucket{limit: 120, window: 5 * time.Minute},
				})
			}

			var selected rateLimitDecision
			for i, rule := range rules {
				decision := attestationLimiter.allow(now, rule)
				if i == 0 || decision.remaining < selected.remaining {
					selected = decision
				}
				if !decision.allowed {
					writeRateLimitHeaders(w, decision)
					writeRateLimitError(w, "Too many requests. Please try again later.")
					return
				}
			}

			writeRateLimitHeaders(w, selected)
			next.ServeHTTP(w, req)
		})
	}
}

func writeRateLimitHeaders(w http.ResponseWriter, decision rateLimitDecision) {
	if decision.retryAfter > 0 {
		w.Header().Set("Retry-After", itoa(decision.retryAfter))
	}
	w.Header().Set("X-RateLimit-Limit", itoa(decision.limit))
	w.Header().Set("X-RateLimit-Remaining", itoa(decision.remaining))
	w.Header().Set("X-RateLimit-Reset", itoa(decision.resetIn))
	w.Header().Set("X-RateLimit-Scope", decision.scope)
}

func extractInstallID(req *http.Request) string {
	if req.Body == nil {
		return ""
	}
	body, _ := io.ReadAll(io.LimitReader(req.Body, 64*1024))
	if len(body) == 0 {
		return ""
	}

	req.Body = io.NopCloser(bytes.NewReader(body))

	var payload struct {
		InstallID string `json:"install_id"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(payload.InstallID))
}

func normalizePath(path string) string {
	clean := strings.ToLower(path)
	clean = strings.ReplaceAll(clean, "//", "/")
	parts := strings.Split(clean, "/")
	if len(parts) >= 4 {
		parts[3] = ":app_id"
	}
	return strings.Join(parts, "/")
}

func pathParam(path string, index int) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if index < 0 || index >= len(parts) {
		return ""
	}
	return parts[index]
}

func requestIP(req *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(req.Header.Get(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			value = strings.TrimSpace(strings.Split(value, ",")[0])
		}
		if value != "" {
			return strings.ToLower(value)
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(req.RemoteAddr))
	if err == nil && host != "" {
		return strings.ToLower(host)
	}
	if req.RemoteAddr != "" {
		return strings.ToLower(strings.TrimSpace(req.RemoteAddr))
	}
	return "unknown"
}

func itoa(value int) string { return strconv.Itoa(value) }

func writeRateLimitError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}{
		Error:   true,
		Message: message,
	})
}
