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

package rpc

import (
	"fmt"
	"net/url"
	"strconv"

	restful "github.com/emicklei/go-restful"
)

type request struct {
	endpoint   string
	pathFormat string
	namespace  string
	resource   string
	name       string
	selector   string
	timeoutSec int
	source     string
	watch      bool

	resourceversion string
}

func (r *request) URL() string {
	u := url.URL{}
	u.Host = r.endpoint
	u.Path = fmt.Sprintf(r.pathFormat, r.namespace, r.resource)
	if r.name != "" {
		u.Path = u.Path + "/" + r.name
	}
	u.Scheme = "http"
	u.ForceQuery = true
	qs := make(url.Values)
	qs.Set("src", r.source)
	if r.selector != "" {
		qs.Set("labelSelector", r.selector)
	}
	if r.timeoutSec > 0 {
		qs.Set("timeoutSeconds", strconv.Itoa(r.timeoutSec))
	}
	if r.resourceversion != "" {
		qs.Set("resourceVersion", r.resourceversion)
	}
	if r.watch {
		qs.Set("watch", "1")
	}
	u.RawQuery = qs.Encode()
	return u.String()
}

func parseRequest(req *restful.Request) *request {
	request := &request{
		namespace:  req.PathParameter("namespace"),
		resource:   req.PathParameter("resource"),
		name:       req.PathParameter("name"),
		selector:   req.QueryParameter("labelSelector"),
		source:     req.QueryParameter("src"),
		timeoutSec: int(checkParseNum(req, "timeoutSeconds", 0)),
		watch:      req.QueryParameter("watch") == "1",

		resourceversion: req.QueryParameter("resourceVersion"),
	}
	return request
}

func checkParseNum(req *restful.Request, k string, d uint64) uint64 {
	if s := req.QueryParameter(k); s != "" {
		if u, err := strconv.ParseUint(s, 10, 64); err == nil {
			return u
		}
	}
	return d
}
