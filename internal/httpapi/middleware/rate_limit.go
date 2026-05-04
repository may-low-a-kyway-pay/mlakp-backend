package middleware

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"mlakp-backend/internal/httpapi/response"
)

type RateLimiter struct {
	mu          sync.Mutex
	maxRequests int
	window      time.Duration
	now         func() time.Time
	entries     map[string]rateLimitEntry
}

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	if window <= 0 {
		window = time.Minute
	}

	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		now:         time.Now,
		entries:     make(map[string]rateLimitEntry),
	}
}

func (l *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l == nil || l.allow(rateLimitKey(r)) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Retry-After", strconv.Itoa(int(l.window.Seconds())))
		response.Error(w, http.StatusTooManyRequests, "rate_limited", "Too many requests. Please retry later")
	})
}

func (l *RateLimiter) allow(key string) bool {
	if l.maxRequests <= 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.cleanupExpired(now)

	entry, ok := l.entries[key]
	if !ok || !now.Before(entry.resetAt) {
		l.entries[key] = rateLimitEntry{
			count:   1,
			resetAt: now.Add(l.window),
		}
		return true
	}
	if entry.count >= l.maxRequests {
		return false
	}

	entry.count++
	l.entries[key] = entry
	return true
}

func (l *RateLimiter) cleanupExpired(now time.Time) {
	for key, entry := range l.entries {
		if !now.Before(entry.resetAt) {
			delete(l.entries, key)
		}
	}
}

func rateLimitKey(r *http.Request) string {
	host := r.RemoteAddr
	if parsedHost, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = parsedHost
	}

	return r.Method + " " + r.URL.Path + " " + host
}
