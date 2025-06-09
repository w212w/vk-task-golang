package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Worker struct {
	id     int
	stopCh chan struct{} // Канал для остановки
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
	wp := &WorkerPool{
		ctx:      ctx,
		jobsChan: make(chan string, jobsBuffer),
		resChan:  make(chan string, jobsBuffer),
		workers:  make(map[int]*Worker),
		workerID: 1,
	}

	return wp
}

func (wp *WorkerPool) Shutdown() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	fmt.Println("WorkerPool is shutting down...")

	for id, worker := range wp.workers {
		select {
		case <-worker.stopCh:
			// Канал уже закрыт
		default:
			close(worker.stopCh)
			fmt.Printf("Worker: %d is shutting down\n", id)
		}
	}

	wp.mu.Unlock()
	wp.wg.Wait()
	wp.mu.Lock()

	wp.workers = make(map[int]*Worker) // Очищаем мапу
	fmt.Println("WorkerPool shutdown completed")
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
				time.Sleep(2 * time.Second) // Cимуляция работы воркера
				wp.resChan <- msg
				fmt.Printf("Worker: %d has finished job: %s\n", wrk.id, job)
			case <-wp.ctx.Done():
				fmt.Printf("Worker: %d was stopped by Context...\n", wrk.id)
				return
			case <-wrk.stopCh:
				fmt.Printf("Worker: %d was stopped by stopCh...\n", wrk.id)
				return
			}
		}
	}()
}

// Функция для остановки и удаления рандомного воркера из воркерпула
func (wp *WorkerPool) RemoveWorker() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) == 0 {
		fmt.Printf("All workers are closed\n")
		return
	}
	for id, worker := range wp.workers {
		close(worker.stopCh)
		fmt.Printf("Worker: %d is stopping...\n", id)
		delete(wp.workers, id)
		fmt.Printf("Worker: %d is removed from workerpool\n", id)
		break
	}
}
