package concun

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestProcess(t *testing.T) {
	pool, err := NewPool[int, int](1, func(ctx context.Context, r int) (*int, error) {
		temp := r + 1
		return &temp, nil
	})
	if err != nil {
		t.Errorf("Error creating pool: %s", err)
	}

	process, err := pool.Process(1)
	if err != nil {
		t.Errorf("Error processing request: %s", err)
	}

	if *process != 2 {
		t.Errorf("Expected 2, got %d", *process)
	}

	if pool.QueueLength() != 0 {
		t.Errorf("Expected 0, got %d", pool.QueueLength())
	}

	pool.Close()
	if pool.IsRunning() {
		t.Errorf("pool is running after close call")
	}
}

func TestProcessMany(t *testing.T) {
	pool, err := NewPool[int, int](1, func(ctx context.Context, r int) (*int, error) {
		time.Sleep(500 * time.Millisecond)
		temp := r + 1
		return &temp, nil
	})
	if err != nil {
		t.Errorf("Error creating pool: %s", err)
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			wg.Done()
			_, _ = pool.Process(1)
		}()
	}

	wg.Wait()

	if pool.QueueLength() == 0 {
		t.Errorf("Expected 10, got %d", pool.QueueLength())
	}

	pool.Close()

	if pool.QueueLength() != 0 {
		t.Errorf("Expected 0, got %d", pool.QueueLength())
	}

	if pool.IsRunning() {
		t.Errorf("pool is running after close call")
	}
}
