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

func registerUserScopedReadRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler) {
	userScoped := api.Group("")
	userScoped.Use(requireSessionAuth(orderHandler.LookupActiveSession))
	userScoped.Use(enforceUserScope())

	userScoped.GET("/orders", orderHandler.ListOrders)
	userScoped.GET("/balances", orderHandler.ListBalances)
	userScoped.GET("/positions", orderHandler.ListPositions)
	userScoped.GET("/payouts", orderHandler.ListPayouts)
	userScoped.GET("/profile", orderHandler.GetProfile)
	userScoped.GET("/freezes", orderHandler.ListFreezes)
	userScoped.GET("/deposits", orderHandler.ListDeposits)
	userScoped.GET("/withdrawals", orderHandler.ListWithdrawals)
	userScoped.GET("/ledger/entries", orderHandler.ListLedgerEntries)
	userScoped.GET("/ledger/entries/:entry_id/postings", orderHandler.ListLedgerPostings)
	userScoped.GET("/reports/liabilities", orderHandler.GetLiabilityReport)
}
