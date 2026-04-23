package memory

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockEvictor struct {
	CallCount int32
}

func (m *MockEvictor) EvictStaleEpisodes(ctx context.Context, tenantID string, maxAgeDays int) error {
	atomic.AddInt32(&m.CallCount, 1)
	return nil
}

func TestReconciler_StartAndCancel(t *testing.T) {
	t.Parallel()

	evictor := &MockEvictor{}
	// Use a very short interval
	rec := NewReconciler(evictor, 5*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	
	// Run start in background
	done := make(chan struct{})
	go func() {
		rec.Start(ctx, []string{"t1", "t2"}, 30)
		close(done)
	}()

	// Wait long enough for at least one tick
	time.Sleep(20 * time.Millisecond)
	cancel()

	// Wait for Start to exit
	<-done

	calls := atomic.LoadInt32(&evictor.CallCount)
	// Expect calls to be made for both tenants (must be multiple of 2)
	assert.GreaterOrEqual(t, calls, int32(2))
	assert.Equal(t, int32(0), calls%2, "calls should be even since there are 2 tenants")
}
