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

package flag

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Setter represents a FlagSet usually.
type Setter interface {
	Set(name, value string) error
}

// NewFlagHandler creates a new http handler, to set flags to setter.
func NewFlagHandler(setter Setter) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer w.Write([]byte("\r\n"))
		decoder := json.NewDecoder(r.Body)
		fs := make(map[string]string)
		if err := decoder.Decode(&fs); err != nil {
			w.Write([]byte("decode body error: " + err.Error()))
			return
		}
		for k, v := range fs {
			if err := setter.Set(k, v); err != nil {
				w.Write([]byte(fmt.Sprintf("set flag error: %s, %s, %v", k, v, err)))
				return
			}
		}
		w.Write([]byte("set flags success"))
	}
}
