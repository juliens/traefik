package engine

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInstance struct {
	slot   int
	closed atomic.Bool
}

func (f *fakeInstance) Close(context.Context) error {
	f.closed.Store(true)

	return nil
}

func TestPool_PickRoundRobin(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	p, err := NewPool(t.Context(), 3, func(_ context.Context, slot int) (Instance, error) {
		return &fakeInstance{slot: slot}, nil
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close(context.Background()) })

	var seen []int
	for range 6 {
		slot, err := p.Pick()
		require.NoError(t, err)
		seen = append(seen, slot.Index())
	}

	assert.Equal(t, []int{1, 2, 0, 1, 2, 0}, seen)
	assert.Equal(t, Stats{LiveInstances: 3}, p.Stats())
}

func TestPool_FailsFast(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	var created []*fakeInstance

	_, err := NewPool(t.Context(), 3, func(_ context.Context, slot int) (Instance, error) {
		if slot == 1 {
			return nil, errors.New("boom")
		}
		inst := &fakeInstance{slot: slot}
		created = append(created, inst)

		return inst, nil
	}, &logger)
	require.EqualError(t, err, "creating instance 1: boom")

	require.Len(t, created, 1)
	assert.True(t, created[0].closed.Load())
}

func TestPool_Replace(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	// failures is the number of creations that fail before one succeeds.
	var failures atomic.Int32

	p, err := NewPool(t.Context(), 1, func(_ context.Context, slot int) (Instance, error) {
		if failures.Add(-1) >= 0 {
			return nil, errors.New("not yet")
		}

		return &fakeInstance{slot: slot}, nil
	}, &logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close(context.Background()) })

	slot, err := p.Pick()
	require.NoError(t, err)
	first := slot.Instance().(*fakeInstance)

	failures.Store(1)
	p.Replace(slot, errors.New("trap"))

	_, err = p.Pick()
	require.ErrorIs(t, err, ErrNoInstance)

	require.Eventually(t, func() bool {
		return slot.Instance() != nil
	}, 5*time.Second, 10*time.Millisecond)

	assert.True(t, first.closed.Load())
	assert.NotSame(t, first, slot.Instance())
}

func TestPool_CloseDrains(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()
	p, err := NewPool(t.Context(), 1, func(_ context.Context, slot int) (Instance, error) {
		return &fakeInstance{slot: slot}, nil
	}, &logger)
	require.NoError(t, err)

	require.NoError(t, p.Enter())

	var wg sync.WaitGroup
	wg.Add(1)
	closed := make(chan error, 1)
	go func() {
		defer wg.Done()
		closed <- p.Close(context.Background())
	}()

	require.Eventually(t, func() bool {
		if p.Enter() != nil {
			return true
		}
		p.Leave()

		return false
	}, time.Second, 5*time.Millisecond)

	select {
	case err := <-closed:
		t.Fatalf("Close returned with a request in flight: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	p.Leave()
	wg.Wait()
	require.NoError(t, <-closed)

	_, err = p.Pick()
	require.ErrorIs(t, err, ErrClosing)
}
