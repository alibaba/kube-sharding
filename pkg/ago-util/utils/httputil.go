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
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"runtime"
	"strconv"
	"strings"
)

// HttpContentTypes
const (
	HTTPContentTypeWWW   = "application/x-www-form-urlencoded"
	HTTPContentTypeJSON  = "application/json"
	HTTPContentTypeXML   = "application/xml"
	HTTPContentTypePLAIN = "text/plain"
	HTTPContentTypeHTML  = "text/html"
)

var defaultHTTPClient = http.Client{
	Transport: &http.Transport{},
}

var defaultClient = &LiteHTTPClient{impl: &defaultHTTPClient}

func SendRawRequest(ctx context.Context, url string, body string, method string, headers map[string]string) (*LiteHTTPResp, error) {
	return defaultClient.Request(ctx, url, body, method, headers)
}

// SendFormRequest send http request with form format
func SendFormRequest(ctx context.Context, method, urls string, reqForm map[string]string, headers map[string]string) (*LiteHTTPResp, error) {
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
	headers["Content-Type"] = HTTPContentTypeWWW

	return defaultClient.Request(ctx, urls, body, method, headers)
}

// SendJSONRequest send http request with json format
func SendJSONRequest(ctx context.Context, method, url string, req interface{}, headers map[string]string) (*LiteHTTPResp, error) {
	var body string
	switch v := req.(type) {
	case []byte:
		body = string(v)
	case string:
		body = v
	default:
		body = ObjJSON(req)
	}
	if len(headers) == 0 {
		headers = map[string]string{}
	}
	headers["Content-Type"] = "application/json"
	return defaultClient.Request(ctx, url, body, method, headers)
}

// ParseRespToString parse http resp to string
func ParseRespToString(resp *http.Response, err error) (string, error) {
	if nil != err {
		return "", err
	}
	if nil == resp || nil == resp.Body {
		err = fmt.Errorf("invalid data")
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		err = fmt.Errorf("status code error :%d, with message :%s", resp.StatusCode, buf.String())
		return "", err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if nil != err {
		return "", err
	}

	var s strings.Builder
	s.Write(buf.Bytes())
	return s.String(), err
}

// ParseJSONResp parse http json resp to interface
func ParseJSONResp(resp *http.Response, err error, result interface{}) error {
	if nil != err {
		return err
	}
	if nil == resp || nil == resp.Body {
		err = fmt.Errorf("invalid data")
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		err = fmt.Errorf("status code error :%d", resp.StatusCode)
		return err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if nil != err {
		return err
	}
	decoder := json.NewDecoder(buf)
	decoder.UseNumber()
	err = decoder.Decode(&result)
	return err
}

// FixHTTPUrl app http:// to url
func FixHTTPUrl(url string) string {
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	return url
}

//
const httpPrefixPProf = "/debug/pprof"

// PProfHandlers get all pprof path and handlers to new http handler
func PProfHandlers() map[string]http.Handler {
	// set only when there's no existing setting
	if runtime.SetMutexProfileFraction(-1) == 0 {
		// 1 out of 5 mutex events are reported, on average
		runtime.SetMutexProfileFraction(5)
	}

	m := make(map[string]http.Handler)

	m[httpPrefixPProf+"/"] = http.HandlerFunc(pprof.Index)
	m[httpPrefixPProf+"/profile"] = http.HandlerFunc(pprof.Profile)
	m[httpPrefixPProf+"/symbol"] = http.HandlerFunc(pprof.Symbol)
	m[httpPrefixPProf+"/cmdline"] = http.HandlerFunc(pprof.Cmdline)
	m[httpPrefixPProf+"/trace "] = http.HandlerFunc(pprof.Trace)
	m[httpPrefixPProf+"/heap"] = pprof.Handler("heap")
	m[httpPrefixPProf+"/goroutine"] = pprof.Handler("goroutine")
	m[httpPrefixPProf+"/threadcreate"] = pprof.Handler("threadcreate")
	m[httpPrefixPProf+"/block"] = pprof.Handler("block")
	m[httpPrefixPProf+"/mutex"] = pprof.Handler("mutex")

	return m
}

// ClientIP get http request client ip
func ClientIP(req *http.Request) string {
	clientIP := req.Header.Get("X-Forwarded-For")
	if clientIP != "" {
		return strings.TrimSpace(strings.Split(clientIP, ",")[0])
	}
	clientIP = req.Header.Get("X-Real-IP")
	if clientIP != "" {
		return clientIP
	}
	clientIP = req.Header.Get("http_x_forwarded_for")
	if clientIP != "" {
		return clientIP
	}
	h, _, _ := net.SplitHostPort(req.RemoteAddr)
	return h
}

// HTTPRecover send 500 code response when panic
func HTTPRecover(panicReason interface{}, httpWriter http.ResponseWriter) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("recover from panic situation: - %v\r\n", panicReason))
	for i := 2; ; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		buffer.WriteString(fmt.Sprintf("    %s:%d\r\n", file, line))
	}
	log.Print(buffer.String())
	httpWriter.WriteHeader(http.StatusInternalServerError)

	httpWriter.Write([]byte(http.StatusText(http.StatusInternalServerError)))
}

func writeHeader(w http.ResponseWriter, code int, contentType string) {
	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	w.WriteHeader(code)
}

// RenderJSON send json response
func RenderJSON(w http.ResponseWriter, code int, data interface{}) error {
	writeHeader(w, code, "application/json")
	b, err := json.Marshal(data)
	if nil != err {
		return err
	}
	newline := []byte("\r\n")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(newline)+len(b)))
	w.Write(b)
	w.Write(newline)
	return nil
}

// RenderPlain send plain response
func RenderPlain(w http.ResponseWriter, code int, format string, args ...interface{}) error {
	writeHeader(w, code, "text/plain")
	var err error
	if len(args) > 0 {
		_, err = w.Write([]byte(fmt.Sprintf(format, args...)))
	} else {
		_, err = w.Write([]byte(format))
	}
	return err
}
