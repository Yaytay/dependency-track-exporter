package store

import (
	"context"
	"sync/atomic"
	"time"

	"dependency-track-exporter/internal/client"
	"dependency-track-exporter/internal/config"
	"dependency-track-exporter/internal/snapshot"
)

type Store struct {
	current atomic.Value
}

func NewStore() *Store {
	s := &Store{}
	s.current.Store(snapshot.Snapshot{
		GeneratedAt: time.Now().UTC(),
		Projects:    nil,
		LastError:   "initial refresh not completed",
		Up:          false,
	})
	return s
}

func (s *Store) Snapshot() snapshot.Snapshot {
	return s.current.Load().(snapshot.Snapshot)
}

func (s *Store) Replace(next snapshot.Snapshot) {
	s.current.Store(next)
}

type Poller struct {
	logger *config.Logger
	client *client.Client
	store  *Store
	period time.Duration
}

func NewPoller(logger *config.Logger, dtrackClient *client.Client, metricsStore *Store, period time.Duration) *Poller {
	return &Poller{
		logger: logger,
		client: dtrackClient,
		store:  metricsStore,
		period: period,
	}
}

func (p *Poller) Run(ctx context.Context) {
	p.refresh(ctx)

	ticker := time.NewTicker(p.period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.refresh(ctx)
		}
	}
}

func (p *Poller) refresh(ctx context.Context) {
	start := time.Now()
	p.logger.Info("starting refresh")

	projectSnapshots, err := p.client.FetchProjectSnapshots(ctx)
	if err != nil {
		p.logger.Error("refresh failed", "err", err)
		p.store.Replace(snapshot.Snapshot{
			GeneratedAt: time.Now().UTC(),
			Projects:    nil,
			LastError:   err.Error(),
			Up:          false,
		})
		return
	}

	p.store.Replace(snapshot.Snapshot{
		GeneratedAt: time.Now().UTC(),
		Projects:    projectSnapshots,
		LastError:   "",
		Up:          true,
	})

	p.logger.Info(
		"refresh complete",
		"projects", len(projectSnapshots),
		"duration", time.Since(start).String(),
	)
}
