package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewWorkerPool(t *testing.T) {
	ctx := context.Background()
	wp := NewWorkerPool(ctx, 10)

	require.NotNil(t, wp)
	require.Equal(t, 10, cap(wp.jobsChan))
	require.Equal(t, 10, cap(wp.resChan))
	require.Equal(t, 0, len(wp.workers))
	require.Equal(t, 1, wp.workerID)
}

func TestAddWorker(t *testing.T) {
	ctx := context.Background()
	wp := NewWorkerPool(ctx, 5)

	wp.AddWorker()
	wp.mu.Lock()
	defer wp.mu.Unlock()

	require.Equal(t, 1, len(wp.workers))
	require.NotNil(t, wp.workers[1])
}

func TestRemoveWorker(t *testing.T) {
	ctx := context.Background()
	wp := NewWorkerPool(ctx, 5)

	wp.AddWorker()
	wp.RemoveWorker()

	wp.mu.Lock()
	defer wp.mu.Unlock()
	require.Equal(t, 0, len(wp.workers))
}

func TestShutdown(t *testing.T) {
	ctx := context.Background()
	wp := NewWorkerPool(ctx, 5)

	wp.AddWorker()
	wp.AddWorker()
	go func() {
		for i := 0; i < 5; i++ {
			wp.jobsChan <- fmt.Sprintf("filename_%d.go", i)
		}
		close(wp.jobsChan)
	}()

	wp.wg.Wait()
	wp.Shutdown()

	wp.mu.Lock()
	defer wp.mu.Unlock()
	require.Equal(t, 0, len(wp.workers))
}

func TestWorkerPool_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wp := NewWorkerPool(ctx, 10)
	wp.AddWorker()
	wp.AddWorker()

	var wg sync.WaitGroup
	jobCount := 6
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < jobCount; i++ {
			select {
			case wp.jobsChan <- fmt.Sprintf("filename_%d.go", i):
			case <-ctx.Done():
				return
			}
		}
		close(wp.jobsChan)
	}()

	go func() {
		wg.Wait()
		wp.wg.Wait()
		close(wp.resChan)
	}()

	results := []string{}
	for res := range wp.resChan {
		results = append(results, res)
	}

	wp.Shutdown()

	require.Equal(t, jobCount, len(results))

	expectedResults := []string{
		"Result for job: filename_0.go",
		"Result for job: filename_1.go",
		"Result for job: filename_2.go",
		"Result for job: filename_3.go",
		"Result for job: filename_4.go",
		"Result for job: filename_5.go",
	}

	require.ElementsMatch(t, results, expectedResults)
}
