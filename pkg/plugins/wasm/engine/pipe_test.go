package engine

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipe_GuestRead(t *testing.T) {
	t.Parallel()

	events := 0
	p := NewPipe(0, func() { events++ })

	buf := make([]byte, 8)

	n, err := p.GuestRead(buf)
	require.ErrorIs(t, err, ErrAgain)
	assert.Equal(t, 0, n)
	assert.Equal(t, 0, p.Readable())

	_, err = p.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, p.Readable())
	assert.Equal(t, 5, p.Buffered())
	assert.Equal(t, 1, events)

	n, err = p.GuestRead(buf)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(buf[:n]))
	assert.Equal(t, 0, p.Buffered())

	require.NoError(t, p.Close())
	assert.Equal(t, 1, p.Readable())

	n, err = p.GuestRead(buf)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestPipe_BoundedWriteBlocks(t *testing.T) {
	t.Parallel()

	p := NewPipe(4, nil)

	n, err := p.GuestWrite([]byte("abcdef"))
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.False(t, p.Writable())

	_, err = p.GuestWrite([]byte("x"))
	require.ErrorIs(t, err, ErrAgain)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := p.Write([]byte("ghij"))
		assert.NoError(t, err)
	}()

	select {
	case <-done:
		t.Fatal("Write returned while the pipe was full")
	case <-time.After(50 * time.Millisecond):
	}

	buf := make([]byte, 8)
	n, err = p.Read(t.Context(), buf)
	require.NoError(t, err)
	assert.Equal(t, "abcd", string(buf[:n]))

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Write did not resume after a read")
	}

	n, err = p.Read(t.Context(), buf)
	require.NoError(t, err)
	assert.Equal(t, "ghij", string(buf[:n]))
}

func TestPipe_CloseWithError(t *testing.T) {
	t.Parallel()

	p := NewPipe(0, nil)
	errBoom := errors.New("boom")

	require.NoError(t, p.CloseWithError(errBoom))

	_, err := p.Read(t.Context(), make([]byte, 1))
	require.ErrorIs(t, err, errBoom)

	_, err = p.GuestRead(make([]byte, 1))
	require.ErrorIs(t, err, errBoom)

	_, err = p.Write([]byte("x"))
	require.ErrorIs(t, err, errBoom)

	_, err = p.GuestWrite([]byte("x"))
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestPipe_ReadCancel(t *testing.T) {
	t.Parallel()

	p := NewPipe(0, nil)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := p.Read(ctx, make([]byte, 1))
	require.ErrorIs(t, err, context.Canceled)
}
