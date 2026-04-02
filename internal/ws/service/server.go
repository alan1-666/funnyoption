package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"funnyoption/internal/shared/config"
	sharedkafka "funnyoption/internal/shared/kafka"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Run(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig) error {
	if cfg.HTTPAddr == "" {
		return errors.New("http listen address is empty")
	}

	hub := NewHub()
	depthConsumer := sharedkafka.NewJSONConsumer(logger, cfg.KafkaBrokers, cfg.KafkaTopics.QuoteDepth, "funnyoption-ws", hub.HandleDepth)
	tickerConsumer := sharedkafka.NewJSONConsumer(logger, cfg.KafkaBrokers, cfg.KafkaTopics.QuoteTicker, "funnyoption-ws", hub.HandleTicker)
	candleConsumer := sharedkafka.NewJSONConsumer(logger, cfg.KafkaBrokers, cfg.KafkaTopics.QuoteCandle, "funnyoption-ws", hub.HandleCandle)
	marketConsumer := sharedkafka.NewJSONConsumer(logger, cfg.KafkaBrokers, cfg.KafkaTopics.MarketEvent, "funnyoption-ws", hub.HandleMarketEvent)
	settlementConsumer := sharedkafka.NewJSONConsumer(logger, cfg.KafkaBrokers, cfg.KafkaTopics.SettlementDone, "funnyoption-ws", hub.HandleSettlementCompleted)
	depthConsumer.Start(ctx)
	defer depthConsumer.Close()
	tickerConsumer.Start(ctx)
	defer tickerConsumer.Close()
	candleConsumer.Start(ctx)
	defer candleConsumer.Close()
	marketConsumer.Start(ctx)
	defer marketConsumer.Close()
	settlementConsumer.Start(ctx)
	defer settlementConsumer.Close()

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": cfg.Name, "env": cfg.Env})
	})
	engine.GET("/ws", func(c *gin.Context) {
		stream := c.DefaultQuery("stream", "depth")
		bookKey := c.Query("book_key")
		marketID := c.Query("market_id")
		if bookKey == "" {
			if stream == "market" && marketID != "" {
				bookKey = marketID
			}
		}
		if bookKey == "" {
			if stream == "market" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "market_id is required"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "book_key is required"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		topic := stream + ":" + bookKey
		sub, unsubscribe := hub.Subscribe(topic)
		defer unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			case body, ok := <-sub.ch:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, body); err != nil {
					return
				}
			}
		}
	})

	server := &http.Server{Addr: cfg.HTTPAddr, Handler: engine}
	errCh := make(chan error, 1)
	go func() {
		logger.Info(
			"ws service listening",
			"addr", cfg.HTTPAddr,
			"depth_topic", cfg.KafkaTopics.QuoteDepth,
			"ticker_topic", cfg.KafkaTopics.QuoteTicker,
			"candle_topic", cfg.KafkaTopics.QuoteCandle,
			"market_topic", cfg.KafkaTopics.MarketEvent,
			"settlement_topic", cfg.KafkaTopics.SettlementDone,
		)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return server.Shutdown(context.Background())
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
