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

package restutil

import (
	"time"

	"github.com/emicklei/go-restful"
)

func init() {
	// You can specify the Content-Type in http header to use other encoders&decoders.
	restful.DefaultRequestContentType(restful.MIME_JSON)
	restful.DefaultResponseContentType(restful.MIME_JSON)
}

// PatchContainerOptions options to patch container
type PatchContainerOptions struct {
	// Global logging
	LoggingFunc RecordTimeFunc
}

// PatchContainer patch the container
func PatchContainer(container *restful.Container, opts *PatchContainerOptions) {
	if opts.LoggingFunc != nil {
		container.Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
			start := time.Now()
			chain.ProcessFilter(req, resp)
			opts.LoggingFunc(req, resp, time.Now().Sub(start))
		})
	}
}
