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
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}

// FieldSep is the seperator for object property full key.
const FieldSep = "/"

// ConvertToStringKV convert  map[interface{}]interface{} to map[string]interface{}
func ConvertToStringKV(i map[interface{}]interface{}) map[string]interface{} {
	kvs := make(map[string]interface{})
	for k, v := range i {
		if s, ok := k.(string); ok {
			kvs[s] = ValidateJsonizeValue(v)
		}
	}
	return kvs
}

// ValidateJsonizeValue used when you want to marshal  map[interface{}]interface{}
func ValidateJsonizeValue(i interface{}) interface{} {
	switch v := i.(type) {
	// because json.Marshal doesn't support interface{} key
	case map[interface{}]interface{}:
		return ConvertToStringKV(v)
	// ensure no map[interface{}]interface exists
	case []interface{}:
		for n, item := range v {
			v[n] = ValidateJsonizeValue(item)
		}
		return v
	default:
		return v
	}
}

// ValidateStringKV ValidateStringKV
func ValidateStringKV(is map[string]interface{}) map[string]interface{} {
	for k, v := range is {
		is[k] = ValidateJsonizeValue(v)
	}
	return is
}

// IsEmptyValue copy from https://github.com/golang/go/blob/master/src/encoding/json/encode.go#L621
func IsEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return v.Len() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// StripJSONNullValue strip "some-key": null
func StripJSONNullValue(kv map[string]interface{}) {
	for k, v := range kv {
		if IsEmptyValue(reflect.ValueOf(v)) {
			delete(kv, k)
		}
	}
}

// Indirect returns the value that v points to,or the interface contains
// If v is a nil pointer, Indirect returns a zero Value.
// If v is not a pointer, Indirect returns v.
func Indirect(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr && v.Kind() != reflect.Interface {
		return v
	}
	return v.Elem()
}

// ElemType return the elem type of slice or ptr
func ElemType(v interface{}) reflect.Type {
	reflectType := reflect.ValueOf(v).Type()
	for reflectType.Kind() == reflect.Slice || reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	return reflectType
}

// IsSimpleType return is the type is simple type
func IsSimpleType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64,
		reflect.Complex128, reflect.String:
		return true
	default:
		return false
	}
}

// FindFieldByTag find field by name or json tag
func FindFieldByTag(rv reflect.Value, tagType, key string) (reflect.Value, bool) {
	if rv.Kind() != reflect.Struct {
		return rv, false
	}
	for i := 0; i < rv.Type().NumField(); i++ {
		sf := rv.Type().Field(i)
		tags := sf.Tag.Get(tagType)
		if tags == "" || tags == "-" {
			continue
		} else {
			tagspl := strings.SplitN(tags, ",", 2)
			if len(tagspl) > 0 && tagspl[0] == key {
				return rv.Field(i), true
			}
		}
	}
	return rv, false
}

// FindField find field value of st
func FindField(st interface{}, ks []string) (reflect.Value, error) {
	var rv = Indirect(reflect.ValueOf(st))
	for _, k := range ks {
		rv = Indirect(rv)
		if rv.Kind() == reflect.Slice {
			index, err := strconv.Atoi(k)
			if nil != err {
				return rv, fmt.Errorf("get slice without int key: %s", k)
			}
			if index > rv.Len()-1 {
				return rv, fmt.Errorf("out of range  with key: %s", k)
			}
			rv = rv.Index(index)
		} else if rv.Kind() == reflect.Map {
			rv = rv.MapIndex(reflect.ValueOf(k))
		} else if rv.Kind() == reflect.Struct {
			var ok bool
			rv, ok = FindFieldByTag(rv, "json", k)
			if ok {
				continue
			}
			rv = rv.FieldByName(k)
		} else {
			return rv, fmt.Errorf("type %s can't be indexed", rv.Kind())
		}
		if !rv.IsValid() {
			return rv, fmt.Errorf("no such field: %s", k)
		}
	}
	if !rv.IsValid() {
		return rv, fmt.Errorf("no such field: %s", ks)
	}
	return rv, nil
}

// SetFieldValue set value to field
func SetFieldValue(st interface{}, fullKey string, value interface{}) error {
	f, err := FindField(st, strings.Split(fullKey, FieldSep))
	if err != nil {
		return fmt.Errorf("no such field: %s, %v", fullKey, err)
	}
	if !f.CanSet() {
		return fmt.Errorf("can NOT set field: %s", fullKey)
	}
	rv := reflect.ValueOf(value)
	if f.Type() != rv.Type() {
		return fmt.Errorf("type mismatch on field: %s", fullKey)
	}
	f.Set(rv)
	return nil
}

