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
	"reflect"
	"testing"
	"time"
)

func newTestExecutor() *AsyncExecutor {
	p := &AsyncExecutor{}
	p.Max = 3
	p.Tickets = make(chan *struct{}, 3)
	for i := 0; i < p.Max; i++ {
		p.Tickets <- &struct{}{}
	}
	p.stop = make(chan *struct{})
	return p
}

func waitFunc() {
	time.Sleep(time.Second * 100)
}

func noWaitFunc() {
}

func TestNewExecutor(t *testing.T) {
	type args struct {
		max int
	}
	tests := []struct {
		name string
		args args
		want *AsyncExecutor
	}{
		{
			name: "test_new",
			args: args{max: 3},
			want: newTestExecutor(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewExecutor(tt.args.max); got.Max != tt.want.Max {
				t.Errorf("NewExecutor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsyncExecutor_ActiveCount(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	e := newTestExecutor()
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			want: 1,
		},
	}

	e.Go(context.Background(), false, waitFunc)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &AsyncExecutor{
				Max:     tt.fields.Max,
				Tickets: tt.fields.Tickets,
				stop:    tt.fields.stop,
				stoped:  tt.fields.stoped,
			}
			time.Sleep(time.Second)
			if got := p.ActiveCount(); got != tt.want {
				t.Errorf("AsyncExecutor.ActiveCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsyncExecutor_Return(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	type args struct {
		ticket *struct{}
	}
	var executeTimes = 0

	e := newTestExecutor()
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantTicket int
		want       int
	}{
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args: args{
				ticket: &struct{}{},
			},
			wantTicket: 0,
			want:       1,
		},
	}
	e.Go(context.Background(), false, func() {
		executeTimes++
		time.Sleep(100 * time.Second)
	})
	time.Sleep(time.Second)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e.Return(tt.args.ticket)

			if got := e.ActiveCount(); got != tt.wantTicket {
				t.Errorf("AsyncExecutor.ActiveCount() = %v, want %v", got, tt.wantTicket)
			}
			if executeTimes != tt.want {
				t.Errorf("AsyncExecutor.executeTimes = %v, want %v", executeTimes, tt.want)
			}
		})
	}
}

