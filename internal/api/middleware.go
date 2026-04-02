package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"funnyoption/internal/api/dto"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const (
	authLaneContextKey = "api.auth_lane"
	requestBodyKey     = "api.request_body"

	authLaneWalletSession = "wallet_session"
	authLaneTradeWrite    = "trade_write"
	authLaneTradeSession  = "trade_session"
	authLaneTradeOperator = "trade_operator"
	authLaneClaimWrite    = "claim_write"
	authLaneOperatorWrite = "operator_write"

	rateLimitSessionCreate   = "session.create"
	rateLimitSessionWrite    = "session.write"
	rateLimitTradeWrite      = "trade.write"
	rateLimitClaimWrite      = "claim.write"
	rateLimitPrivilegedWrite = "privileged.write"
)

type tradeWriteEnvelope struct {
	UserID            int64               `json:"user_id"`
	SessionID         string              `json:"session_id"`
	SessionSignature  string              `json:"session_signature"`
	RequestedAtMillis int64               `json:"requested_at"`
	Operator          *dto.OperatorAction `json:"operator"`
}

type operatorEnvelope struct {
	Operator *dto.OperatorAction `json:"operator"`
}

type rateLimitPolicy struct {
	Limit rate.Limit
	Burst int
	KeyFn func(*gin.Context) string
	Label string
}

type rateLimiter struct {
	mu       sync.Mutex
	now      func() time.Time
	policies map[string]rateLimitPolicy
	buckets  map[string]*rateBucket
	ttl      time.Duration
}

type rateBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if origin := ctx.GetHeader("Origin"); origin != "" {
			ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			ctx.Writer.Header().Set("Vary", "Origin")
			ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}

func requestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		fields := []any{
			"method", ctx.Request.Method,
			"path", ctx.FullPath(),
			"status", ctx.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", clientIdentifier(ctx),
		}

		if fullPath := ctx.FullPath(); fullPath == "" {
			fields[3] = ctx.Request.URL.Path
		}
		if authLane, ok := ctx.Get(authLaneContextKey); ok {
			fields = append(fields, "auth_lane", authLane)
		}
		if len(ctx.Errors) > 0 {
			fields = append(fields, "errors", ctx.Errors.String())
		}
		if traceID := strings.TrimSpace(ctx.GetHeader("X-Trace-Id")); traceID != "" {
			fields = append(fields, "trace_id", traceID)
		}

		logger.Info("http request", fields...)
	}
}

func markAuthLane(lane string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set(authLaneContextKey, lane)
		ctx.Next()
	}
}

func requireTradeWriteBoundary() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var payload tradeWriteEnvelope
		if err := decodeJSONBody(ctx, &payload); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hasSessionFields := strings.TrimSpace(payload.SessionID) != "" ||
			strings.TrimSpace(payload.SessionSignature) != ""

		if hasSessionFields {
			if strings.TrimSpace(payload.SessionID) == "" ||
				strings.TrimSpace(payload.SessionSignature) == "" ||
				payload.RequestedAtMillis <= 0 {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "session-backed trade authorization is required",
				})
				return
			}
			ctx.Set(authLaneContextKey, authLaneTradeSession)
			ctx.Next()
			return
		}

		if payload.Operator != nil {
			if strings.TrimSpace(payload.Operator.WalletAddress) == "" ||
				strings.TrimSpace(payload.Operator.Signature) == "" ||
				payload.Operator.RequestedAt <= 0 {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "operator wallet, signature, and requested_at are required",
				})
				return
			}
			ctx.Set(authLaneContextKey, authLaneTradeOperator)
			ctx.Next()
			return
		}

		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "session-backed trade authorization or operator proof is required",
		})
	}
}

func requireOperatorProofEnvelope() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var payload operatorEnvelope
		if err := decodeJSONBody(ctx, &payload); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if payload.Operator == nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "operator proof is required for privileged actions",
			})
			return
		}
		if strings.TrimSpace(payload.Operator.WalletAddress) == "" ||
			strings.TrimSpace(payload.Operator.Signature) == "" ||
			payload.Operator.RequestedAt <= 0 {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "operator wallet, signature, and requested_at are required",
			})
			return
		}
		ctx.Next()
	}
}

