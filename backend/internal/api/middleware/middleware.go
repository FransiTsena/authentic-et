package middleware

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/beecloud/paygate/internal/service"
	"github.com/google/uuid"
)

type contextKey string

const (
	RequestIDKey contextKey = "requestID"
	UserKey      contextKey = "user"
	APIKeyKey    contextKey = "apiKey"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SecurityHeaders sets hardening HTTP response headers
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			reqID, _ := r.Context().Value(RequestIDKey).(string)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"durationMs", duration.Milliseconds(),
				"ip", GetClientIP(r),
				"requestId", reqID,
			)
		})
	}
}

func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					reqID, _ := r.Context().Value(RequestIDKey).(string)
					logger.Error("panic recovered in http handler", "error", err, "requestId", reqID)
					http.Error(w, `{"code":"INTERNAL_SERVER_ERROR","message":"An unexpected error occurred"}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// In-memory rate limiter using sliding token bucket
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	tokens    float64
	maxTokens float64
	refillRPS float64
	lastCheck time.Time
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
	}
	// Background garbage collection of stale buckets
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) Allow(key string, maxTokens float64, refillRPS float64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &bucket{
			tokens:    maxTokens - 1,
			maxTokens: maxTokens,
			refillRPS: refillRPS,
			lastCheck: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens = b.tokens + elapsed*b.refillRPS
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastCheck = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-10 * time.Minute)
	for k, b := range rl.buckets {
		if b.lastCheck.Before(cutoff) {
			delete(rl.buckets, k)
		}
	}
}

func RequireAuth(cfg *config.Config, authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				token = strings.TrimSpace(authHeader[7:])
			} else if apiKeyHeader := r.Header.Get("X-API-KEY"); apiKeyHeader != "" {
				token = strings.TrimSpace(apiKeyHeader)
			}

			if token == "" {
				http.Error(w, `{"code":"UNAUTHORIZED","message":"Missing Authorization Bearer token or X-API-KEY header"}`, http.StatusUnauthorized)
				return
			}

			// Check if token matches preset environment keys or master token
			if cfg != nil {
				for _, preset := range cfg.PresetAPIKeys {
					if subtle.ConstantTimeCompare([]byte(token), []byte(preset)) == 1 {
						claims := &service.AuthTokenClaims{
							UserID: "env_admin",
							Email:  cfg.AdminEmail,
							Name:   "Preset Administrator",
							Role:   domain.RoleAdmin,
						}
						ctx := context.WithValue(r.Context(), UserKey, claims)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			claims, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"code":"UNAUTHORIZED","message":"%s"}`, err.Error()), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(UserKey).(*service.AuthTokenClaims)
		if !ok || claims.Role != domain.RoleAdmin {
			http.Error(w, `{"code":"FORBIDDEN","message":"Administrator access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func OptionalAuth(cfg *config.Config, authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				token = strings.TrimSpace(authHeader[7:])
			} else if apiKeyHeader := r.Header.Get("X-API-KEY"); apiKeyHeader != "" {
				token = strings.TrimSpace(apiKeyHeader)
			}

			if token != "" {
				if cfg != nil {
					for _, preset := range cfg.PresetAPIKeys {
						if subtle.ConstantTimeCompare([]byte(token), []byte(preset)) == 1 {
							claims := &service.AuthTokenClaims{
								UserID: "env_admin",
								Email:  cfg.AdminEmail,
								Name:   "Preset Administrator",
								Role:   domain.RoleAdmin,
							}
							ctx := context.WithValue(r.Context(), UserKey, claims)
							r = r.WithContext(ctx)
							next.ServeHTTP(w, r)
							return
						}
					}
				}
				if claims, err := authService.ValidateToken(token); err == nil {
					ctx := context.WithValue(r.Context(), UserKey, claims)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
