package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

const (
	minReplaceBackoff = 100 * time.Millisecond
	maxReplaceBackoff = 5 * time.Second
)

// Instance is a guest instance managed by a Pool.
type Instance interface {
	Close(ctx context.Context) error
}

// Factory creates the instance of a slot.
type Factory func(ctx context.Context, slot int) (Instance, error)

// Slot holds one instance of the pool.
type Slot struct {
	idx  int
	mu   sync.Mutex
	inst Instance
	dead bool
}

// Index returns the position of the slot in the pool.
func (s *Slot) Index() int {
	return s.idx
}

// Instance returns the instance of the slot, or nil when it is dead.
func (s *Slot) Instance() Instance {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dead {
		return nil
	}

	return s.inst
}

// Pool keeps a fixed number of long-lived instances, dispatches requests to them round-robin and replaces the
// ones that stop.
type Pool struct {
	factory Factory
	logger  *zerolog.Logger
	slots   []*Slot

	rr       atomic.Uint32
	inFlight atomic.Int64
	closing  atomic.Bool
	drained  chan struct{}
	wg       sync.WaitGroup
}

// NewPool creates n instances; it fails if any of them cannot be created.
func NewPool(ctx context.Context, n int, factory Factory, logger *zerolog.Logger) (*Pool, error) {
	p := &Pool{factory: factory, logger: logger, drained: make(chan struct{})}

	for i := range n {
		inst, err := factory(ctx, i)
		if err != nil {
			_ = p.Close(ctx)

			return nil, fmt.Errorf("creating instance %d: %w", i, err)
		}

		p.slots = append(p.slots, &Slot{idx: i, inst: inst})
	}

	return p, nil
}

// Pick returns a live slot, round-robin.
func (p *Pool) Pick() (*Slot, error) {
	if p.closing.Load() {
		return nil, ErrClosing
	}

	n := len(p.slots)
	start := int(p.rr.Add(1))
	for i := range n {
		slot := p.slots[(start+i)%n]
		if slot.Instance() != nil {
			return slot, nil
		}
	}

	return nil, ErrNoInstance
}

// Enter registers an in-flight request; it fails when the pool is closing.
func (p *Pool) Enter() error {
	if p.closing.Load() {
		return ErrClosing
	}

	p.inFlight.Add(1)
	p.wg.Add(1)

	return nil
}

// Leave unregisters an in-flight request.
func (p *Pool) Leave() {
	p.inFlight.Add(-1)
	p.wg.Done()
}

// Replace marks the slot dead and re-creates its instance in the background, with a backoff on failures.
func (p *Pool) Replace(slot *Slot, cause error) {
	slot.mu.Lock()
	if slot.dead {
		slot.mu.Unlock()

		return
	}

	slot.dead = true
	old := slot.inst
	slot.inst = nil
	slot.mu.Unlock()

	p.logger.Error().Err(cause).Int("slot", slot.idx).Msg("Wasm instance stopped, replacing it")

	go func() {
		if old != nil {
			if err := old.Close(context.Background()); err != nil {
				p.logger.Debug().Err(err).Int("slot", slot.idx).Msg("Closing stopped wasm instance")
			}
		}

		backoff := minReplaceBackoff
		for !p.closing.Load() {
			inst, err := p.factory(context.Background(), slot.idx)
			if err == nil {
				slot.mu.Lock()
				slot.inst = inst
				slot.dead = false
				slot.mu.Unlock()

				p.logger.Debug().Int("slot", slot.idx).Msg("Wasm instance replaced")

				return
			}

			p.logger.Warn().Err(err).Int("slot", slot.idx).Dur("retryIn", backoff).Msg("Creating wasm instance")
			time.Sleep(backoff)
			backoff = min(backoff*2, maxReplaceBackoff)
		}
	}()
}

// Stats returns the number of in-flight requests and of live instances.
func (p *Pool) Stats() Stats {
	live := 0
	for _, slot := range p.slots {
		if slot.Instance() != nil {
			live++
		}
	}

	return Stats{InFlight: p.inFlight.Load(), LiveInstances: live}
}

// Close refuses new requests, waits for the in-flight ones until ctx is done, then closes every instance.
func (p *Pool) Close(ctx context.Context) error {
	if p.closing.Swap(true) {
		<-p.drained

		return nil
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	var errs []error
	select {
	case <-done:
	case <-ctx.Done():
		errs = append(errs, fmt.Errorf("waiting for in-flight requests: %w", ctx.Err()))
	}

	for _, slot := range p.slots {
		slot.mu.Lock()
		inst := slot.inst
		slot.inst = nil
		slot.dead = true
		slot.mu.Unlock()

		if inst == nil {
			continue
		}

		if err := inst.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("closing instance %d: %w", slot.idx, err))
		}
	}

	close(p.drained)

	return errors.Join(errs...)
}
