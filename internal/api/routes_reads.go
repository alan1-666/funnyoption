package api

import (
	"funnyoption/internal/api/handler"

	"github.com/gin-gonic/gin"
)

func registerPublicReadRoutes(api *gin.RouterGroup, orderHandler *handler.OrderHandler) {
	api.GET("/markets", orderHandler.ListMarkets)
	api.GET("/markets/:market_id", orderHandler.GetMarket)
	api.GET("/orders", orderHandler.ListOrders)
	api.GET("/trades", orderHandler.ListTrades)
	api.GET("/balances", orderHandler.ListBalances)
	api.GET("/positions", orderHandler.ListPositions)
	api.GET("/payouts", orderHandler.ListPayouts)
	api.GET("/profile", orderHandler.GetProfile)
	api.GET("/freezes", orderHandler.ListFreezes)
	api.GET("/ledger/entries", orderHandler.ListLedgerEntries)
	api.GET("/ledger/entries/:entry_id/postings", orderHandler.ListLedgerPostings)
	api.GET("/reports/liabilities", orderHandler.GetLiabilityReport)
	api.GET("/deposits", orderHandler.ListDeposits)
	api.GET("/withdrawals", orderHandler.ListWithdrawals)
	api.GET("/chain-transactions", orderHandler.ListChainTransactions)
}
