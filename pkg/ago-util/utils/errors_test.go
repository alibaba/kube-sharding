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
	"fmt"
	"reflect"
	"testing"
)

func TestNewErrors(t *testing.T) {
	tests := []struct {
		name string
		want Errors
	}{
		{
			name: "new",
			want: Errors([]error{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewErrors(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrors_Add(t *testing.T) {
	type args struct {
		err error
	}
	errors := NewErrors()
	tests := []struct {
		name string
		e    *Errors
		args args
		want Errors
	}{
		{
			name: "add nil",
			e:    &Errors{},
			args: args{
				nil,
			},
			want: Errors([]error{}),
		},
		{
			name: "add err",
			e:    &errors,
			args: args{
				fmt.Errorf("test error"),
			},
			want: Errors([]error{fmt.Errorf("test error")}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.e.Add(tt.args.err)
			if !reflect.DeepEqual(*tt.e, tt.want) {
				t.Errorf("Add not equal, %v, want %v", *tt.e, tt.want)
			}
		})
	}
}

func TestErrors_Empty(t *testing.T) {
	errors := NewErrors()
	tests := []struct {
		name string
		e    *Errors
		want bool
	}{
		{
			name: "nil",
			e:    &Errors{},
			want: true,
		},
		{
			name: "empty",
			e:    &errors,
			want: true,
		},
		{
			name: "not empty",
			e:    (*Errors)(&[]error{fmt.Errorf("test error")}),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Empty(); got != tt.want {
				t.Errorf("Errors.Empty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrors_String(t *testing.T) {
	tests := []struct {
		name string
		e    *Errors
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.String(); got != tt.want {
				t.Errorf("Errors.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrors_Error(t *testing.T) {
	errors := NewErrors()
	tests := []struct {
		name    string
		e       *Errors
		want    error
		wantErr bool
	}{
		{
			name:    "nil",
			e:       &Errors{},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "empty",
			e:       &errors,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "not empty",
			e:       (*Errors)(&[]error{fmt.Errorf("test error")}),
			want:    fmt.Errorf("test error"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if err = tt.e.Error(); (err != nil) != tt.wantErr {
				t.Errorf("Errors.Error() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(err, tt.want) {
				t.Errorf("Error() not equal, %v, want %v", err, tt.want)
			}
		})
	}
}
