package api

import (
	"funnyoption/internal/api/handler"

	"github.com/gin-gonic/gin"
)

func registerPublicReadRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler) {
	api.GET("/markets", orderHandler.ListMarkets)
	api.GET("/markets/:market_id", orderHandler.GetMarket)
	api.GET("/trades", orderHandler.ListTrades)

	api.GET("/rollup/forced-withdrawals", orderHandler.ListRollupForcedWithdrawals)
	api.GET("/rollup/escape-collateral", orderHandler.ListRollupEscapeCollateralClaims)
	api.GET("/rollup/withdrawal-claims", orderHandler.ListRollupWithdrawalClaims)
	api.GET("/rollup/freeze-state", orderHandler.GetRollupFreezeState)
	api.GET("/chain-transactions", orderHandler.ListChainTransactions)
}

func registerUserScopedReadRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler, notifHandler *handler.NotificationHandler) {
	userScoped := api.Group("")
	userScoped.Use(requireSessionAuth(orderHandler.LookupActiveSession))
	userScoped.Use(enforceUserScope())

	userScoped.GET("/orders", orderHandler.ListOrders)
	userScoped.GET("/balances", orderHandler.ListBalances)
	userScoped.GET("/positions", orderHandler.ListPositions)
	userScoped.GET("/payouts", orderHandler.ListPayouts)
	userScoped.GET("/profile", orderHandler.GetProfile)
	userScoped.GET("/freezes", orderHandler.ListFreezes)
	userScoped.GET("/ledger/entries", orderHandler.ListLedgerEntries)
	userScoped.GET("/ledger/entries/:entry_id/postings", orderHandler.ListLedgerPostings)
	userScoped.GET("/reports/liabilities", orderHandler.GetLiabilityReport)
	userScoped.GET("/notifications", notifHandler.ListNotifications)
	userScoped.GET("/notifications/unread-count", notifHandler.UnreadCount)
}

func registerNotificationWriteRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler, notifHandler *handler.NotificationHandler) {
	authed := api.Group("")
	authed.Use(requireSessionAuth(orderHandler.LookupActiveSession))

	authed.PATCH("/notifications/:notification_id/read", notifHandler.MarkRead)
	authed.PATCH("/notifications/read-all", notifHandler.MarkAllRead)
}
