package utils

import (
	"sync"

	"github.com/massonsky/buffalo/pkg/errors"
)

// Task represents a unit of work to be executed by a worker pool.
type Task func() error

// TaskResult contains the result of a task execution.
type TaskResult struct {
	Index int   // Index of the task in the original queue
	Error error // Error if the task failed
}

// WorkerPool manages a pool of workers for concurrent task execution.
type WorkerPool struct {
	workerCount int
	wg          sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
func NewWorkerPool(workerCount int) (*WorkerPool, error) {
	if workerCount <= 0 {
		return nil, errors.New(errors.ErrInvalidArgument, "worker count must be positive")
	}

	return &WorkerPool{
		workerCount: workerCount,
	}, nil
}

// Execute executes all tasks and returns the results.
func (wp *WorkerPool) Execute(tasks []Task) []TaskResult {
	results := make([]TaskResult, len(tasks))
	tasksChan := make(chan int, len(tasks))

	// Start workers
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for index := range tasksChan {
				results[index] = TaskResult{
					Index: index,
					Error: tasks[index](),
				}
			}
		}()
	}

	// Send tasks
	for i := range tasks {
		tasksChan <- i
	}
	close(tasksChan)

	// Wait for all workers
	wp.wg.Wait()

	return results
}

// Close closes the worker pool (no-op for simplified implementation).
func (wp *WorkerPool) Close() {
	// Nothing to close in simplified implementation
}

// ExecuteAndWait is a convenience function that creates a pool, executes tasks, and cleans up.
func ExecuteAndWait(workerCount int, tasks []Task) ([]TaskResult, error) {
	pool, err := NewWorkerPool(workerCount)
	if err != nil {
		return nil, err
	}
	defer pool.Close()

	results := pool.Execute(tasks)
	return results, nil
}

// ParallelExecute executes tasks in parallel with the specified worker count.
// Returns the number of successful tasks and any errors that occurred.
func ParallelExecute(workerCount int, tasks []Task) (int, []error) {
	if workerCount <= 0 {
		workerCount = 1
	}

	pool, err := NewWorkerPool(workerCount)
	if err != nil {
		return 0, []error{err}
	}
	defer pool.Close()

	results := pool.Execute(tasks)

	var errs []error
	successCount := 0
	for _, result := range results {
		if result.Error != nil {
			errs = append(errs, result.Error)
		} else {
			successCount++
		}
	}
	return successCount, errs
}
