/*
Copyright 2024 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

// Err defines
var (
	ErrTimeout    = fmt.Errorf("get ticket timeout")
	ErrExeTimeout = fmt.Errorf("execute timeout")
	ErrStoped     = fmt.Errorf("executor stoped")
	ErrNoTicket   = fmt.Errorf("no ticket right now")
)

type runFunc func(params ...interface{}) (results []interface{}, err error)
type fallbackFunc func([]interface{}) error

type simpaleRunFunc func()

// AsyncExecutor is groutine pool
type AsyncExecutor struct {
	Max     int
	Tickets chan *struct{}
	stop    chan *struct{}
	stoped  bool
	sync.RWMutex
}

// NewExecutor create groutine pool
func NewExecutor(max int) *AsyncExecutor {
	p := &AsyncExecutor{}
	p.Max = max
	p.Tickets = make(chan *struct{}, p.Max)
	for i := 0; i < p.Max; i++ {
		p.Tickets <- &struct{}{}
	}
	p.stop = make(chan *struct{})
	return p
}

// ActiveCount get active count
func (p *AsyncExecutor) ActiveCount() int {
	return p.Max - len(p.Tickets)
}

// Return ticket to pool
func (p *AsyncExecutor) Return(ticket *struct{}) {
	if ticket == nil {
		return
	}

	p.Tickets <- ticket
}

// GetTicket from pool
func (p *AsyncExecutor) GetTicket(ctx context.Context) (*struct{}, error) {
	select {
	case ticket := <-p.Tickets:
		return ticket, nil
	case <-ctx.Done():
		return nil, ErrTimeout
	case <-p.stop:
		return nil, ErrStoped
	}
}

// TryGetTicket with no wait
func (p *AsyncExecutor) TryGetTicket(ctx context.Context, try bool) (*struct{}, error) {
	if !try {
		return p.GetTicket(ctx)
	}
	select {
	case ticket := <-p.Tickets:
		return ticket, nil
	case <-p.stop:
		return nil, ErrStoped
	default:
		return nil, ErrNoTicket
	}
}

// Close the pool
func (p *AsyncExecutor) Close() {
	p.Lock()
	if !p.stoped {
		close(p.stop)
		p.stoped = true
	}
	p.Unlock()
}

// CreateRunFunc from run func
func (p *AsyncExecutor) CreateRunFunc(run interface{}) func(params ...interface{}) (results []interface{}, err error) {
	r := func(params ...interface{}) (results []interface{}, err error) {
		f := reflect.ValueOf(run)
		var in []reflect.Value
		if nil != params && 0 != len(params) {
			if len(params) != f.Type().NumIn() {
				err = fmt.Errorf("The number of params is not adapted")
				log.Print(fmt.Sprintf("len(params): %d, params: %+v", len(params), params))
				return
			}
			in = make([]reflect.Value, len(params))
			for k, param := range params {
				in[k] = reflect.ValueOf(param)
			}
		}

		callResults := f.Call(in)
		results = make([]interface{}, len(callResults))
		for k, result := range callResults {
			(results)[k] = result.Interface()
		}

		return
	}
	return r
}

// Do runs your function in a synchronous manner,
func (p *AsyncExecutor) Do(ctx context.Context, try bool, run interface{}, params ...interface{}) ([]interface{}, error) {
	cmd, err := p.Go(ctx, try, run, params...)
	if nil != err {
		return nil, err
	}
	select {
	case <-cmd.done:
		return cmd.Results, cmd.ExecutorError
	}
}

// Command is the return model, for user to get results and errinfo
type Command struct {
	sync.Mutex
	ticket   *struct{}
	finished chan bool
	done     chan bool
	run      runFunc

	Start         time.Time
	End           time.Time
	ExecutorError error
	FuncError     error
	Results       []interface{}
}

// Go call run
func (p *AsyncExecutor) Go(ctx context.Context, try bool, run interface{}, params ...interface{}) (*Command, error) {
	return p.gofunc(ctx, try, false, run, params...)
}

func (p *AsyncExecutor) gofunc(ctx context.Context, try, queue bool, run interface{}, params ...interface{}) (*Command, error) {
	r := p.CreateRunFunc(run)
	cmd := &Command{
		run:      r,
		Start:    time.Now(),
		finished: make(chan bool, 1),
		done:     make(chan bool, 1),
	}

	var err error
	if try || queue {
		cmd.ticket, err = p.TryGetTicket(ctx, try)
		if nil != err {
			return nil, err
		}
	}
	go func() {
		defer func() { cmd.finished <- true }()
		if nil == cmd.ticket {
			cmd.ticket, err = p.TryGetTicket(ctx, try)
			if nil != err {
				cmd.ExecutorError = err
				return
			}
		}
		defer func() {
			p.Return(cmd.ticket)
		}()
		cmd.Start = time.Now()
		results, runErr := r(params...)
		cmd.End = time.Now()
		cmd.Results = results
		if 0 != len(results) {
			if err, ok := results[len(results)-1].(error); ok {
				cmd.FuncError = err
			}
		}
		cmd.ExecutorError = runErr
	}()

	go func() {
		defer func() {
			close(cmd.done)
		}()
		select {
		case <-cmd.finished:
			return
		case <-p.stop:
			cmd.ExecutorError = ErrStoped
			return
		case <-ctx.Done():
			cmd.ExecutorError = ErrExeTimeout
			return
		}
	}()

	return cmd, nil
}

// Queue add a func to pool. in this condition, all user logic must be in f.
func (p *AsyncExecutor) Queue(ctx context.Context, try bool, f simpaleRunFunc) error {
	_, err := p.gofunc(ctx, try, true, f)
	return err
}

// Batcher exectue f batch and wait result together
type Batcher struct {
	wg       sync.WaitGroup
	executor *AsyncExecutor
	sync.Mutex
}

// NewBatcher create batcher
func NewBatcher(max int) *Batcher {
	var b = &Batcher{}
	b.executor = NewExecutor(max)
	return b
}

// NewBatcherWithExecutor create batcher
func NewBatcherWithExecutor(executor *AsyncExecutor) *Batcher {
	var b = &Batcher{}
	b.executor = executor
	return b
}

// Go execute a func async
func (b *Batcher) Go(ctx context.Context, try bool, run interface{}, params ...interface{}) (*Command, error) {
	b.wg.Add(1)
	cmd, err := b.executor.Go(ctx, try, run, params...)
	if nil != err {
		b.wg.Done()
		return nil, err
	}
	go func() {
		defer b.wg.Done()
		select {
		case <-cmd.done:
			return
		}
	}()
	return cmd, err
}

// Wait batche executed funcs done
func (b *Batcher) Wait() {
	b.wg.Wait()
}
