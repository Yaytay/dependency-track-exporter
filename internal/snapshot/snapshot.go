package snapshot

import (
	"context"
	"sync/atomic"
	"time"

	"dependency-track-exporter/internal/client"
	"dependency-track-exporter/internal/config"
)

type Snapshot struct {
	GeneratedAt time.Time
	Projects    []client.ProjectSnapshot
	LastError   string
	Up          bool
}

type Store struct {
	current atomic.Value
}

func NewStore() *Store {
	s := &Store{}
	s.current.Store(Snapshot{
		GeneratedAt: time.Now().UTC(),
		Projects:    nil,
		LastError:   "initial refresh not completed",
		Up:          false,
	})
	return s
}

func (s *Store) Snapshot() Snapshot {
	return s.current.Load().(Snapshot)
}

func (s *Store) Replace(snapshot Snapshot) {
	s.current.Store(snapshot)
}

type Poller struct {
	logger *config.Logger
	client *client.Client
	store  *Store
	period time.Duration
}

func NewPoller(logger *config.Logger, client *client.Client, store *Store, period time.Duration) *Poller {
	return &Poller{
		logger: logger,
		client: client,
		store:  store,
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
	p.logger.Info("starting snapshot refresh")

	snapshots, err := p.client.FetchProjectSnapshots(ctx)
	if err != nil {
		p.logger.Error("refresh failed", "err", err)
		p.store.Replace(Snapshot{
			GeneratedAt: time.Now().UTC(),
			Projects:    nil,
			LastError:   err.Error(),
			Up:          false,
		})
		return
	}

	p.store.Replace(Snapshot{
		GeneratedAt: time.Now().UTC(),
		Projects:    snapshots,
		LastError:   "",
		Up:          true,
	})

	p.logger.Info("refresh complete",
		"projects", len(snapshots),
		"duration", time.Since(start).String(),
	)
}
