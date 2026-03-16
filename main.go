package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dependency-track-exporter/internal/client"
	"dependency-track-exporter/internal/config"
	"dependency-track-exporter/internal/render"
	"dependency-track-exporter/internal/store"
	"dependency-track-exporter/internal/version"
)

func main() {
	cfg, err := config.Parse(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if cfg.ExitCode >= 0 {
		os.Exit(cfg.ExitCode)
	}

	logger := config.NewLogger(cfg.LogFormat, cfg.LogLevel, os.Stdout)
	logger.Info("starting dependency-track-exporter",
		"version", version.String(),
		"listen", cfg.WebListenAddress,
		"metrics_path", cfg.WebMetricsPath,
		"dtrack_address", cfg.DTrackAddress,
		"poll_period", cfg.PollPeriod.String(),
		"client_request_timeout", cfg.ClientRequestTimeout.String(),
	)

	dtrackClient := client.NewClient(cfg.DTrackAddress, cfg.DTrackAPIKey, cfg.ClientRequestTimeout, logger)
	metricsStore := store.NewStore()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	poller := store.NewPoller(logger, dtrackClient, metricsStore, cfg.PollPeriod)
	go poller.Run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc(cfg.WebMetricsPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		render.WriteMetrics(w, metricsStore.Snapshot())
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "dependency-track-exporter %s\n", version.String())
		fmt.Fprintf(w, "metrics: %s\n", cfg.WebMetricsPath)
	})

	server := &http.Server{
		Addr:              cfg.WebListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("http server failed", "err", err)
		os.Exit(1)
	}

	logger.Info("shutdown complete")
}
