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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/emicklei/go-restful"
	glog "k8s.io/klog"
)

// ProxyTo proxies the request to the host's server directly.
// It tries its best to keep the origal data from host's server.
func ProxyTo(req *restful.Request, responseWriter http.ResponseWriter, host string) (int, error) {
	// 复制一个request 避免原request无法读取
	var proxyRequest *http.Request
	r := req.Request
	// 复制body
	if r.Body != nil {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			glog.Errorf("read request body error:%v", err)
			return http.StatusInternalServerError, err
		}
		r.Body.Close()
		glog.Infof("proxy bodyBytes:%s %d", string(bodyBytes), len(bodyBytes))
		proxyRequest, _ = http.NewRequest(r.Method, r.RequestURI, ioutil.NopCloser(bytes.NewReader(bodyBytes)))
		r.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
	} else {
		proxyRequest, _ = http.NewRequest(r.Method, r.RequestURI, nil)
	}
	// 复制header信息
	if !strings.HasPrefix(host, "http://") {
		host = "http://" + host
	}
	hosturl, err := url.Parse(host)
	if nil == err {
		host = hosturl.Host
	}
	proxyRequest.URL.Host = host
	proxyRequest.URL.Scheme = "http"
	proxyRequest.URL.RawQuery = r.URL.RawQuery
	proxyRequest.URL.Path = r.URL.Path

	for k, v := range r.Header {
		for _, vv := range v {
			proxyRequest.Header.Add(k, vv)
		}
	}
	// 支持长连接
	proxyRequest.Header.Del("Connection")
	proxyRequest.Header.Set("Connection", "Keep-Alive")
	glog.Infof("proxy: request to %#q, url %#q, method :%s, connection :%s", host, proxyRequest.URL.String(), r.Method,
		proxyRequest.Header.Get("Connection"))

	// 发送请求
	resp, err := defaultHTTPClient.Do(proxyRequest)
	if err != nil {
		glog.Errorf("proxy redirect to host[%v] error: %v", host, err)
		utils.RenderPlain(responseWriter, http.StatusInternalServerError, err.Error())
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()
	if glog.V(4) {
		glog.Infof("proxy: response from %#q %s OK, status %v", host, proxyRequest.URL.String(), resp.Status)
	}

	// 复制response
	for k := range responseWriter.Header() {
		delete(responseWriter.Header(), k)
	}
	for k, v := range resp.Header {
		responseWriter.Header()[k] = v
	}

	responseWriter.WriteHeader(resp.StatusCode)
	if nil != resp.Body {
		io.Copy(responseWriter, resp.Body)
	}
	// 返回
	return resp.StatusCode, nil
}

//QueryParameter QueryParameter
func QueryParameter(req *http.Request, paramName string) string {
	newValues, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		glog.Error(err)
	}
	parameters := make(map[string]string)
	for k, v := range newValues {
		parameters[k] = v[0]
	}
	return parameters[paramName]
}

var (
	// HealthCheckPath HealthCheckPath
	HealthCheckPath = "/status.html"
)

// HealthCheck HealthCheck
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	path := staticAssets + upath
	http.ServeFile(w, r, path)
}
