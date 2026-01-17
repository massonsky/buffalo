package utils

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool, err := NewWorkerPool(4)
	if err != nil {
		t.Fatalf("NewWorkerPool failed: %v", err)
	}
	defer pool.Close()

	if pool.workerCount != 4 {
		t.Errorf("expected 4 workers, got %d", pool.workerCount)
	}
}

func TestNewWorkerPool_InvalidWorkerCount(t *testing.T) {
	_, err := NewWorkerPool(0)
	if err == nil {
		t.Error("expected error for invalid worker count")
	}

	_, err = NewWorkerPool(-1)
	if err == nil {
		t.Error("expected error for negative worker count")
	}
}

func TestWorkerPool_Execute(t *testing.T) {
	pool, err := NewWorkerPool(2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	executed := false
	task := func() error {
		executed = true
		return nil
	}

	results := pool.Execute([]Task{task})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error != nil {
		t.Fatalf("task failed: %v", results[0].Error)
	}

	if !executed {
		t.Error("task was not executed")
	}
}

func TestWorkerPool_ExecuteBatch(t *testing.T) {
	pool, err := NewWorkerPool(2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var counter int32
	tasks := []Task{
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { atomic.AddInt32(&counter, 1); return nil },
	}

	results := pool.Execute(tasks)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if counter != 3 {
		t.Errorf("expected counter=3, got %d", counter)
	}
}

func TestWorkerPool_ExecuteWithErrors(t *testing.T) {
	pool, err := NewWorkerPool(2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var counter int32
	tasks := []Task{
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { return fmt.Errorf("error") },
	}

	results := pool.Execute(tasks)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	errorCount := 0
	for _, result := range results {
		if result.Error != nil {
			errorCount++
		}
	}

	if errorCount != 1 {
		t.Errorf("expected 1 error, got %d", errorCount)
	}

	if counter != 2 {
		t.Errorf("expected 2 successful tasks, got %d", counter)
	}
}

func TestExecuteAndWait(t *testing.T) {
	var counter int32
	tasks := []Task{
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { atomic.AddInt32(&counter, 1); return nil },
	}

	results, err := ExecuteAndWait(2, tasks)
	if err != nil {
		t.Fatalf("ExecuteAndWait failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	if counter != 3 {
		t.Errorf("expected counter=3, got %d", counter)
	}
}

func TestParallelExecute(t *testing.T) {
	var counter int32
	tasks := []Task{
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { atomic.AddInt32(&counter, 1); return nil },
		func() error { return fmt.Errorf("error") },
		func() error { atomic.AddInt32(&counter, 1); return nil },
	}

	successCount, errs := ParallelExecute(2, tasks)

	if successCount != 3 {
		t.Errorf("expected 3 successful tasks, got %d", successCount)
	}

	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}

	if counter != 3 {
		t.Errorf("expected counter=3, got %d", counter)
	}
}

func TestWorkerPool_ConcurrentExecution(t *testing.T) {
	pool, err := NewWorkerPool(4)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const taskCount = 20
	var counter int32

	tasks := make([]Task, taskCount)
	for i := 0; i < taskCount; i++ {
		tasks[i] = func() error {
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond)
			return nil
		}
	}

	results := pool.Execute(tasks)

	if len(results) != taskCount {
		t.Errorf("expected %d results, got %d", taskCount, len(results))
	}

	if counter != taskCount {
		t.Errorf("expected counter=%d, got %d", taskCount, counter)
	}
}

func TestWorkerPool_ErrorHandling(t *testing.T) {
	pool, err := NewWorkerPool(2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tasks := []Task{
		func() error { return nil },
		func() error { return fmt.Errorf("error 1") },
		func() error { return nil },
		func() error { return fmt.Errorf("error 2") },
	}

	results := pool.Execute(tasks)

	errorCount := 0
	for _, result := range results {
		if result.Error != nil {
			errorCount++
		}
	}

	if errorCount != 2 {
		t.Errorf("expected 2 errors, got %d", errorCount)
	}
}

func BenchmarkWorkerPool_Sequential(b *testing.B) {
	tasks := make([]Task, b.N)
	for i := 0; i < b.N; i++ {
		tasks[i] = func() error {
			time.Sleep(1 * time.Microsecond)
			return nil
		}
	}

	b.ResetTimer()
	_, _ = ExecuteAndWait(1, tasks)
}

func BenchmarkWorkerPool_Parallel(b *testing.B) {
	tasks := make([]Task, b.N)
	for i := 0; i < b.N; i++ {
		tasks[i] = func() error {
			time.Sleep(1 * time.Microsecond)
			return nil
		}
	}

	b.ResetTimer()
	_, _ = ExecuteAndWait(4, tasks)
}
