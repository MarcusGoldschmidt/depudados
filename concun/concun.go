package concun

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrWorkerCountLessThenOne = errors.New("worker count is less then one")
	ErrJobTimedOut            = errors.New("job request timed out")
	ErrPoolNotRunning         = errors.New("pool is not running")
	ErrCouldNotStartPool      = errors.New("pool is not in initial or stopped state")
)

type workerRequest[Request any, Response any] struct {
	ctx       context.Context
	request   Request
	valueBack chan workerResult[Response]
}

type workerResult[Response any] struct {
	response *Response
	err      error
}

//go:generate stringer -type=PoolState -output=pool_state_string.go
type PoolState int8

const (
	Initial PoolState = iota
	Starting
	Running
	Stopping
	Stopped
)

type PoolHandler[Request any, Response any] func(ctx context.Context, r Request) (*Response, error)

type Pool[Request any, Response any] struct {
	mutex sync.RWMutex
	state PoolState

	workerCount int
	queuedJobs  int64

	handleFunc PoolHandler[Request, Response]

	workerQueue chan workerRequest[Request, Response]

	wg sync.WaitGroup
}

func NewPool[Request any, Response any](workerCount int, handler PoolHandler[Request, Response]) (*Pool[Request, Response], error) {
	if workerCount < 1 {
		return nil, ErrWorkerCountLessThenOne
	}

	pool := &Pool[Request, Response]{
		workerCount: workerCount,
		workerQueue: make(chan workerRequest[Request, Response], workerCount),
		queuedJobs:  0,
		handleFunc:  handler,
		wg:          sync.WaitGroup{},
		state:       Initial,
	}

	err := pool.startPool()
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (p *Pool[Request, Response]) startPool() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.state != Initial && p.state != Stopped {
		return ErrCouldNotStartPool
	}

	p.state = Starting

	for i := 0; i < p.workerCount; i++ {
		go func() {
			for chanValue := range p.workerQueue {
				func(value workerRequest[Request, Response]) {
					defer p.wg.Done()
					response, err := p.handleFunc(value.ctx, value.request)
					value.valueBack <- workerResult[Response]{
						response: response,
						err:      err,
					}
				}(chanValue)
			}
		}()
	}

	p.state = Running

	return nil
}

func (p *Pool[Request, Response]) QueueLength() int64 {
	return atomic.LoadInt64(&p.queuedJobs)
}

func (p *Pool[Request, Response]) Process(request Request) (*Response, error) {
	return p.ProcessWithCtx(context.Background(), request)
}

func (p *Pool[Request, Response]) ProcessWithCtx(ctx context.Context, request Request) (*Response, error) {
	atomic.AddInt64(&p.queuedJobs, 1)

	valueBack := make(chan workerResult[Response])

	wRequest := workerRequest[Request, Response]{
		ctx:       ctx,
		request:   request,
		valueBack: valueBack,
	}

	p.mutex.RLock()

	if p.state != Running {
		return nil, ErrPoolNotRunning
	}

	p.wg.Add(1)
	p.workerQueue <- wRequest
	atomic.AddInt64(&p.queuedJobs, -1)

	p.mutex.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case wResponse := <-valueBack:
		return wResponse.response, wResponse.err
	}
}

func (p *Pool[Request, Response]) ProcessWithTimeout(ctx context.Context, request Request, timeout time.Duration) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return p.ProcessWithCtx(ctx, request)
}

// Wait for all jobs to finish, does not close the worker queue
func (p *Pool[Request, Response]) Wait() {
	p.wg.Wait()
}

func (p *Pool[Request, Response]) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.state = Stopping
	p.wg.Wait()
	close(p.workerQueue)

	p.state = Stopped
}

func (p *Pool[Request, Response]) IsRunning() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.state == Running
}
