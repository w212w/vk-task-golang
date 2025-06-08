package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	jobsBuffer int           = 5  // Буфер канала задач
	jobTasks   int           = 5  // Количество задач
	timeout    time.Duration = 10 // Таймаут для контекста
)

type Worker struct {
	id     int
	stopCh chan struct{}
}

type WorkerPool struct {
	ctx      context.Context
	jobsChan chan string
	resChan  chan string
	workers  map[int]*Worker
	workerID int
	mu       sync.Mutex
	wg       sync.WaitGroup
}

func NewWorkerPool(ctx context.Context, jobsBuffer int) *WorkerPool {
	return &WorkerPool{
		ctx:      ctx,
		jobsChan: make(chan string, jobsBuffer),
		resChan:  make(chan string, jobsBuffer),
		workers:  make(map[int]*Worker),
		workerID: 1,
	}
}

// Функция для добавления воркера в воркерпул
func (wp *WorkerPool) AddWorker() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	id := wp.workerID
	wp.workerID++

	worker := &Worker{
		id:     id,
		stopCh: make(chan struct{}),
	}
	wp.workers[id] = worker
	fmt.Printf("Worker: %d was added\n", worker.id)
	wp.runWorker(worker)
}

// Функция для запуска воркера
func (wp *WorkerPool) runWorker(wrk *Worker) {
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		for {
			select {
			case job, ok := <-wp.jobsChan:
				if !ok {
					fmt.Printf("Worker: %d is stopping, jobs channel closed\n", wrk.id)
					return
				}
				fmt.Printf("Worker: %d is starting job: %s\n", wrk.id, job)
				msg := fmt.Sprintf("Result for job: %s", job)
				time.Sleep(3 * time.Second) // симуляция работы воркера
				wp.resChan <- msg
				fmt.Printf("Worker: %d is finished job: %s\n", wrk.id, job)
			case <-wp.ctx.Done():
				fmt.Printf("Worker: %d was stopped by Context\n", wrk.id)
				return
			case <-wrk.stopCh:
				fmt.Printf("Worker: %d was stopped by Remove function\n", wrk.id)
				return

			}
		}
	}()
}

// Функция для остановки и удаления воркера из воркерпула
func (wp *WorkerPool) RemoveWorker() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) == 0 {
		fmt.Printf("All workers are closed\n")
		return
	}
	for id, worker := range wp.workers {
		close(worker.stopCh)
		fmt.Printf("Worker: %d is stopped\n", id)
		delete(wp.workers, id)
		fmt.Printf("Worker: %d is removed from workerpool\n", id)
		break
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	wp := NewWorkerPool(ctx, jobsBuffer)
	wp.AddWorker()
	wp.AddWorker()
	wp.AddWorker()
	wp.RemoveWorker()
	wp.RemoveWorker()
	wp.AddWorker()
	wp.AddWorker()

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

	done := make(chan struct{})

	go func() {
		defer close(done)
		wg.Wait()
		wp.wg.Wait()
		close(wp.resChan)
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Context cancelled")
	case <-done:
		fmt.Println("All workers done")
	}

	for res := range wp.resChan {
		fmt.Println(res)
	}
}
