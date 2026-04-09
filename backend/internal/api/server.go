package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	accountclient "funnyoption/internal/account/client"
	"funnyoption/internal/api/handler"
	"funnyoption/internal/custody"
	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	sharedkafka "funnyoption/internal/shared/kafka"

	"github.com/gin-gonic/gin"
)

func Run(ctx context.Context, logger *slog.Logger, cfg config.ServiceConfig) error {
	if cfg.HTTPAddr == "" {
		return errors.New("http listen address is empty")
	}

	gin.SetMode(gin.ReleaseMode)
	publisher := sharedkafka.NewJSONPublisher(logger, cfg.KafkaBrokers)
	defer publisher.Close()
	dbConn, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	accountRPC, err := accountclient.NewGRPCClient(cfg.AccountGRPCAddr)
	if err != nil {
		return err
	}
	defer accountRPC.Close()

	var saasClient *custody.SaaSClient
	if cfg.CustodySaaSBaseURL != "" {
		saasClient = custody.NewSaaSClient(cfg.CustodySaaSBaseURL, cfg.CustodyDepositToken, cfg.CustodySaaSTenantID)
	}
	custodyStore := custody.NewStore(dbConn)
	custodyHandler := custody.NewHandler(custody.HandlerDeps{
		Logger:       logger,
		Store:        custodyStore,
		SaaS:         saasClient,
		Account:      accountRPC,
		DepositToken: cfg.CustodyDepositToken,
		Chain:            cfg.CustodyChainName,
		Network:          cfg.CustodyNetworkName,
		Coin:             cfg.CollateralSymbol,
		ChainDecimals:    cfg.CollateralDecimals,
		AccountingDigits: cfg.CollateralDisplayDigits,
	})

	engine := NewEngineWithCustody(Meta{
		Service: cfg.Name,
		Env:     cfg.Env,
	}, handler.Dependencies{
		Logger:                logger,
		DB:                    dbConn,
		KafkaPublisher:        publisher,
		KafkaTopics:           cfg.KafkaTopics,
		AccountClient:         accountRPC,
		QueryStore:            handler.NewSQLStore(dbConn).WithRollup(rollup.NewStore(dbConn)),
		OperatorWallets:       cfg.OperatorWallets,
		DefaultOperatorUserID: cfg.DefaultOperatorUserID,
		ExpectedChainID:       cfg.ChainID,
	}, custodyHandler, cfg.CORSExtraOrigins)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: engine,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http service listening", "service", cfg.Name, "addr", cfg.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("http service shutting down", "service", cfg.Name)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