func TestAsyncExecutor_GetTicket(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	type args struct {
		ctx context.Context
	}
	e := newTestExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *struct{}
		wantErr bool
	}{
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: ctx},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &AsyncExecutor{
				Max:     tt.fields.Max,
				Tickets: tt.fields.Tickets,
				stop:    tt.fields.stop,
				stoped:  tt.fields.stoped,
			}
			_, err := p.GetTicket(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsyncExecutor.GetTicket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestAsyncExecutor_TryGetTicket(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	type args struct {
		ctx context.Context
		try bool
	}
	e := newTestExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *struct{}
		wantErr bool
	}{
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: context.Background()},
			wantErr: false,
		},
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: ctx, try: true},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &AsyncExecutor{
				Max:     tt.fields.Max,
				Tickets: tt.fields.Tickets,
				stop:    tt.fields.stop,
				stoped:  tt.fields.stoped,
			}
			_, err := p.TryGetTicket(tt.args.ctx, tt.args.try)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsyncExecutor.TryGetTicket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestAsyncExecutor_TryGetTicketTrue(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	type args struct {
		ctx context.Context
		try bool
	}
	e := newTestExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *struct{}
		wantErr bool
	}{
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
			},
			args:    args{ctx: ctx, try: true},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &AsyncExecutor{
				Max:     tt.fields.Max,
				Tickets: tt.fields.Tickets,
				stop:    tt.fields.stop,
				stoped:  tt.fields.stoped,
			}
			for i := 0; i < 100; i++ {
				ticket, err := p.TryGetTicket(tt.args.ctx, tt.args.try)
				if (err != nil) != false {
					t.Errorf("AsyncExecutor.TryGetTicket() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				p.Return(ticket)
			}
			for i := 0; i < 3; i++ {
				_, err := p.TryGetTicket(tt.args.ctx, tt.args.try)
				if (err != nil) != false {
					t.Errorf("AsyncExecutor.TryGetTicket() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			for i := 0; i < 100; i++ {
				_, err := p.TryGetTicket(tt.args.ctx, tt.args.try)
				if (err != nil) != true {
					t.Errorf("AsyncExecutor.TryGetTicket() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			for i := 0; i < 3; i++ {
				p.Return(&struct{}{})
			}
			for i := 0; i < 100; i++ {
				ticket, err := p.TryGetTicket(tt.args.ctx, tt.args.try)
				if (err != nil) != false {
					t.Errorf("AsyncExecutor.TryGetTicket() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				p.Return(ticket)
			}
		})
	}

}

func TestAsyncExecutor_Close(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	type args struct {
		ctx context.Context
	}
	var executeTimes = 0
	e := newTestExecutor()
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantTicket int
		want       int
	}{
		{
			name: "count",
			fields: fields{
				Max:     e.Max,
				Tickets: e.Tickets,
				stop:    e.stop,
			},
			wantTicket: 1,
			want:       1,
		},
	}
	e.Queue(context.Background(), false, func() {
		executeTimes++
		time.Sleep(100 * time.Second)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &AsyncExecutor{
				Max:     tt.fields.Max,
				Tickets: tt.fields.Tickets,
				stop:    tt.fields.stop,
				stoped:  tt.fields.stoped,
			}
			p.Close()
			time.Sleep(time.Second * 1)
			if got := e.ActiveCount(); got != tt.wantTicket {
				t.Errorf("AsyncExecutor.ActiveCount() = %v, want %v", got, tt.wantTicket)
			}
			if executeTimes != tt.want {
				t.Errorf("AsyncExecutor.executeTimes = %v, want %v", executeTimes, tt.want)
			}
		})
	}
}

func TestAsyncExecutor_Do(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	type args struct {
		ctx    context.Context
		try    bool
		run    interface{}
		params []interface{}
	}
	e := newTestExecutor()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	f := func(in int) {
		if in != 1 {
			t.Errorf("want parames 1")
		}
		time.Sleep(time.Second * 100)
	}
	ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel1()

	f1 := func(in, inn int) (int, int) {
		if in != 1 || inn != 2 {
			t.Errorf("want parames 1")
		}
		return 1, 2
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		want        []interface{}
		wantErr     bool
		wantErrChan bool
	}{

		{
			name: "timeout",
			args: args{
				try:    false,
				run:    f,
				ctx:    ctx,
				params: []interface{}{1},
			},
			want:    nil,
			wantErr: true,
		},

		{
			name: "ok",
			args: args{
				try:    false,
				run:    f1,
				ctx:    ctx1,
				params: []interface{}{1, 2},
			},
			want:    []interface{}{1, 2},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.Do(tt.args.ctx, tt.args.try, tt.args.run, tt.args.params...)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsyncExecutor.Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AsyncExecutor.Do() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsyncExecutor_Go(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	type args struct {
		ctx    context.Context
		try    bool
		run    interface{}
		params []interface{}
	}
	e := newTestExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	f := func(in int) {
		if in != 1 {
			t.Errorf("want parames 1")
		}
		time.Sleep(time.Second * 100)
	}
	ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel1()

	f1 := func(in, inn int) {
		if in != 1 || inn != 2 {
			t.Errorf("want parames 1")
		}
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErrChan bool
		wantErr     bool
	}{
		{
			name: "timeout",
			args: args{
				try:    false,
				run:    f,
				ctx:    ctx,
				params: []interface{}{1},
			},
			wantErrChan: true,
			wantErr:     false,
		},
		{
			name: "ok",
			args: args{
				try:    false,
				run:    f1,
				ctx:    ctx1,
				params: []interface{}{1, 2},
			},
			wantErrChan: false,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.Go(tt.args.ctx, tt.args.try, tt.args.run, tt.args.params...)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsyncExecutor.Go() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			<-got.done
			err = got.ExecutorError
			if (err != nil) != tt.wantErrChan {
				t.Errorf("AsyncExecutor.Go() error = %v, wantErr %v", err, tt.wantErrChan)
				return
			}
		})
	}
}

func TestAsyncExecutor_Queue(t *testing.T) {
	type fields struct {
		Max     int
		Tickets chan *struct{}
		stop    chan *struct{}
		stoped  bool
	}
	type args struct {
		ctx context.Context
		try bool
		f   simpaleRunFunc
	}
	e := NewExecutor(2)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	f := func() {
		time.Sleep(time.Second * 10)
	}
	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel1()

	f1 := func() {
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "functimeout",
			args: args{
				try: false,
				f:   f,
				ctx: ctx,
			},
			wantErr: false,
		},
		{
			name: "ok",
			args: args{
				try: false,
				f:   f1,
				ctx: ctx1,
			},
			wantErr: false,
		},
		{
			name: "ok",
			args: args{
				try: false,
				f:   f,
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "timeout",
			args: args{
				try: false,
				f:   f,
				ctx: ctx1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := e.Queue(tt.args.ctx, tt.args.try, tt.args.f); (err != nil) != tt.wantErr {
				t.Errorf("AsyncExecutor.Queue() error = %v, wantErr %v", err, tt.wantErr)
			}
			fmt.Println("ActiveCount()", e.ActiveCount())
		})
	}
}

func TestBatcher_Go(t *testing.T) {
	type fields struct {
		executor *AsyncExecutor
	}
	type args struct {
		ctx    context.Context
		try    bool
		run    interface{}
		params []interface{}
	}

	e := NewExecutor(2)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	f := func(in int) {
		if in != 1 {
			t.Errorf("want parames 1")
		}
		time.Sleep(time.Second * 100)
	}
	ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel1()

	f1 := func(in, inn int) (int, error) {
		if in != 1 || inn != 2 {
			t.Errorf("want parames 1")
		}
		return 3, fmt.Errorf("test error")
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		wantErrChan bool
		wantResults []interface{}
		wantUserErr error
	}{
		{
			name: "timeout",
			fields: fields{
				executor: e,
			},
			args: args{
				try:    false,
				run:    f,
				ctx:    ctx,
				params: []interface{}{1},
			},
			wantErr:     false,
			wantErrChan: true,
			wantResults: nil,
			wantUserErr: nil,
		},
		{
			name: "ok",
			fields: fields{
				executor: e,
			},
			args: args{
				try:    false,
				run:    f1,
				ctx:    ctx1,
				params: []interface{}{1, 2},
			},
			wantErr:     false,
			wantErrChan: false,
			wantResults: []interface{}{3, fmt.Errorf("test error")},
			wantUserErr: fmt.Errorf("test error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Batcher{
				executor: tt.fields.executor,
			}
			got, err := b.Go(tt.args.ctx, tt.args.try, tt.args.run, tt.args.params...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Batcher.Go() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			<-got.done
			err = got.ExecutorError
			if (err != nil) != tt.wantErrChan {
				t.Errorf("AsyncExecutor.Go() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			results := got.Results
			if (err != nil) != tt.wantErrChan {
				t.Errorf("AsyncExecutor.Go() error = %v, wantErr %v", results, tt.wantResults)
				return
			}
			userErr := got.FuncError
			if (err != nil) != tt.wantErrChan {
				t.Errorf("AsyncExecutor.Go() error = %v, wantErr %v", userErr, tt.wantUserErr)
				return
			}
		})
	}
}

func TestBatcher_Wait(t *testing.T) {
	type fields struct {
		executor *AsyncExecutor
	}
	type args struct {
		ctx    context.Context
		try    bool
		run    interface{}
		params []interface{}
	}

	e := NewExecutor(2)
	e1 := NewExecutor(0)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	f := func(in int) {
		if in != 1 {
			t.Errorf("want parames 1")
		}
		time.Sleep(time.Second * 100)
	}
	ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel1()

	f1 := func(in, inn int) (int, error) {
		if in != 1 || inn != 2 {
			t.Errorf("want parames 1")
		}
		return 3, fmt.Errorf("test error")
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		wantErrChan bool
		wantResults []interface{}
		wantUserErr error
	}{
		{
			name: "timeout",
			fields: fields{
				executor: e,
			},
			args: args{
				try:    false,
				run:    f,
				ctx:    ctx,
				params: []interface{}{1},
			},
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				executor: e1,
			},
			args: args{
				try:    true,
				run:    f1,
				ctx:    ctx1,
				params: []interface{}{1, 2},
			},
			wantErr: true,
		},
		{
			name: "ok",
			fields: fields{
				executor: e,
			},
			args: args{
				try:    true,
				run:    f1,
				ctx:    ctx1,
				params: []interface{}{1, 2},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Batcher{
				executor: tt.fields.executor,
			}
			_, err := b.Go(tt.args.ctx, tt.args.try, tt.args.run, tt.args.params...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Batcher.Go() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			b.Wait()
		})
	}
}
