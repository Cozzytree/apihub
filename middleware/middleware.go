package middleware

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("%s %s %d [%d bytes] in %v",
			r.Method,
			r.URL.Path,
			rw.statusCode,
			rw.size,
			duration,
		)
	})
}

type client struct {
	lastRequest time.Time
	count       uint
}

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*client
	Window  time.Duration
	Limit   uint
}

func NewRateLimiter(window time.Duration, limit uint) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*client),
		Window:  window,
		Limit:   limit,
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Can be a comma-separated list — use the first IP
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// Fallback: use remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func (rl *RateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Origin")
		if key == "" {
			key = getClientIP(r)
		}
		now := time.Now()
		rl.mu.Lock()

		c, ok := rl.clients[key]
		if !ok || now.Sub(c.lastRequest) > rl.Window {
			rl.clients[key] = &client{
				count:       1,
				lastRequest: now,
			}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}

		if c.count >= rl.Limit {
			rl.mu.Unlock()
			http.Error(w, "429 - Too Many Requests", http.StatusTooManyRequests)
			return
		}

		c.count++
		c.lastRequest = now
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
