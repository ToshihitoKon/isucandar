package parallel

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallel(t *testing.T) {
	parallel := NewParallel(2)
	defer parallel.Close()

	pcount := int32(0)
	pmcount := int32(0)
	exited := uint32(0)
	f := func(_ context.Context) {
		atomic.AddInt32(&pcount, 1)
		defer atomic.AddInt32(&pcount, -1)
		time.Sleep(10 * time.Millisecond)
	}

	ctx := context.TODO()

	parallel.Do(ctx, f)
	go func() {
		parallel.Do(ctx, f)
		parallel.Do(ctx, f)
		parallel.Do(ctx, f)
	}()

	go func() {
		for atomic.LoadUint32(&exited) == 0 {
			m := atomic.LoadInt32(&pcount)
			if atomic.LoadInt32(&pmcount) < m {
				atomic.StoreInt32(&pmcount, m)
			}
		}
	}()

	parallel.Wait()
	atomic.StoreUint32(&exited, 1)

	maxCount := atomic.LoadInt32(&pmcount)
	if maxCount != 2 {
		t.Fatalf("Invalid parallel count: %d / %d", maxCount, 2)
	}
}

func TestParallelClosed(t *testing.T) {
	parallel := NewParallel(2)
	parallel.Close()

	ctx := context.TODO()

	called := false
	err := parallel.Do(ctx, func(_ context.Context) {
		called = true
	})

	parallel.Wait()

	if err == nil || err != ErrLimiterClosed {
		t.Fatalf("missmatch error: %+v", err)
	}

	if called {
		t.Fatalf("Do not process on closed")
	}
}

func TestParallelUnlimited(t *testing.T) {
	parallel := NewParallel(0)

	if parallel.limit != 0 {
		t.Fatalf("Invalid limit: %d", parallel.limit)
	}
}

func TestParallelCanceled(t *testing.T) {
	parallel := NewParallel(0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	parallel.Do(ctx, func(_ context.Context) {
		t.Fatal("Do not call")
	})

	parallel.Wait()
}

func TestParallelPanicOnNegative(t *testing.T) {
	parallel := NewParallel(0)

	var err interface{}
	func() {
		defer func() { err = recover() }()
		parallel.done(parallel.state)
	}()

	if err != ErrNegativeCount {
		t.Fatal(err)
	}
}

func TestParallelSetParallelism(t *testing.T) {
	parallel := NewParallel(0)

	check := func(expectTime time.Duration) {
		parallel.Reset()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		pcount := int32(0)
		pmcount := int32(0)
		exited := uint32(0)
		f := func(c context.Context) {
			atomic.AddInt32(&pcount, 1)
			defer atomic.AddInt32(&pcount, -1)

			time.Sleep(10 * time.Millisecond)
		}

		parallel.Do(ctx, f)
		go func() {
			parallel.Do(ctx, f)
			parallel.Do(ctx, f)
			parallel.Do(ctx, f)
		}()

		go func() {
			for atomic.LoadUint32(&exited) == 0 {
				m := atomic.LoadInt32(&pcount)
				if atomic.LoadInt32(&pmcount) < m {
					atomic.StoreInt32(&pmcount, m)
				}
			}
		}()
		parallel.Wait()
		atomic.StoreUint32(&exited, 1)

		maxCount := atomic.LoadInt32(&pmcount)
		if maxCount != parallel.CurrentLimit() && parallel.CurrentLimit() > 0 {
			t.Fatalf("Invalid parallel count: %d / %d", maxCount, parallel.CurrentLimit())
		}

		parallel.Wait()
	}

	parallel.SetParallelism(2)
	check(3 * time.Millisecond)

	parallel.AddParallelism(-1)
	check(6 * time.Millisecond)

	parallel.AddParallelism(-1)
	unlimitedCount := 4 / runtime.GOMAXPROCS(0)
	if unlimitedCount <= 0 {
		unlimitedCount = 1
	}
	check(time.Duration(unlimitedCount)*time.Millisecond + (time.Duration(unlimitedCount) * 500 * time.Microsecond))
}

func BenchmarkParallel(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	parallel := NewParallel(-1)
	nop := func(_ context.Context) {}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parallel.Do(ctx, nop)
	}
	parallel.Wait()
	b.StopTimer()
}
