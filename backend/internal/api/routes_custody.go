package api

import (
	"funnyoption/internal/api/handler"
	"funnyoption/internal/custody"

	"github.com/gin-gonic/gin"
)

func registerCustodyRoutes(engine *gin.Engine, deps handler.Dependencies, ch *custody.Handler) {
	internal := engine.Group("/internal/custody")
	internal.POST("/deposit/notify", ch.DepositNotify)

	// SaaS ProjectNotify callback path (called from scan-account-service)
	engine.POST("/v1/account/asset/deposit/notify", ch.DepositNotify)

	orderHandler := handler.NewOrderHandler(deps)
	api := engine.Group("/api/v1/custody")
	api.Use(requireSessionAuth(orderHandler.LookupActiveSession))
	api.Use(enforceUserScope())
	api.GET("/deposit-address", ch.GetDepositAddress)
	api.POST("/withdraw", ch.RequestWithdraw)
}
