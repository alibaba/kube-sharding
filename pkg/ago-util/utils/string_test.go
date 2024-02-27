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
	"reflect"
	"testing"
)

func TestAppendString(t *testing.T) {
	type args struct {
		strs []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "append",
			args: args{
				strs: []string{"test", ":", "01"},
			},
			want: "test:01",
		},
		{
			name: "signal",
			args: args{
				strs: []string{"test"},
			},
			want: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppendString(tt.args.strs...); got != tt.want {
				t.Errorf("AppendString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringToByte(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "09",
			args: args{
				s: "09",
			},
			want: []byte{48, 57},
		},
		{
			name: "az",
			args: args{
				s: "az",
			},
			want: []byte{97, 122},
		},
		{
			name: "AZ",
			args: args{
				s: "AZ",
			},
			want: []byte{65, 90},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringCastByte(tt.args.s)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringToByte() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestByteToString(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "09",
			args: args{
				b: []byte{48, 57},
			},
			want: "09",
		},
		{
			name: "az",
			args: args{
				b: []byte{97, 122},
			},
			want: "az",
		},
		{
			name: "AZ",
			args: args{
				b: []byte{65, 90},
			},
			want: "AZ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ByteCastString(tt.args.b)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ByteToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestObjJSON(t *testing.T) {
	type args struct {
		o interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success",
			args: args{
				o: map[string]string{
					"a": "b",
				},
			},
			want: `{"a":"b"}`,
		},
		{
			name: "failed",
			args: args{
				o: map[interface{}]string{
					true: "b",
				},
			},
			want: ``,
		},
		{
			name: "failed",
			args: args{},
			want: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ObjJSON(tt.args.o)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ObjJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMaxLengthCommonString(t *testing.T) {
	type args struct {
		str1 string
		str2 string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case1",
			args: args{
				str1: string("AD_alliance_14-AD_alliance_14_suez_ops_na61_7u-default"),
				str2: string("AD_alliance_14-AD_alliance_14_suez_ops_na61_7u-default_partition_0"),
			},
			want: string("AD_alliance_14-AD_alliance_14_suez_ops_na61_7u-default"),
		},
		{
			name: "case2",
			args: args{
				str1: string("d2_postbuy_be_proxy_na61_01"),
				str2: string("d2_postbuy_be_proxy_na61_01_partition_0.97a460082a52fd85"),
			},
			want: string("d2_postbuy_be_proxy_na61_01"),
		},
		{
			name: "case3",
			args: args{
				str1: string("adv_mdhighqpsin0."),
				str2: string("adv_mdhighqpsin0_partition_0"),
			},
			want: string("adv_mdhighqpsin0"),
		},
		{
			name: "case4",
			args: args{
				str1: string("mainse_coupon_search"),
				str2: string("mainse_coupon_search_partition_1"),
			},
			want: string("mainse_coupon_search"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMaxLengthCommonString(tt.args.str1, tt.args.str2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMaxLengthCommonString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMaxLengthSubRepeatString(t *testing.T) {
	type args struct {
		str1 string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case1",
			args: args{
				str1: string("AD_alliance_14-AD_alliance_14_suez_ops_na61_7u-default"),
			},
			want: string("AD_alliance_14"),
		},
		{
			name: "case2",
			args: args{
				str1: string("aiLab_2-aiLab_2_suez_ops_na61_7u-default"),
			},
			want: string("aiLab_2"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMaxLengthSubRepeatString(tt.args.str1); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMaxLengthSubRepeatString() = %v, want %v", got, tt.want)
			}
		})
	}
}
