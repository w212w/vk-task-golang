package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	jobsBuffer int           = 100 // Буфер канала задач
	jobTasks   int           = 20  // Количество задач
	timeout    time.Duration = 10  // Таймаут для контекста
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	wp := NewWorkerPool(ctx, jobsBuffer)
	done := make(chan struct{})

	wp.AddWorker()
	wp.AddWorker()
	wp.AddWorker()
	wp.AddWorker()

	wp.RemoveWorker()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(wp.jobsChan)
		for i := 0; i < jobTasks; i++ {
			select {
			case wp.jobsChan <- fmt.Sprintf("filename_%d.go", i):
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer close(done)
		wg.Wait()
		wp.wg.Wait()
		close(wp.resChan)
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Context cancelled")
		wp.Shutdown()
	case <-done:
		fmt.Println("All workers done or removed")
		wp.Shutdown()
	}

	for res := range wp.resChan {
		fmt.Println(res)
	}
}