func decodeJSONBody(ctx *gin.Context, target any) error {
	raw, err := cachedRequestBody(ctx)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return io.EOF
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func cachedRequestBody(ctx *gin.Context) ([]byte, error) {
	if cached, ok := ctx.Get(requestBodyKey); ok {
		if raw, ok := cached.([]byte); ok {
			ctx.Request.Body = io.NopCloser(bytes.NewReader(raw))
			return raw, nil
		}
	}

	if ctx.Request == nil || ctx.Request.Body == nil {
		return nil, io.EOF
	}

	raw, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, err
	}
	ctx.Set(requestBodyKey, raw)
	ctx.Request.Body = io.NopCloser(bytes.NewReader(raw))
	return raw, nil
}

func defaultRateLimitPolicies() map[string]rateLimitPolicy {
	keyFn := func(ctx *gin.Context) string { return clientIdentifier(ctx) }
	return map[string]rateLimitPolicy{
		rateLimitSessionCreate: {
			Limit: requestsPerWindow(5, time.Minute),
			Burst: 5,
			KeyFn: keyFn,
			Label: "session create",
		},
		rateLimitSessionWrite: {
			Limit: requestsPerWindow(10, time.Minute),
			Burst: 5,
			KeyFn: keyFn,
			Label: "session write",
		},
		rateLimitTradeWrite: {
			Limit: requestsPerWindow(30, time.Minute),
			Burst: 10,
			KeyFn: keyFn,
			Label: "trade write",
		},
		rateLimitClaimWrite: {
			Limit: requestsPerWindow(5, time.Minute),
			Burst: 3,
			KeyFn: keyFn,
			Label: "claim write",
		},
		rateLimitPrivilegedWrite: {
			Limit: requestsPerWindow(10, time.Minute),
			Burst: 5,
			KeyFn: keyFn,
			Label: "privileged write",
		},
	}
}

func newRateLimiter(policies map[string]rateLimitPolicy) *rateLimiter {
	return &rateLimiter{
		now:      time.Now,
		policies: policies,
		buckets:  make(map[string]*rateBucket),
		ttl:      15 * time.Minute,
	}
}

func (r *rateLimiter) Middleware(policyName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		allowed, retryAfter, ok := r.allow(policyName, ctx)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "rate limit policy is not configured"})
			return
		}
		if allowed {
			ctx.Next()
			return
		}

		if retryAfter > 0 {
			ctx.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
		}
		ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
	}
}

func (r *rateLimiter) allow(policyName string, ctx *gin.Context) (bool, time.Duration, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	policy, ok := r.policies[policyName]
	if !ok {
		return false, 0, false
	}

	now := r.now()
	r.cleanup(now)

	keyFn := policy.KeyFn
	if keyFn == nil {
		keyFn = func(ctx *gin.Context) string { return clientIdentifier(ctx) }
	}

	key := policyName + ":" + keyFn(ctx)
	bucket, found := r.buckets[key]
	if !found {
		bucket = &rateBucket{
			limiter: rate.NewLimiter(policy.Limit, policy.Burst),
		}
		r.buckets[key] = bucket
	}
	bucket.lastSeen = now

	reservation := bucket.limiter.ReserveN(now, 1)
	if !reservation.OK() {
		return false, 0, true
	}
	delay := reservation.DelayFrom(now)
	if delay > 0 {
		reservation.CancelAt(now)
		return false, delay, true
	}
	return true, 0, true
}

func (r *rateLimiter) cleanup(now time.Time) {
	for key, bucket := range r.buckets {
		if now.Sub(bucket.lastSeen) > r.ttl {
			delete(r.buckets, key)
		}
	}
}

func requestsPerWindow(requests int, window time.Duration) rate.Limit {
	if requests <= 0 || window <= 0 {
		return rate.Inf
	}
	return rate.Every(window / time.Duration(requests))
}

func clientIdentifier(ctx *gin.Context) string {
	if ctx == nil || ctx.Request == nil {
		return "unknown"
	}
	return remoteHost(ctx.Request.RemoteAddr)
}

func remoteHost(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" {
		return "unknown"
	}
	return host
}
