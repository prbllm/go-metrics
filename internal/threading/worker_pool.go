package threading

import (
	"context"
	"sync"
)

type WorkerPool struct {
	workers int
	queue   chan func() error
	wg      sync.WaitGroup
	once    sync.Once
	errChan chan error
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWorkerPool(workers int) *WorkerPool {
	return NewWorkerPoolWithQueueSize(workers, workers)
}

func NewWorkerPoolWithQueueSize(workers int, queueSize int) *WorkerPool {
	if queueSize <= 0 {
		queueSize = workers
	}
	return &WorkerPool{
		workers: workers,
		queue:   make(chan func() error, queueSize),
		errChan: make(chan error, workers),
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	wp.once.Do(func() {
		wp.ctx, wp.cancel = context.WithCancel(ctx)
		for i := 0; i < wp.workers; i++ {
			wp.wg.Add(1)
			go func() {
				defer wp.wg.Done()
				for {
					select {
					case <-wp.ctx.Done():
						return
					case job, ok := <-wp.queue:
						if !ok {
							return
						}
						if job != nil {
							if err := job(); err != nil {
								select {
								case wp.errChan <- err:
								default:
								}
							}
						}
					}
				}
			}()
		}
	})
}

func (wp *WorkerPool) AddJob(job func() error) {
	wp.queue <- job
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

func (wp *WorkerPool) Errors() <-chan error {
	return wp.errChan
}

func (wp *WorkerPool) Stop() {
	if wp.cancel != nil {
		wp.cancel()
	}
	wp.wg.Wait()
	close(wp.queue)
	close(wp.errChan)
}