// SetFieldValueFromJSON SetFieldValueFromJSON
func SetFieldValueFromJSON(st interface{}, fullKey string, decoder *json.Decoder) (interface{}, error) {
	f, err := FindField(st, strings.Split(fullKey, FieldSep))
	if err != nil {
		return nil, fmt.Errorf("no such field: %s, %v", fullKey, err)
	}
	if !f.CanSet() {
		return nil, fmt.Errorf("can NOT set field: %s", fullKey)
	}
	obj := reflect.New(f.Type()).Interface()
	err = decoder.Decode(&obj)
	if err == nil {
		v := reflect.ValueOf(obj).Elem()
		f.Set(v)
		return v.Interface(), nil
	}
	return nil, err
}

// GetFieldValue return the Value of field
func GetFieldValue(st interface{}, fullKey string) (interface{}, error) {
	f, err := FindField(st, strings.Split(fullKey, FieldSep))
	if err != nil {
		return nil, fmt.Errorf("no such field: %T,%s", st, fullKey)
	}
	if !f.IsValid() {
		return f, fmt.Errorf("field is not valid: %T,%s", st, fullKey)
	}
	return f.Interface(), nil
}

// GetFieldElemValue return the elem Value of field
func GetFieldElemValue(st interface{}, fullKey string) (interface{}, error) {
	f, err := FindField(st, strings.Split(fullKey, FieldSep))
	if err != nil {
		return nil, fmt.Errorf("no such field: %T,%s", st, fullKey)
	}
	if !f.IsValid() {
		return f, fmt.Errorf("field is not valid: %T,%s", st, fullKey)
	}
	f = Indirect(f)
	return f.Interface(), nil
}

// GetMapFieldValue GetMapFieldValue
func GetMapFieldValue(kvs map[string]interface{}, fullKey string) (interface{}, error) {
	ks := strings.Split(fullKey, FieldSep)
	var v interface{}
	for i, k := range ks {
		v = kvs[k]
		if v == nil {
			return nil, fmt.Errorf("no such field: %s", fullKey)
		}
		if i != len(ks)-1 {
			ok := true
			kvs, ok = v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("field is not a map (%s)", k)
			}
		}
	}
	return v, nil
}

// IsZero copy from go1.13 reflect value.go
func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(v.Float()) == 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !IsZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !IsZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		// This should never happens, but will act as a safeguard for
		// later, as a default value doesn't makes sense here.
		panic(&reflect.ValueError{Method: "reflect.Value.IsZero", Kind: v.Kind()})
	}
}

// IsFieldZero find field from st and check if it is zero
func IsFieldZero(st interface{}, fullKey string) error {
	f, err := FindField(st, strings.Split(fullKey, FieldSep))
	if err != nil || !f.IsValid() {
		return fmt.Errorf("no such field: %s", fullKey)
	}
	if IsZero(f) {
		return fmt.Errorf("field empty: %s", fullKey)
	}
	return nil
}

// IsFieldsZero find fields from st and check if they are zero
func IsFieldsZero(st interface{}, fullKeys []string) error {
	for _, k := range fullKeys {
		if err := IsFieldZero(st, k); err != nil {
			return err
		}
	}
	return nil
}

// GobDeepCopy deepcopy by gob marshal and unmarshal
func GobDeepCopy(src interface{}, dst interface{}) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(src)
	if err != nil {
		return err
	}
	err = dec.Decode(dst)
	if err != nil {
		return err
	}
	return nil
}

// JSONDeepCopy deepcopy by json marshal and unmarshal
func JSONDeepCopy(st interface{}, dst interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(st); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewBuffer(buf.Bytes()))
	decoder.UseNumber()
	return decoder.Decode(dst)
}

// DeepCopyObject deepcopy by reflect copy
func DeepCopyObject(st interface{}, dst interface{}) error {
	return copier.CopyWithOption(dst, st, copier.Option{DeepCopy: true})
}

// Addr return address of value
func Addr(sv reflect.Value) reflect.Value {
	if !sv.CanAddr() {
		sv2 := reflect.New(sv.Type())
		sv2.Elem().Set(sv)
		sv = sv2
	} else {
		sv = sv.Addr()
	}
	return sv
}

// ToSlice change interface{} to []interface{}
func ToSlice(arr interface{}) []interface{} {
	var v = Indirect(reflect.ValueOf(arr))
	if v.Kind() != reflect.Slice {
		return nil
	}
	l := v.Len()
	ret := make([]interface{}, l)
	for i := 0; i < l; i++ {
		ret[i] = v.Index(i).Interface()
	}
	return ret
}

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

// Stack returns a nicely formated stack frame, skipping skip frames
func Stack(skip int) string {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return ByteCastString(buf.Bytes())
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contains dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

// DecodeMap ...
func DecodeMap(decoder *json.Decoder) (map[string]interface{}, error) {
	var kvs map[string]interface{}
	err := decoder.Decode(&kvs)
	return kvs, err
}

// DecodeMapString ...
func DecodeMapString(s string) (map[string]interface{}, error) {
	return DecodeMap(json.NewDecoder(strings.NewReader(s)))
}
