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

package common

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	glog "k8s.io/klog"
)

// HTTPClient is
type HTTPClient struct {
	client *http.Client
}

var defaultHTTPClient = &http.Client{
	Transport: &http.Transport{},
}
var defaultClient = &HTTPClient{client: defaultHTTPClient}

// NewHTTPClient create http client
func NewHTTPClient() *HTTPClient {
	return defaultClient
}

// NewWithHTTPClient create with http client
func NewWithHTTPClient(client *http.Client) *HTTPClient {
	return &HTTPClient{client: client}
}

func (c *HTTPClient) sendRequest(ctx context.Context, method, url string, body string, headers map[string]string) (*http.Response, error) {
	var err error
	// 构造request
	var request *http.Request
	url = utils.FixHTTPUrl(url)
	if "" != body {
		request, err = http.NewRequest(method, url, bytes.NewBufferString(body))
		if err != nil {
			glog.Errorf("create request error:%v", err)
			return nil, err
		}
	} else {
		request, err = http.NewRequest(method, url, nil)
		if err != nil {
			glog.Errorf("new request error:%v", err)
			return nil, err
		}
	}
	if nil != ctx {
		request = request.WithContext(ctx)
	}
	for k, v := range headers {
		request.Header.Add(k, v)
	}

	response, err := c.client.Do(request)
	if nil != err {
		glog.Errorf("do request error:%v", err)
		return response, err
	}

	var reader io.ReadCloser
	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(response.Body)
	case "deflate":
		reader, err = zlib.NewReader(response.Body)
	default:
		reader = response.Body
	}

	response.Body = reader
	return response, err
}

// SendFormRequest SendFormRequest
func (c *HTTPClient) SendFormRequest(ctx context.Context, method, urls string, reqForm map[string]string, headers map[string]string) (*http.Response, error) {
	form := url.Values{}

	for key, value := range reqForm {
		form.Set(key, value)
	}
	var body string
	if nil != reqForm && 0 != len(reqForm) {
		if "GET" == method || "DELETE" == method {
			urls = urls + "?" + form.Encode()
		} else {
			body = form.Encode()
		}
	}

	if nil == headers {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = utils.HTTPContentTypeWWW

	return c.sendRequest(ctx, method, urls, body, headers)
}

// SendJSONRequest SendJSONRequest
func (c *HTTPClient) SendJSONRequest(ctx context.Context, method, url string, req interface{}, headers map[string]string) (*http.Response, error) {
	var body string
	if nil != req {
		body = utils.ObjJSON(req)
	}

	return c.sendRequest(ctx, method, url, body, headers)
}
