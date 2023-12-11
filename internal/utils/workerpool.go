package utils

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type taskInput[T any] struct {
	index int
	total int
	value T
}

type taskOutput[T any] struct {
	index  int
	result T
}

type WorkerPool[I any, O any] struct {
	maxWorkers int
	f          func(value I) (O, error)
	onProgress func(current int, total int)
}

func NewWorkerPool[I any, O any](f func(value I) (O, error), maxWorkers int) *WorkerPool[I, O] {
	return &WorkerPool[I, O]{
		maxWorkers: maxWorkers,
		f:          f,
	}
}

func (wp *WorkerPool[I, O]) worker(ctx context.Context, id int, inputCh <-chan taskInput[I], outputCh chan<- taskOutput[O]) error {
	for {
		select {
		case input, ok := <-inputCh:
			if !ok {
				return nil
			}

			if wp.onProgress != nil {
				go wp.onProgress(input.index, input.total)
			}

			result, err := wp.f(input.value)
			if err != nil {
				return fmt.Errorf("worker %d error: %w", id, err)
			}

			outputCh <- taskOutput[O]{
				index:  input.index,
				result: result,
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (wp *WorkerPool[I, O]) Map(ctx context.Context, input []I) ([]O, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	var workerErrs error

	inputCh := make(chan taskInput[I])
	outputCh := make(chan taskOutput[O])

	for i := 0; i < wp.maxWorkers; i++ {
		id := i + 1
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := wp.worker(ctx, id, inputCh, outputCh)
			if err != nil {
				workerErrs = errors.Join(workerErrs, err)
				cancel()
			}
		}()
	}

	go func() {
		inputLen := len(input)
		for index, value := range input {
			inputCh <- taskInput[I]{
				index: index,
				value: value,
				total: inputLen,
			}
		}
		close(inputCh)
	}()

	go func() {
		wg.Wait()
		close(outputCh)
	}()

	output := make([]O, len(input))
	for taskOutput := range outputCh {
		output[taskOutput.index] = taskOutput.result
	}

	return output, workerErrs
}

func (wp *WorkerPool[I, O]) OnProgress(f func(current int, total int)) {
	wp.onProgress = f
}
