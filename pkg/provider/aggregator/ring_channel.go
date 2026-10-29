package aggregator

import (
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// RingChannel implements a channel in a way that never blocks the writer.
// Specifically, if a value is written to a RingChannel when its buffer is full then the oldest
// value in the buffer is discarded to make room (just like a standard ring-buffer).
// Note that Go's scheduler can cause discarded values when they could be avoided, simply by scheduling
// the writer before the reader, so caveat emptor.
// The buffer keeps a value per provider name: a provider plugin can run several providers (one per namespace),
// which must not discard the values of each other.
type RingChannel struct {
	input, output chan dynamic.Message
	buffer        map[string]dynamic.Message
	// names are the provider names of the buffer, the oldest first.
	names []string
}

func newRingChannel() *RingChannel {
	ch := &RingChannel{
		input:  make(chan dynamic.Message),
		output: make(chan dynamic.Message),
		buffer: map[string]dynamic.Message{},
	}
	go ch.ringBuffer()
	return ch
}

func (ch *RingChannel) in() chan<- dynamic.Message {
	return ch.input
}

func (ch *RingChannel) out() <-chan dynamic.Message {
	return ch.output
}

// for all buffered cases.
func (ch *RingChannel) ringBuffer() {
	var input, output chan dynamic.Message
	var next dynamic.Message
	input = ch.input

	for input != nil || output != nil {
		select {
		// Prefer to write if possible, which is surprisingly effective in reducing
		// dropped elements due to overflow. The naive read/write select chooses randomly
		// when both channels are ready, which produces unnecessary drops 50% of the time.
		case output <- next:
			ch.pop()
		default:
			select {
			case elem, open := <-input:
				if !open {
					input = nil
					break
				}

				if _, ok := ch.buffer[elem.ProviderName]; !ok {
					ch.names = append(ch.names, elem.ProviderName)
				}
				ch.buffer[elem.ProviderName] = elem
			case output <- next:
				ch.pop()
			}
		}

		if len(ch.names) == 0 {
			output = nil
			continue
		}

		output = ch.output
		next = ch.buffer[ch.names[0]]
	}

	close(ch.output)
}

func (ch *RingChannel) pop() {
	delete(ch.buffer, ch.names[0])
	ch.names = ch.names[1:]
}
