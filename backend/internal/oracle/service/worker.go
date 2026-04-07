package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	oraclecore "funnyoption/internal/oracle"
	sharedkafka "funnyoption/internal/shared/kafka"
)

type Worker struct {
	logger         *slog.Logger
	store          MarketStore
	provider       priceProvider
	publisher      sharedkafka.Publisher
	topics         sharedkafka.Topics
	pollInterval   time.Duration
	batchSize      int
	signerKey      *ecdsa.PrivateKey
	trustedSigners []common.Address
}

func NewWorker(logger *slog.Logger, store MarketStore, provider priceProvider, publisher sharedkafka.Publisher, topics sharedkafka.Topics, pollInterval time.Duration) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	return &Worker{
		logger:       logger,
		store:        store,
		provider:     provider,
		publisher:    publisher,
		topics:       topics,
		pollInterval: pollInterval,
		batchSize:    20,
	}
}

func (w *Worker) SetSignerKey(key *ecdsa.PrivateKey) {
	w.signerKey = key
}

func (w *Worker) SetTrustedSigners(signers []common.Address) {
	w.trustedSigners = signers
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	if err := w.pollOnce(ctx); err != nil {
		w.logger.Error("initial oracle poll failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.pollOnce(ctx); err != nil {
				w.logger.Error("oracle poll failed", "err", err)
			}
		}
	}
}

