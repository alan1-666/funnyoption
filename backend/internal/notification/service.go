package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	"funnyoption/internal/shared/health"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type processor struct {
	logger    *slog.Logger
	db        *sql.DB
	publisher sharedkafka.Publisher
	topics    sharedkafka.Topics
}

func Run(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig) error {
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	defer publisher.Close()

	p := &processor{
		logger:    logger,
		db:        dbConn,
		publisher: publisher,
		topics:    cfg.KafkaTopics,
	}

	tradeConsumer := sharedkafka.NewJSONConsumer(
		logger, cfg.KafkaBrokers, cfg.KafkaTopics.TradeMatched,
		"funnyoption-notification", p.handleTradeMatched,
	)
	tradeConsumer.Start(ctx)
	defer tradeConsumer.Close()

	settlementConsumer := sharedkafka.NewJSONConsumer(
		logger, cfg.KafkaBrokers, cfg.KafkaTopics.SettlementDone,
		"funnyoption-notification", p.handleSettlementCompleted,
	)
	settlementConsumer.Start(ctx)
	defer settlementConsumer.Close()

	marketConsumer := sharedkafka.NewJSONConsumer(
		logger, cfg.KafkaBrokers, cfg.KafkaTopics.MarketEvent,
		"funnyoption-notification", p.handleMarketEvent,
	)
	marketConsumer.Start(ctx)
	defer marketConsumer.Close()

	health.ListenAndServe(ctx, logger, cfg.HTTPAddr, cfg.Name, cfg.Env)

	logger.Info("notification service started",
		"trade_topic", cfg.KafkaTopics.TradeMatched,
		"settlement_topic", cfg.KafkaTopics.SettlementDone,
		"market_topic", cfg.KafkaTopics.MarketEvent,
		"health_addr", cfg.HTTPAddr,
	)

	<-ctx.Done()
	logger.Info("notification service shutting down")
	return nil
}

func (p *processor) handleTradeMatched(ctx context.Context, msg sharedkafka.Message) error {
	var ev sharedkafka.TradeMatchedEvent
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		p.logger.Error("notification: unmarshal trade event", "err", err)
		return nil
	}

	takerTitle := fmt.Sprintf("Order filled: %s %s @ %d",
		ev.TakerSide, ev.Outcome, ev.Price)
	makerTitle := fmt.Sprintf("Order filled: %s %s @ %d",
		ev.MakerSide, ev.Outcome, ev.Price)
	meta := fmt.Sprintf(`{"market_id":%d,"trade_id":"%s","quantity":%d}`,
		ev.MarketID, ev.EventID, ev.Quantity)

	if err := p.insertAndPublish(ctx, ev.TakerUserID, "trade_filled", takerTitle, "", meta); err != nil {
		p.logger.Error("notification: insert taker trade notif", "err", err, "user_id", ev.TakerUserID)
	}
	if ev.MakerUserID != ev.TakerUserID {
		if err := p.insertAndPublish(ctx, ev.MakerUserID, "trade_filled", makerTitle, "", meta); err != nil {
			p.logger.Error("notification: insert maker trade notif", "err", err, "user_id", ev.MakerUserID)
		}
	}
	return nil
}

func (p *processor) handleSettlementCompleted(ctx context.Context, msg sharedkafka.Message) error {
	var ev sharedkafka.SettlementCompletedEvent
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		p.logger.Error("notification: unmarshal settlement event", "err", err)
		return nil
	}

	title := fmt.Sprintf("Settlement payout: %d %s for market %d",
		ev.PayoutAmount, ev.PayoutAsset, ev.MarketID)
	meta := fmt.Sprintf(`{"market_id":%d,"payout_amount":%d,"payout_asset":"%s"}`,
		ev.MarketID, ev.PayoutAmount, ev.PayoutAsset)

	if err := p.insertAndPublish(ctx, ev.UserID, "settlement_payout", title, "", meta); err != nil {
		p.logger.Error("notification: insert settlement notif", "err", err, "user_id", ev.UserID)
	}
	return nil
}

func (p *processor) handleMarketEvent(ctx context.Context, msg sharedkafka.Message) error {
	var ev sharedkafka.MarketEvent
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		p.logger.Error("notification: unmarshal market event", "err", err)
		return nil
	}

	if ev.Status != "RESOLVED" {
		return nil
	}

	title := fmt.Sprintf("Market %d resolved: %s", ev.MarketID, ev.ResolvedOutcome)
	meta := fmt.Sprintf(`{"market_id":%d,"resolved_outcome":"%s"}`, ev.MarketID, ev.ResolvedOutcome)

	rows, err := p.db.QueryContext(ctx, `
		SELECT DISTINCT user_id FROM positions WHERE market_id = $1 AND quantity > 0
	`, ev.MarketID)
	if err != nil {
		p.logger.Error("notification: query positioned users", "err", err, "market_id", ev.MarketID)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		if err := p.insertAndPublish(ctx, userID, "market_resolved", title, "", meta); err != nil {
			p.logger.Error("notification: insert market resolved notif", "err", err, "user_id", userID)
		}
	}
	return nil
}

func (p *processor) insertAndPublish(ctx context.Context, userID int64, notifType, title, body, metadata string) error {
	if metadata == "" {
		metadata = "{}"
	}
	now := time.Now().Unix()
	var id int64
	err := p.db.QueryRowContext(ctx, `
		INSERT INTO notifications (user_id, type, title, body, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		RETURNING notification_id
	`, userID, notifType, title, body, metadata, now).Scan(&id)
	if err != nil {
		return err
	}

	ev := sharedkafka.NotificationCreatedEvent{
		NotificationID: id,
		UserID:         userID,
		Type:           notifType,
		Title:          title,
		CreatedAt:      now,
	}
	return p.publisher.PublishJSON(ctx, p.topics.NotificationCreated, fmt.Sprintf("%d", userID), ev)
}
