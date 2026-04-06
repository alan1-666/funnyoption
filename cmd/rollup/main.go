package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	chainservice "funnyoption/internal/chain/service"
	"funnyoption/internal/rollup"
	"funnyoption/internal/shared/config"
	shareddb "funnyoption/internal/shared/db"
	"funnyoption/internal/shared/logger"
)

func main() {
	cfg := config.Load("rollup")
	logr := logger.New(cfg.LogLevel)

	mode := flag.String("mode", "prepare-next", "rollup mode: prepare-next | submit-next | submit-until-idle | print-genesis-root | request-forced-withdrawal | freeze-forced-withdrawal | claim-escape-collateral")
	limit := flag.Int("limit", 256, "max journal entries to materialize into one new batch when needed")
	timeout := flag.Duration("timeout", 10*time.Second, "overall command timeout")
	walletPrivateKey := flag.String("wallet-private-key", "", "wallet private key for user-driven escape hatch actions")
	recipient := flag.String("recipient", "", "recipient address for forced-withdrawal or escape collateral claim (defaults to wallet address)")
	amount := flag.Int64("amount", 0, "accounting amount for forced withdrawal requests")
	requestID := flag.Uint64("request-id", 0, "forced withdrawal request id")
	accountID := flag.Int64("account-id", 0, "account id filter for escape collateral claims")
	claimID := flag.String("claim-id", "", "escape collateral claim id filter")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	switch *mode {
	case "prepare-next":
		db, err := shareddb.OpenPostgres(ctx, cfg.PostgresDSN)
		if err != nil {
			log.Fatalf("open postgres: %v", err)
		}
		defer db.Close()

		store := rollup.NewStore(db)
		prepared, err := store.PrepareNextSubmission(ctx, *limit)
		if err != nil {
			if errors.Is(err, rollup.ErrNoPendingSubmission) {
				log.Fatalf("no pending rollup submission")
			}
			log.Fatalf("prepare next rollup submission: %v", err)
		}
		if err := writeJSON(prepared); err != nil {
			log.Fatalf("write output: %v", err)
		}
	case "submit-next":
		cfg.RollupBatchLimit = *limit
		progress, err := chainservice.RunRollupSubmissionOnce(ctx, logr, cfg)
		if err != nil {
			log.Fatalf("submit next rollup submission: %v", err)
		}
		if err := writeJSON(progress); err != nil {
			log.Fatalf("write output: %v", err)
		}
	case "submit-until-idle":
		cfg.RollupBatchLimit = *limit
		run, err := chainservice.RunRollupSubmissionUntilIdle(ctx, logr, cfg)
		if err != nil {
			log.Fatalf("submit rollup until idle: %v", err)
		}
		if err := writeJSON(run); err != nil {
			log.Fatalf("write output: %v", err)
		}
	case "print-genesis-root":
		if err := writeJSON(map[string]string{
			"genesis_state_root": rollup.ZeroStateRoot(),
		}); err != nil {
			log.Fatalf("write output: %v", err)
		}
	case "request-forced-withdrawal":
		progress, err := chainservice.RunRequestForcedWithdrawalOnce(ctx, logr, cfg, *walletPrivateKey, *amount, *recipient)
		if err != nil {
			log.Fatalf("request forced withdrawal: %v", err)
		}
		if err := writeJSON(progress); err != nil {
			log.Fatalf("write output: %v", err)
		}
	case "freeze-forced-withdrawal":
		progress, err := chainservice.RunFreezeForcedWithdrawalOnce(ctx, logr, cfg, *requestID)
		if err != nil {
			log.Fatalf("freeze forced withdrawal: %v", err)
		}
		if err := writeJSON(progress); err != nil {
			log.Fatalf("write output: %v", err)
		}
	case "claim-escape-collateral":
		progress, err := chainservice.RunClaimEscapeCollateralOnce(ctx, logr, cfg, *walletPrivateKey, *accountID, *claimID, *recipient)
		if err != nil {
			log.Fatalf("claim escape collateral: %v", err)
		}
		if err := writeJSON(progress); err != nil {
			log.Fatalf("write output: %v", err)
		}
	default:
		log.Fatalf("unsupported rollup mode: %s", *mode)
	}
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}