func (w *Worker) pollOnce(ctx context.Context) error {
	if w.store == nil {
		return fmt.Errorf("oracle market store is required")
	}
	frozen, err := w.store.RollupFrozen(ctx)
	if err != nil {
		return err
	}
	if frozen {
		return nil
	}
	now := time.Now().Unix()
	markets, err := w.store.ListEligibleMarkets(ctx, now, w.batchSize)
	if err != nil {
		return err
	}
	for _, market := range markets {
		if err := w.processMarket(ctx, now, market); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) processMarket(ctx context.Context, now int64, market EligibleMarket) error {
	optionKeys, err := optionKeysFromSchema(market.OptionSchema)
	if err != nil {
		return w.writeTerminalResolution(ctx, market, nil, ErrorCodeInvalidMetadata)
	}
	contract, isOracle, err := ParseContract(market.CategoryKey, optionKeys, market.ResolveAt, market.Metadata)
	if err != nil {
		return w.writeTerminalResolution(ctx, market, contract, ErrorCodeInvalidMetadata)
	}
	if !isOracle {
		return nil
	}

	resolverRef := contract.ResolverRef(market.ResolveAt)
	if status := normalizeUpper(market.ResolutionStatus); status == "RESOLVED" {
		if normalizeUpper(market.ResolverType) == ResolverTypeOraclePrice &&
			strings.TrimSpace(market.ResolverRef) == resolverRef &&
			normalizeUpper(market.ResolvedOutcome) != "" {
			return nil
		}
		return w.writeTerminalResolution(ctx, market, contract, ErrorCodeConflictingObservation)
	}
	if status := normalizeUpper(market.ResolutionStatus); status == "OBSERVED" {
		if normalizeUpper(market.ResolverType) == ResolverTypeOraclePrice &&
			strings.TrimSpace(market.ResolverRef) == resolverRef &&
			normalizeUpper(market.ResolvedOutcome) != "" {
			if !dispatchPending(market.Evidence) {
				return nil
			}
			return w.dispatchObservedResolution(ctx, ResolutionUpdate{
				MarketID:        market.MarketID,
				Status:          "OBSERVED",
				ResolvedOutcome: normalizeUpper(market.ResolvedOutcome),
				ResolverType:    ResolverTypeOraclePrice,
				ResolverRef:     resolverRef,
				Evidence:        market.Evidence,
			})
		}
		return w.writeTerminalResolution(ctx, market, contract, ErrorCodeConflictingObservation)
	}

	observation, providerErr := w.provider.Observe(ctx, contract)
	if providerErr != nil {
		return w.writeObservationError(ctx, now, market, contract, providerErr)
	}
	if observation == nil {
		return w.writeObservationError(ctx, now, market, contract, &providerError{
			Code:      ErrorCodeSourceUnavailable,
			Retryable: true,
			Message:   "oracle provider returned no observation",
		})
	}

	statusCode := w.validateObservationTiming(now, market.ResolveAt, contract, observation)
	if statusCode != "" {
		return w.writeObservationError(ctx, now, market, contract, &providerError{
			Code:      statusCode,
			Retryable: statusCode == ErrorCodePriceNotInWindow || statusCode == ErrorCodeStalePrice,
			Message:   statusCode,
		})
	}

	resolvedOutcome, err := contract.ResolveOutcome(observation.ObservedScaled)
	if err != nil {
		return w.writeTerminalResolution(ctx, market, contract, ErrorCodeUnsupportedRule)
	}

	var attestation *EvidenceAttestation
	if w.signerKey != nil {
		att, attErr := oraclecore.BuildOracleAttestation(
			contract.Metadata.Oracle.Instrument.Symbol,
			observation.ObservedPrice,
			observation.EffectiveAt,
			contract.Metadata.Oracle.ProviderKey,
			w.signerKey,
		)
		if attErr != nil {
			w.logger.Warn("failed to build oracle attestation", "market_id", market.MarketID, "err", attErr)
		} else {
			attestation = &EvidenceAttestation{
				Version:       att.Version,
				AssetPair:     att.AssetPair,
				Price:         att.Price,
				Timestamp:     att.Timestamp,
				Provider:      att.Provider,
				Signature:     att.Signature,
				SignerAddress:  att.SignerAddress,
			}
		}
	}

	update := ResolutionUpdate{
		MarketID:        market.MarketID,
		Status:          "OBSERVED",
		ResolvedOutcome: resolvedOutcome,
		ResolverType:    ResolverTypeOraclePrice,
		ResolverRef:     resolverRef,
		Evidence: BuildEvidence(
			contract,
			market.ResolveAt,
			observation,
			market.Evidence,
			0,
			0,
			"",
			resolvedOutcome,
			attestation,
		),
	}
	if err := w.store.UpsertResolution(ctx, update); err != nil {
		return err
	}

	return w.dispatchObservedResolution(ctx, update)
}

func (w *Worker) writeObservationError(ctx context.Context, now int64, market EligibleMarket, contract *Contract, providerErr *providerError) error {
	if providerErr == nil {
		return nil
	}
	status := "TERMINAL_ERROR"
	nextRetryAt := int64(0)
	deadline := market.ResolveAt
	if contract != nil {
		deadline += contract.Metadata.Oracle.Window.AfterSec
	}
	if providerErr.Retryable && now <= deadline {
		status = "RETRYABLE_ERROR"
		nextRetryAt = now + retryDelaySeconds(w.pollInterval)
	}

	return w.store.UpsertResolution(ctx, ResolutionUpdate{
		MarketID:        market.MarketID,
		Status:          status,
		ResolvedOutcome: "",
		ResolverType:    ResolverTypeOraclePrice,
		ResolverRef:     resolverRefForMarket(contract, market.ResolveAt),
		Evidence: BuildEvidence(
			contract,
			market.ResolveAt,
			nil,
			market.Evidence,
			0,
			nextRetryAt,
			providerErr.Code,
			"",
			nil,
		),
	})
}

func (w *Worker) writeTerminalResolution(ctx context.Context, market EligibleMarket, contract *Contract, errorCode string) error {
	return w.store.UpsertResolution(ctx, ResolutionUpdate{
		MarketID:        market.MarketID,
		Status:          "TERMINAL_ERROR",
		ResolvedOutcome: "",
		ResolverType:    ResolverTypeOraclePrice,
		ResolverRef:     resolverRefForMarket(contract, market.ResolveAt),
		Evidence: BuildEvidence(
			contract,
			market.ResolveAt,
			nil,
			market.Evidence,
			0,
			0,
			errorCode,
			"",
			nil,
		),
	})
}

func (w *Worker) validateObservationTiming(now, resolveAt int64, contract *Contract, observation *Observation) string {
	if contract == nil || observation == nil {
		return ErrorCodeInvalidMetadata
	}
	targetBefore := resolveAt - contract.Metadata.Oracle.Window.BeforeSec
	targetAfter := resolveAt + contract.Metadata.Oracle.Window.AfterSec
	if observation.EffectiveAt < targetBefore || observation.EffectiveAt > targetAfter {
		if now > targetAfter {
			return ErrorCodePriceNotInWindow
		}
		return ErrorCodePriceNotInWindow
	}
	if observation.FetchedAt-observation.EffectiveAt > contract.Metadata.Oracle.Price.MaxDataAgeSec {
		return ErrorCodeStalePrice
	}
	return ""
}

func (w *Worker) publishResolvedEvent(ctx context.Context, marketID int64, outcome string) error {
	if w.publisher == nil {
		return fmt.Errorf("oracle market publisher is required")
	}
	event := sharedkafka.MarketEvent{
		EventID:          sharedkafka.NewID("evt_market"),
		MarketID:         marketID,
		Status:           "RESOLVED",
		ResolvedOutcome:  normalizeUpper(outcome),
		OccurredAtMillis: time.Now().UnixMilli(),
	}
	return w.publisher.PublishJSON(ctx, w.topics.MarketEvent, fmt.Sprintf("%d", marketID), event)
}

func (w *Worker) dispatchObservedResolution(ctx context.Context, update ResolutionUpdate) error {
	attemptedAt := nowUnix()
	if err := w.publishResolvedEvent(ctx, update.MarketID, update.ResolvedOutcome); err != nil {
		update.Evidence = recordDispatchAttempt(update.Evidence, false, attemptedAt, err.Error())
		if markErr := w.store.UpsertResolution(ctx, update); markErr != nil {
			return fmt.Errorf("publish resolved market.event: %w (also failed to persist pending dispatch: %v)", err, markErr)
		}
		return err
	}
	update.Evidence = recordDispatchAttempt(update.Evidence, true, attemptedAt, "")
	return w.store.UpsertResolution(ctx, update)
}

func optionKeysFromSchema(raw json.RawMessage) ([]string, error) {
	type marketOption struct {
		Key string `json:"key"`
	}
	var options []marketOption
	if err := json.Unmarshal(raw, &options); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(options))
	for _, option := range options {
		keys = append(keys, option.Key)
	}
	return keys, nil
}

func resolverRefForMarket(contract *Contract, resolveAt int64) string {
	if contract == nil {
		return ""
	}
	return contract.ResolverRef(resolveAt)
}

func retryDelaySeconds(interval time.Duration) int64 {
	if interval <= 0 {
		return 5
	}
	seconds := int64(interval / time.Second)
	if seconds <= 0 {
		return 1
	}
	return seconds
}
