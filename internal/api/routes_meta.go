package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerHealthRoutes(engine *gin.Engine, meta Meta) {
	engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": meta.Service,
			"env":     meta.Env,
		})
	})
}

func registerMetaRoutes(api *gin.RouterGroup, meta Meta) {
	api.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"service": meta.Service,
			"env":     meta.Env,
		})
	})
}
