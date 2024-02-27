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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestCopyObject(t *testing.T) {
	assert := assert.New(t)
	src := map[string]interface{}{
		"Name": "mia",
		"Age":  12,
		"Size": []int{1, 2, 3},
		"Rel": map[string]interface{}{
			"1st": "mic",
		},
	}
	dst := map[string]interface{}{
		"Loc": "nowhere",
		"Rel": map[string]interface{}{
			"2nd": "dst",
		},
	}
	GobDeepCopy(src, &dst)
	fmt.Printf("%+v\n", dst)
	assert.Equal("mia", dst["Name"])
	assert.Equal("nowhere", dst["Loc"])
	assert.Nil(dst["Rel"].(map[string]interface{})["2nd"])
}

func TestStripJsonNull(t *testing.T) {
	assert := assert.New(t)
	s := `{"num": 0, "str": "", "list": [], "map": {}}`
	var kvs map[string]interface{}
	err := json.Unmarshal([]byte(s), &kvs)
	assert.Nil(err)
	StripJSONNullValue(kvs)
	bs, err := json.Marshal(kvs)
	assert.Nil(err)
	fmt.Printf("%s\n", string(bs))
	assert.Equal(`{"num":0,"str":""}`, string(bs))
	_, ok := kvs["num"]
	assert.True(ok)
	_, ok = kvs["str"]
	assert.True(ok)
	_, ok = kvs["list"]
	assert.False(ok)
	_, ok = kvs["map"]
	assert.False(ok)
}

func TestIndirect(t *testing.T) {
	type args struct {
		v reflect.Value
	}
	var i interface{} = 7
	var pi = &i
	var raw = 6
	var intf interface{}
	intf = raw
	tests := []struct {
		name string
		args args
		want reflect.Value
	}{
		{
			name: "getpoint",
			args: args{
				v: reflect.ValueOf(pi),
			},
			want: reflect.ValueOf(i),
		},
		{
			name: "getinterface",
			args: args{
				v: reflect.ValueOf(intf),
			},
			want: reflect.ValueOf(raw),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Indirect(tt.args.v); !reflect.DeepEqual(got.Interface(), tt.want.Interface()) {
				t.Errorf("Indirect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestElemType(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name string
		args args
		want reflect.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ElemType(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ElemType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSimpleType(t *testing.T) {
	type args struct {
		t reflect.Type
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSimpleType(tt.args.t); got != tt.want {
				t.Errorf("IsSimpleType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFieldByTag(t *testing.T) {
	type args struct {
		rv      reflect.Value
		tagType string
		key     string
	}
	tests := []struct {
		name  string
		args  args
		want  reflect.Value
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := FindFieldByTag(tt.args.rv, tt.args.tagType, tt.args.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FieldByTag() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("FieldByTag() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFindField(t *testing.T) {
	type args struct {
		st interface{}
		ks []string
	}
	tests := []struct {
		name  string
		args  args
		want  reflect.Value
		want1 bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := FindField(tt.args.st, tt.args.ks)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindField() got = %v, want %v", got, tt.want)
			}
			if (got1 == nil) != tt.want1 {
				t.Errorf("FindField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestSetFieldValue(t *testing.T) {
	type args struct {
		st      interface{}
		fullKey string
		value   interface{}
	}
	var src1 = map[string][]string{
		"a": []string{"b", "c"},
	}
	type src struct {
		A []string
	}
	var src2 = src{
		A: []string{"b", "c"},
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "find",
			args: args{
				st:      src1,
				fullKey: "a/0",
				value:   "d",
			},
			wantErr: false,
		},
		{
			name: "find",
			args: args{
				st:      src2,
				fullKey: "A/0",
				value:   "d",
			},
			wantErr: false,
		},
		{
			name: "notfind",
			args: args{
				st:      src2,
				fullKey: "a/2",
				value:   "d",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetFieldValue(tt.args.st, tt.args.fullKey, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("SetFieldValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeepCopyObject(t *testing.T) {
	assert := assert.New(t)
	type Data struct {
		Details []string
	}
	src := Data{
		Details: []string{"test1"},
	}
	dst := Data{}
	DeepCopyObject(&src, &dst)
	assert.Equal(src.Details[0], dst.Details[0])
	src.Details[0] = "test2"
	assert.NotEqual(src.Details[0], dst.Details[0])
}

func TestGetFieldValue(t *testing.T) {
	type args struct {
		st      interface{}
		fullKey string
	}
	var src1 = map[string][]string{
		"a": []string{"b", "c"},
	}
	type src struct {
		A []string
	}
	var src2 = src{
		A: []string{"b", "c"},
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "find",
			args: args{
				st:      src1,
				fullKey: "a/0",
			},
			want:    "b",
			wantErr: false,
		},
		{
			name: "find",
			args: args{
				st:      src2,
				fullKey: "A/0",
			},
			want:    "b",
			wantErr: false,
		},
		{
			name: "find",
			args: args{
				st:      src2,
				fullKey: "a/2",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFieldValue(tt.args.st, tt.args.fullKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFieldValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFieldValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
