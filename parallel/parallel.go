package parallel

import (
	"context"
	"errors"
	"sync/atomic"
)

var (
	ErrLimiterClosed = errors.New("limiter closed")
	ErrNegativeCount = errors.New("negative count")
)

const (
	closedFalse uint32 = iota
	closedTrue
)

type Parallel struct {
	ctx    context.Context
	limit  int32
	count  int32
	closed uint32
	closer chan struct{}
}

func NewParallel(ctx context.Context, limit int32) *Parallel {
	p := &Parallel{
		ctx:    ctx,
		limit:  limit,
		count:  0,
		closed: closedFalse,
		closer: make(chan struct{}),
	}

	return p
}

func (l *Parallel) CurrentLimit() int32 {
	return atomic.LoadInt32(&l.limit)
}

func (l *Parallel) Do(f func(context.Context)) error {
	atomic.AddInt32(&l.count, 1)

	err := l.start()
	if err != nil {
		atomic.AddInt32(&l.count, -1)
		return err
	}

	go func() {
		defer l.done()
		f(l.ctx)
	}()

	return nil
}

func (l *Parallel) Wait() {
	if atomic.LoadUint32(&l.closed) != closedTrue {
		for {
			select {
			case <-l.ctx.Done():
				l.Close()
			case <-l.closer:
				return
			}
		}
	}
}

func (l *Parallel) Close() {
	if atomic.CompareAndSwapUint32(&l.closed, closedFalse, closedTrue) {
		close(l.closer)
	}
}

func (l *Parallel) SetParallelism(limit int32) {
	atomic.StoreInt32(&l.limit, limit)
}

func (l *Parallel) AddParallelism(limit int32) {
	atomic.AddInt32(&l.limit, limit)
}

func (l *Parallel) start() error {
	for l.isRunning() {
		if count, kept := l.isLimitKept(); kept {
			if atomic.CompareAndSwapInt32(&l.count, count, count+1) {
				return nil
			}
		}
	}

	return ErrLimiterClosed
}

func (l *Parallel) done() {
	count := atomic.AddInt32(&l.count, -2)
	if count < 0 {
		panic(ErrNegativeCount)
	}
	if count == 0 {
		l.Close()
	}
}

func (l *Parallel) isRunning() bool {
	return atomic.LoadUint32(&l.closed) == closedFalse
}

func (l *Parallel) isLimitKept() (int32, bool) {
	limit := atomic.LoadInt32(&l.limit)
	count := atomic.LoadInt32(&l.count)
	return count, limit < 1 || count < (limit*2)
}
