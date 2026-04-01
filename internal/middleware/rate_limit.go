package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TokenBucket implements rate limiting using token bucket algorithm
type TokenBucket struct {
	maxTokens  int
	refillRate float64 // tokens per millisecond
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket with requests per minute
func NewTokenBucket(requestsPerMinute int) *TokenBucket {
	return &TokenBucket{
		maxTokens:  requestsPerMinute,
		refillRate: float64(requestsPerMinute) / (60 * 1000.0), // tokens per millisecond
		tokens:     float64(requestsPerMinute),
		lastRefill: time.Now(),
	}
}

// Take attempts to consume one token from the bucket
// Returns true if successful, false if rate limit exceeded
func (tb *TokenBucket) Take() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Milliseconds()
	tb.lastRefill = now

	// Refill tokens based on elapsed time
	tokensToAdd := tb.refillRate * float64(elapsed)
	tb.tokens = min(float64(tb.maxTokens), tb.tokens+tokensToAdd)

	if tb.tokens >= 1.0 {
		tb.tokens--
		return true
	}

	return false
}

// GetRemaining returns the number of remaining tokens
func (tb *TokenBucket) GetRemaining() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return int(tb.tokens)
}

// RateLimiter provides per-tenant rate limiting
type RateLimiter struct {
	defaultLimit int
	buckets      map[uuid.UUID]*TokenBucket
	mu           sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(defaultLimit int) *RateLimiter {
	return &RateLimiter{
		defaultLimit: defaultLimit,
		buckets:      make(map[uuid.UUID]*TokenBucket),
	}
}

// Allow checks if a request is allowed for a given tenant
func (rl *RateLimiter) Allow(tenantID uuid.UUID) bool {
	rl.mu.Lock()
	bucket, exists := rl.buckets[tenantID]
	if !exists {
		bucket = NewTokenBucket(rl.defaultLimit)
		rl.buckets[tenantID] = bucket
	}
	rl.mu.Unlock()

	return bucket.Take()
}

// GetRemaining returns the number of remaining requests for a tenant
func (rl *RateLimiter) GetRemaining(tenantID uuid.UUID) int {
	rl.mu.RLock()
	bucket, exists := rl.buckets[tenantID]
	rl.mu.RUnlock()

	if !exists {
		return rl.defaultLimit
	}

	return bucket.GetRemaining()
}

// RateLimitMiddleware enforces per-tenant rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := TenantIDFromContext(r.Context())

			// Skip rate limiting if tenant ID is not set (should not happen after auth)
			if tenantID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			if !limiter.Allow(tenantID) {
				w.Header().Set("X-RateLimit-Limit", "100")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", "60")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":{"code":"rate_limit_exceeded","message":"too many requests"}}`))
				return
			}

			remaining := limiter.GetRemaining(tenantID)
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", string(rune(remaining)))
			w.Header().Set("X-RateLimit-Reset", "60")

			next.ServeHTTP(w, r)
		})
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
