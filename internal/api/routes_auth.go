package api

import (
	"funnyoption/internal/api/handler"

	"github.com/gin-gonic/gin"
)

func registerSessionRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler, limiter *rateLimiter) {
	sessions := api.Group("/sessions")
	sessions.GET("", orderHandler.ListSessions)
	// Temporary proof-tool compatibility route. V2 browser auth must use the
	// trading-key challenge + registration flow instead of creating new callers
	// on the legacy session-grant contract.
	sessions.POST("", markAuthLane(authLaneWalletSession), limiter.Middleware(rateLimitSessionCreate), orderHandler.CreateSession)
	sessions.POST("/:session_id/revoke", markAuthLane(authLaneWalletSession), limiter.Middleware(rateLimitSessionWrite), orderHandler.RevokeSession)

	tradingKeys := api.Group("/trading-keys")
	tradingKeys.GET("", orderHandler.ListTradingKeys)
	tradingKeys.POST("/challenge", markAuthLane(authLaneWalletSession), limiter.Middleware(rateLimitSessionCreate), orderHandler.CreateTradingKeyChallenge)
	tradingKeys.POST("", markAuthLane(authLaneWalletSession), limiter.Middleware(rateLimitSessionWrite), orderHandler.RegisterTradingKey)

	profile := api.Group("")
	profile.Use(markAuthLane(authLaneWalletSession))
	profile.Use(limiter.Middleware(rateLimitSessionWrite))
	profile.PUT("/profile", orderHandler.UpdateProfile)
}

func registerTradeRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler, limiter *rateLimiter) {
	// Trade writes accept either end-user session authorization or the narrow
	// privileged bootstrap envelope used by the dedicated admin service.
	tradeWrites := api.Group("")
	tradeWrites.Use(markAuthLane(authLaneTradeWrite))
	tradeWrites.Use(limiter.Middleware(rateLimitTradeWrite))
	tradeWrites.Use(requireTradeWriteBoundary())
	tradeWrites.POST("/orders", orderHandler.CreateOrder)
}

func registerClaimRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler, limiter *rateLimiter) {
	claims := api.Group("")
	claims.Use(markAuthLane(authLaneClaimWrite))
	claims.Use(limiter.Middleware(rateLimitClaimWrite))
	claims.POST("/payouts/:event_id/claim", orderHandler.CreateClaimPayout)
}

func registerPrivilegedRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler, limiter *rateLimiter) {
	operatorWrites := api.Group("")
	operatorWrites.Use(markAuthLane(authLaneOperatorWrite))
	operatorWrites.Use(limiter.Middleware(rateLimitPrivilegedWrite))
	operatorWrites.Use(requireOperatorProofEnvelope())
	operatorWrites.POST("/markets", orderHandler.CreateMarket)
	operatorWrites.POST("/markets/:market_id/resolve", orderHandler.ResolveMarket)

	adminWrites := api.Group("/admin")
	adminWrites.Use(markAuthLane(authLaneOperatorWrite))
	adminWrites.Use(limiter.Middleware(rateLimitPrivilegedWrite))
	adminWrites.Use(requireOperatorProofEnvelope())
	adminWrites.POST("/markets/:market_id/first-liquidity", orderHandler.CreateFirstLiquidity)
}
