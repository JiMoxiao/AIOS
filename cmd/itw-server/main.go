package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aios/internal/config"
	aioshttp "aios/internal/http"
	"aios/internal/providers"
	"aios/internal/providers/mock"
	"aios/internal/providers/openai"
	"aios/internal/router"
	"aios/internal/store"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx := context.Background()

	st, err := store.OpenSQLite(ctx, store.SQLiteStoreConfig{
		DSN:      cfg.DBDSN,
		SaveBody: cfg.SaveBody,
	})
	if err != nil {
		slog.Error("store_open_failed", "err", err)
		os.Exit(1)
	}
	defer func() { _ = st.Close() }()

	var p providers.Provider
	if cfg.OpenAIBaseURL != "" {
		p = &openai.HTTPProvider{
			BaseURL: cfg.OpenAIBaseURL,
			APIKey:  cfg.OpenAIAPIKey,
			Client:  &http.Client{Timeout: 120 * time.Second},
		}
	} else {
		p = &mock.Provider{
			Fall: mock.Step{Response: providers.ChatResponse{Content: "AIOS provider not configured."}},
		}
	}

	r := router.Router{
		Catalog: router.Catalog{
			Models: []router.Model{
				{ID: "premium", QualityScore: 1.0, CostScore: 1.0, LatencyScore: 0.6, ErrorRateScore: 0.1, MaxContextTokens: 200000},
				{ID: "balanced", QualityScore: 0.8, CostScore: 0.4, LatencyScore: 0.4, ErrorRateScore: 0.2, MaxContextTokens: 64000},
				{ID: "claude-sonnet", QualityScore: 0.95, CostScore: 0.9, LatencyScore: 0.6, ErrorRateScore: 0.15, MaxContextTokens: 200000},
				{ID: "gpt-5", QualityScore: 0.98, CostScore: 1.0, LatencyScore: 0.7, ErrorRateScore: 0.15, MaxContextTokens: 200000},
			},
		},
	}

	srv := aioshttp.NewServer(aioshttp.ServerConfig{
		Store:    st,
		Provider: p,
		Router:   r,
	})
	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("server_listen", "addr", cfg.HTTPAddr)
		err := httpSrv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			slog.Error("server_error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}
