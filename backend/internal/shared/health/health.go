package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// ListenAndServe starts a minimal HTTP server exposing GET /healthz.
// It shuts down gracefully when ctx is cancelled.
// If addr is empty the call is a no-op.
func ListenAndServe(ctx context.Context, logger *slog.Logger, addr, service, env string) {
	if addr == "" {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": service,
			"env":     env,
		})
	})

	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		logger.Info("health HTTP started", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health HTTP server error", "err", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		server.Shutdown(shutCtx)
	}()
}
