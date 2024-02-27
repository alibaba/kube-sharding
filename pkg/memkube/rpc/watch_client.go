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
	"io/ioutil"
	"mime"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/watch"
	restclientwatch "k8s.io/client-go/rest/watch"
	glog "k8s.io/klog"
)

type watchClient struct {
	request
	client     *http.Client
	negotiator runtime.ClientNegotiator
	accepts    string
}

// Copy from client-go request.go
func (wc *watchClient) Watch() (watch.Interface, error) {
	url := wc.URL()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Don't compress the response, because the watch server requires a http.Flusher response writer
	req.Header.Set("Accept-Encoding", "")
	if wc.accepts != "" {
		req.Header.Set("Accept", wc.accepts)
	}
	glog.Infof("Do watch request: %s, accepts: %s", url, wc.accepts)
	resp, err := wc.client.Do(req)
	if err != nil {
		// The watch stream mechanism handles many common partial data errors, so closed
		// connections can be retried in many cases.
		if net.IsProbableEOF(err) || net.IsTimeout(err) {
			return watch.NewEmptyWatch(), nil
		}
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, wc.transformError(resp, req)
	}
	contentType := resp.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		glog.Errorf("Unexpected content type from the server: %q: %v", contentType, err)
		return nil, err
	}
	objectDecoder, streamingSerializer, framer, err := wc.negotiator.StreamDecoder(mediaType, params)
	if err != nil {
		glog.Errorf("Create StreamDecoder error: %v", err)
		return nil, err
	}

	frameReader := framer.NewFrameReader(resp.Body)
	watchEventDecoder := streaming.NewDecoder(frameReader, streamingSerializer)

	return watch.NewStreamWatcher(
		restclientwatch.NewDecoder(watchEventDecoder, objectDecoder),
		// use 500 to indicate that the cause of the error is unknown - other error codes
		// are more specific to HTTP interactions, and set a reason
		errors.NewClientErrorReporter(http.StatusInternalServerError, http.MethodGet, "ClientWatchDecoding"),
	), nil
}

// TODO: handle some other binary protocol
func (wc *watchClient) transformError(resp *http.Response, req *http.Request) error {
	var body string
	if resp.Body != nil {
		data, err := ioutil.ReadAll(resp.Body)
		switch err.(type) {
		case nil:
			body = string(data)
			glog.Errorf("Got http client error with body: %s", body)
			return fmt.Errorf("http client error [%d] with body: %s", resp.StatusCode, body)
		default:
			glog.Errorf("Unexpected error when reading response body: %v", err)
			unexpectedErr := fmt.Errorf("unexpected error when reading response body. Please retry. Original error: %v", err)
			return unexpectedErr
		}
	}
	return fmt.Errorf("empty http response error, http code: %d", resp.StatusCode)
}
