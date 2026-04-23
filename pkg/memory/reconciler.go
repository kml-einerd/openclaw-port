package memory

import (
	"context"
	"time"
)

// Evictor defines the interface for executing memory evictions across a database.
type Evictor interface {
	EvictStaleEpisodes(ctx context.Context, tenantID string, maxAgeDays int) error
}

// Reconciler is a background worker that periodically triggers auto-eviction
// of stale memory episodes across tenants.
//
// Adapted from openclaw/extensions/memory-core/retention.ts.
type Reconciler struct {
	evictor  Evictor
	interval time.Duration
}

// NewReconciler creates a new Episode Auto-Eviction Reconciler.
func NewReconciler(evictor Evictor, interval time.Duration) *Reconciler {
	return &Reconciler{
		evictor:  evictor,
		interval: interval,
	}
}

// Start begins the reconciliation loop. It blocks until context is canceled.
func (r *Reconciler) Start(ctx context.Context, tenants []string, maxAgeDays int) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, tenant := range tenants {
				// We intentionally ignore errors here to allow the loop to continue
				// for other tenants. In a real system, these would be logged via telemetry.
				_ = r.evictor.EvictStaleEpisodes(ctx, tenant, maxAgeDays)
			}
		}
	}
}
