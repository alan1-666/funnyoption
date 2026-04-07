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
	rateLimiter      *rateLimiter
	corsExtraOrigins []string
}

func NewEngine(meta Meta, deps handler.Dependencies, corsExtraOrigins []string) *gin.Engine {
	return newEngine(meta, deps, routerOptions{corsExtraOrigins: corsExtraOrigins})
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
	applyGlobalMiddleware(engine, logger, opts)
	registerRoutes(engine, meta, deps, limiter)
	return engine
}

func applyGlobalMiddleware(engine *gin.Engine, logger *slog.Logger, opts routerOptions) {
	engine.Use(gin.Recovery())
	engine.Use(requestBodyLimitMiddleware())
	engine.Use(requestLoggingMiddleware(logger))
	engine.Use(corsMiddleware(opts.corsExtraOrigins...))
}

func registerRoutes(engine *gin.Engine, meta Meta, deps handler.Dependencies, limiter *rateLimiter) {
	orderHandler := handler.NewOrderHandler(deps)

	registerHealthRoutes(engine, meta)

	api := engine.Group("/api/v1")
	registerMetaRoutes(api, meta)
	registerPublicReadRoutes(api, orderHandler)
	registerUserScopedReadRoutes(api, orderHandler)
	registerSessionRoutes(api, orderHandler, limiter)
	registerTradeRoutes(api, orderHandler, limiter)
	registerClaimRoutes(api, orderHandler, limiter)
	registerPrivilegedRoutes(api, orderHandler, limiter)
}
