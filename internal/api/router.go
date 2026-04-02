package api

import (
	"log/slog"

	"funnyoption/internal/api/handler"

	"github.com/gin-gonic/gin"
)

type Meta struct {
	Service string
	Env     string
}

type routerOptions struct {
	rateLimiter *rateLimiter
}

func NewEngine(meta Meta, deps handler.Dependencies) *gin.Engine {
	return newEngine(meta, deps, routerOptions{})
}

func newEngine(meta Meta, deps handler.Dependencies, opts routerOptions) *gin.Engine {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	limiter := opts.rateLimiter
	if limiter == nil {
		limiter = newRateLimiter(defaultRateLimitPolicies())
	}

	engine := gin.New()
	applyGlobalMiddleware(engine, logger)
	registerRoutes(engine, meta, deps, limiter)
	return engine
}

func applyGlobalMiddleware(engine *gin.Engine, logger *slog.Logger) {
	// The stack is ordered so recovery wraps the whole chain, request logging
	// observes the final status, and CORS can short-circuit preflight requests.
	engine.Use(gin.Recovery())
	engine.Use(requestLoggingMiddleware(logger))
	engine.Use(corsMiddleware())
}

func registerRoutes(engine *gin.Engine, meta Meta, deps handler.Dependencies, limiter *rateLimiter) {
	orderHandler := handler.NewOrderHandler(deps)

	registerHealthRoutes(engine, meta)

	api := engine.Group("/api/v1")
	registerMetaRoutes(api, meta)
	registerPublicReadRoutes(api, orderHandler)
	registerSessionRoutes(api, orderHandler, limiter)
	registerTradeRoutes(api, orderHandler, limiter)
	registerClaimRoutes(api, orderHandler, limiter)
	registerPrivilegedRoutes(api, orderHandler, limiter)
}
