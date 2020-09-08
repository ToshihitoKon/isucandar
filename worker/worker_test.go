package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {
	errOpt := func(_ *Worker) error {
		return errors.New("invalid")
	}

	worker, err := NewWorker(nil, errOpt)
	if err == nil || worker != nil {
		t.Fatal("error not occured")
	}

	worker, err = NewWorker(nil, WithLoopCount(1))
	if err != nil {
		t.Fatal(err)
	}

	worker.Process(context.Background())
}

func TestWorkerLimited(t *testing.T) {
	pool := []int{}
	mu := sync.Mutex{}
	f := func(_ context.Context, i int) {
		mu.Lock()
		pool = append(pool, i)
		mu.Unlock()
	}

	worker, err := NewWorker(f, WithLoopCount(5), WithUnlimitedParallelism())
	if err != nil {
		t.Fatal(err)
	}

	worker.Process(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(pool) != 5 {
		t.Fatalf("executed count is missmatch: %d", len(pool))
	}
}

func TestWorkerLimitedCancel(t *testing.T) {
	f := func(_ context.Context, _ int) {
		<-time.After(100 * time.Millisecond)
	}

	worker, err := NewWorker(f, WithLoopCount(100), WithMaxParallelism(1))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	now := time.Now()
	worker.Process(ctx)
	diff := time.Now().Sub(now)

	if diff > 1*time.Second {
		t.Fatalf("Executed all with %s", diff)
	}
}

func TestWorkerLimitedCanceled(t *testing.T) {
	n := int32(0)
	count := &n
	f := func(_ context.Context, _ int) {
		atomic.AddInt32(count, 1)
		<-time.After(100 * time.Millisecond)
	}

	worker, err := NewWorker(f, WithLoopCount(100), WithMaxParallelism(1))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	cancel()

	worker.Process(ctx)

	if n := atomic.LoadInt32(count); n > 0 {
		t.Fatalf("Executed count: %d", n)
	}
}

func TestWorkerUnlimited(t *testing.T) {
	n := int32(0)
	count := &n
	f := func(_ context.Context, i int) {
		atomic.AddInt32(count, 1)
	}

	worker, err := NewWorker(f, WithInfinityLoop(), WithMaxParallelism(100))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	worker.Process(ctx)

	if atomic.LoadInt32(count) == 0 {
		t.Fatalf("worker not executed")
	}
}

func TestWorkerUnlimitedCanceled(t *testing.T) {
	n := int32(0)
	count := &n
	f := func(_ context.Context, i int) {
		atomic.AddInt32(count, 1)
	}

	worker, err := NewWorker(f, WithInfinityLoop(), WithMaxParallelism(100))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	worker.Process(ctx)

	if n := atomic.LoadInt32(count); n > 0 {
		t.Fatalf("Executed count: %d", n)
	}
}
