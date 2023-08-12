package util_context

import (
	"context"
	"io"
	"sync/atomic"
	"time"
)

type ResetFunc func()

type AtomicTime struct {
	underlying atomic.Pointer[time.Time]
}

func NewAtomicTime(t0 time.Time) *AtomicTime {

	result := AtomicTime{
		underlying: atomic.Pointer[time.Time]{},
	}

	result.Set(t0)

	return &result
}

func (at *AtomicTime) Get() time.Time {
	return *at.underlying.Load()
}

func (at *AtomicTime) Set(t time.Time) {
	at.underlying.Store(&t)
}

const DefaultCheckIntervalMillis = 100

type ResetWriter interface {
	io.Writer
}

type resetWriter struct {
	w         io.Writer
	resetFunc ResetFunc
}

func (rw resetWriter) Write(p []byte) (n int, err error) {
	rw.resetFunc()
	return rw.w.Write(p)
}

func NewResetWriter(w io.Writer, resetFunc ResetFunc) ResetWriter {
	return resetWriter{
		w:         w,
		resetFunc: resetFunc,
	}
}

func NewResetWriterCh(w io.Writer, resetChan chan any) ResetWriter {
	return NewResetWriter(w, func() { resetChan <- struct{}{} })
}

func WithTimeoutAndReset(
	parent context.Context,
	timeout time.Duration,
) (context.Context, context.CancelFunc, ResetFunc) {
	return WithTimeoutAndResetC(parent, timeout, DefaultCheckIntervalMillis)
}

func WithTimeoutAndResetC(
	parent context.Context,
	timeout time.Duration,
	checkInterval time.Duration,
) (context.Context, context.CancelFunc, ResetFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	ctx, cancel := context.WithCancel(parent)

	lastUpdateAtm := NewAtomicTime(time.Now())

	resetFunc := func() {
		lastUpdateAtm.Set(time.Now())
	}

	// Start a go routine which times out the context after the given timeout
	// The timer should reset if ResetFunc is called
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond * checkInterval): // or something useful
				if time.Now().Sub(lastUpdateAtm.Get()) > timeout {
					cancel()
					return
				}
			}
		}
	}()

	return ctx, cancel, resetFunc
}
