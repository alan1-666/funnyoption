package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"funnyoption/internal/matching/ha"
	"funnyoption/internal/matching/pipeline"
	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	"funnyoption/internal/shared/fee"
	"funnyoption/internal/shared/grpcx"
	sharedkafka "funnyoption/internal/shared/kafka"
)

func Run(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig) error {
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	store := NewSQLStore(dbConn).WithRollup(rollup.NewStore(dbConn))
	cachedStore := NewCachedCommandStore(store)

	sequence, err := store.MaxTradeSequence(ctx)
	if err != nil {
		return err
	}
	restingOrders, err := store.LoadRestingOrders(ctx)
	if err != nil {
		return err
	}

	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	defer publisher.Close()

	feeSched := fee.Schedule{TakerFeeBps: cfg.TakerFeeBps, MakerFeeBps: cfg.MakerFeeBps}

	candles := NewCandleBook(defaultCandleIntervalMillis, defaultCandleHistoryLimit)

	epochMgr := ha.NewEpochManager(0)
	roleMgr := ha.NewRoleManager(ha.RolePrimary)

	pipe := pipeline.New(
		logger,
		cfg.KafkaBrokers,
		cfg.KafkaTopics.OrderCommand,
		"funnyoption-matching",
		cachedStore,
		cachedStore,
		publisher,
		cfg.KafkaTopics,
		feeSched,
		candles,
		pipeline.Config{
			InputRBSize:  8192,
			OutputRBSize: 8192,
		},
		epochMgr,
	)

	roleMgr.OnTransition(func(old, newRole ha.Role) {
		logger.Info("HA role transition", "from", old.String(), "to", newRole.String())
		if newRole == ha.RolePrimary {
			newEpoch := epochMgr.Advance()
			pipe.SetDispatchMode(pipeline.DispatchModeActive)
			logger.Info("promoted to primary", "epoch", newEpoch)
		} else {
			pipe.SetDispatchMode(pipeline.DispatchModeShadow)
			logger.Info("demoted to standby")
		}
	})

	if err := pipe.Restore(sequence, restingOrders); err != nil {
		return err
	}

	pipe.Start(ctx)
	pipe.StartStatsReporter(ctx, logger, 10*time.Second)
	defer pipe.Close()

	expirySweeper := newOrderExpirySweeper(logger, pipe, store, publisher, cfg.KafkaTopics)
	expirySweeper.Start(ctx)

	startSnapshotHTTP(ctx, logger, pipe, epochMgr, roleMgr)

	logger.Info(
		"matching service bootstrapped (pipeline mode, HA enabled)",
		"ingress", "kafka",
		"grpc_addr", cfg.GRPCAddr,
		"kafka_brokers", cfg.KafkaBrokers,
		"order_command_topic", cfg.KafkaTopics.OrderCommand,
		"order_event_topic", cfg.KafkaTopics.OrderEvent,
		"trade_topic", cfg.KafkaTopics.TradeMatched,
		"depth_topic", cfg.KafkaTopics.QuoteDepth,
		"ticker_topic", cfg.KafkaTopics.QuoteTicker,
		"candle_topic", cfg.KafkaTopics.QuoteCandle,
		"postgres_dsn", cfg.PostgresDSN,
		"restored_trade_sequence", sequence,
		"restored_resting_orders", len(restingOrders),
		"input_rb_size", 8192,
		"output_rb_size", 8192,
		"ha_role", roleMgr.Current().String(),
		"epoch", epochMgr.Current(),
	)

	return grpcx.Run(ctx, logger, cfg.Name, cfg.GRPCAddr, nil)
}

func startSnapshotHTTP(ctx context.Context, logger *slog.Logger, pipe *pipeline.Pipeline, epochMgr *ha.EpochManager, roleMgr *ha.RoleManager) {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "matching"})
	})

	mux.HandleFunc("/ha/snapshot", func(w http.ResponseWriter, r *http.Request) {
		snap := pipe.TakeSnapshot()
		snap.EpochID = epochMgr.Current()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(snap)
	})

	mux.HandleFunc("/ha/status", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"role":            roleMgr.Current().String(),
			"epoch":           epochMgr.Current(),
			"globalSequence":  pipe.GlobalSequence(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	mux.HandleFunc("/ha/promote", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		roleMgr.Transition(ha.RolePrimary)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"role":  roleMgr.Current().String(),
			"epoch": epochMgr.Current(),
		})
	})

	mux.HandleFunc("/ha/demote", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		roleMgr.Transition(ha.RoleStandby)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"role":  roleMgr.Current().String(),
			"epoch": epochMgr.Current(),
		})
	})

	server := &http.Server{Addr: ":9190", Handler: mux}
	go func() {
		logger.Info("HA snapshot HTTP started", "addr", ":9190")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("snapshot HTTP server error", "err", err)
		}
	}()

	go func() {
		<-ctx.Done()
		server.Close()
	}()
}
