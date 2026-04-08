package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"funnyoption/internal/shared/config"
	sharedkafka "funnyoption/internal/shared/kafka"
)

// OracleStats holds counters for observability and replay-oriented auditing.
// Replay: cross-check Kafka topic market.event with rows in market_resolutions (resolver_ref, evidence.dispatch).
type OracleStats struct {
	PollsTotal         atomic.Uint64
	PollErrorsTotal    atomic.Uint64
	FrozenSkipsTotal   atomic.Uint64
	PublishOKTotal     atomic.Uint64
	PublishFailTotal   atomic.Uint64

	LastPollStartedMs  atomic.Int64
	LastPollFinishedMs atomic.Int64
	LastPollEligible   atomic.Uint64
	LastPollProcessed  atomic.Uint64
	LastPollErr        atomic.Value // string, empty if ok

	mu sync.Mutex
	lastPollErrDetail string
}

func (s *OracleStats) recordPollStart() {
	s.PollsTotal.Add(1)
	s.LastPollStartedMs.Store(time.Now().UnixMilli())
}

func (s *OracleStats) recordFrozenSkip() {
	s.FrozenSkipsTotal.Add(1)
}

func (s *OracleStats) recordPollEnd(eligible, processed int, pollErr error) {
	s.LastPollFinishedMs.Store(time.Now().UnixMilli())
	s.LastPollEligible.Store(uint64(eligible))
	s.LastPollProcessed.Store(uint64(processed))
	if pollErr != nil {
		s.PollErrorsTotal.Add(1)
		s.LastPollErr.Store(pollErr.Error())
		s.mu.Lock()
		s.lastPollErrDetail = pollErr.Error()
		s.mu.Unlock()
		return
	}
	s.LastPollErr.Store("")
	s.mu.Lock()
	s.lastPollErrDetail = ""
	s.mu.Unlock()
}

func (s *OracleStats) recordPublishOK() {
	s.PublishOKTotal.Add(1)
}

func (s *OracleStats) recordPublishFail() {
	s.PublishFailTotal.Add(1)
}

// Snapshot returns a stable JSON-serializable view for /debug/oracle and operators.
func (s *OracleStats) Snapshot() map[string]any {
	out := map[string]any{
		"polls_total":          s.PollsTotal.Load(),
		"poll_errors_total":    s.PollErrorsTotal.Load(),
		"frozen_skips_total":   s.FrozenSkipsTotal.Load(),
		"publish_ok_total":     s.PublishOKTotal.Load(),
		"publish_fail_total":   s.PublishFailTotal.Load(),
		"last_poll_started_ms": s.LastPollStartedMs.Load(),
		"last_poll_finished_ms": s.LastPollFinishedMs.Load(),
		"last_poll_eligible":   s.LastPollEligible.Load(),
		"last_poll_processed":  s.LastPollProcessed.Load(),
	}
	if v := s.LastPollErr.Load(); v != nil {
		if str, ok := v.(string); ok && str != "" {
			out["last_poll_error"] = str
		}
	}
	s.mu.Lock()
	detail := s.lastPollErrDetail
	s.mu.Unlock()
	if detail != "" {
		out["last_poll_error_detail"] = detail
	}
	return out
}

// startObservabilityHTTP serves GET /healthz and GET /debug/oracle when addr is non-empty.
func startObservabilityHTTP(ctx context.Context, logger *slog.Logger, addr string, cfg config.ServiceConfig, topics sharedkafka.Topics, stats *OracleStats) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "oracle",
		})
	})
	mux.HandleFunc("/debug/oracle", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := map[string]any{
			"service":       cfg.Name,
			"env":           cfg.Env,
			"kafka_brokers": cfg.KafkaBrokers,
			"market_topic":  topics.MarketEvent,
			"poll_interval": oraclePollInterval(cfg).String(),
			"binance_base":  redactBinanceBase(binanceBaseURL()),
			"replay": map[string]any{
				"market_event_topic": topics.MarketEvent,
				"evidence_table":     "market_resolutions",
				"note":               "Reprocess by replaying Kafka from committed offset; compare resolver_ref and evidence.dispatch with emitted events.",
			},
		}
		if stats != nil {
			payload["stats"] = stats.Snapshot()
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
	})

	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		logger.Info("oracle observability HTTP listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("oracle observability HTTP failed", "err", err)
		}
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
}

func redactBinanceBase(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return raw
}

func observabilityHTTPAddr() string {
	v := strings.TrimSpace(os.Getenv("FUNNYOPTION_ORACLE_HTTP_ADDR"))
	if v != "" {
		return v
	}
	return ":9191"
}
