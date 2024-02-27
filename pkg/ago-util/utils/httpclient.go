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
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

var _ HTTPClient = &LiteHTTPClient{}

// HTTPClient HTTPClient
type HTTPClient interface {
	Get(url string) (*LiteHTTPResp, error)
	Post(url string, body string) (*LiteHTTPResp, error)
	Put(url string, body string) (*LiteHTTPResp, error)
	Delete(url string) (*LiteHTTPResp, error)
}

// LiteHTTPResp contain resp body and status
type LiteHTTPResp struct {
	Body       []byte
	StatusCode int
	Headers    http.Header
}

// HTTPClientOptions is options of client scop
type HTTPClientOptions struct {
	Timeout                      time.Duration
	Headers                      map[string]string
	TransportMaxIdleConns        int
	TransportMaxIdleConnsPerHost int
}

// LiteHTTPClient LiteHTTPClient
type LiteHTTPClient struct {
	impl *http.Client
	opts *HTTPClientOptions
}

// NewLiteHTTPClient create new  LiteHTTPClient
func NewLiteHTTPClient(timeout time.Duration) *LiteHTTPClient {
	client := &LiteHTTPClient{}
	client.impl = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return client
}

// NewLiteHTTPClientOpts create new  LiteHTTPClient
func NewLiteHTTPClientOpts(opts *HTTPClientOptions) *LiteHTTPClient {
	client := &LiteHTTPClient{opts: opts}
	client.impl = &http.Client{
		Timeout: opts.Timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          opts.TransportMaxIdleConns,
			MaxConnsPerHost:       opts.TransportMaxIdleConnsPerHost,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return client
}

// NewLiteHTTPClientWithClient create new  LiteHTTPClient
func NewLiteHTTPClientWithClient(httpClient *http.Client) *LiteHTTPClient {
	client := &LiteHTTPClient{}
	client.impl = httpClient
	return client
}

// Get query with GET method
func (c *LiteHTTPClient) Get(url string) (*LiteHTTPResp, error) {
	return c.Request(nil, url, "", http.MethodGet, nil)
}

// Post query with POST method
func (c *LiteHTTPClient) Post(url string, body string) (*LiteHTTPResp, error) {
	return c.Request(nil, url, body, http.MethodPost, nil)
}

// Put query with PUT method
func (c *LiteHTTPClient) Put(url string, body string) (*LiteHTTPResp, error) {
	return c.Request(nil, url, body, http.MethodPut, nil)
}

// Delete query with DELETE method
func (c *LiteHTTPClient) Delete(url string) (*LiteHTTPResp, error) {
	return c.Request(nil, url, "", http.MethodDelete, nil)
}

// Request send query
func (c *LiteHTTPClient) Request(ctx context.Context, url string, body string, method string, headers map[string]string) (*LiteHTTPResp, error) {
	var req *http.Request
	var err error
	url = FixHTTPUrl(url)
	if body != "" {
		req, err = http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, headers)
	// disable connection resuing.
	// try to fix drogo api EOF error, not sure this is working.
	req.Close = true
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	resp, err := c.impl.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &LiteHTTPResp{Body: r, StatusCode: resp.StatusCode, Headers: resp.Header}, nil
}

// ProxyRequest proxy request to target url
func (c *LiteHTTPClient) ProxyRequest(host string, wr http.ResponseWriter, r *http.Request) error {
	var resp *http.Response
	var err error
	var req *http.Request

	copyHeader := func(src http.Header, dst http.Header) {
		for name, values := range src {
			for _, v := range values {
				dst.Add(name, v)
			}
		}
	}
	req, err = http.NewRequest(r.Method, host+r.RequestURI, r.Body)
	copyHeader(r.Header, req.Header)
	resp, err = c.impl.Do(req)
	r.Body.Close()
	if err != nil {
		return err
	}
	copyHeader(resp.Header, wr.Header())
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
	resp.Body.Close()
	return nil
}

func (c *LiteHTTPClient) setHeaders(req *http.Request, headers map[string]string) {
	if c.opts != nil && c.opts.Headers != nil {
		for k, v := range c.opts.Headers {
			req.Header.Set(k, v)
		}
	}
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
}
