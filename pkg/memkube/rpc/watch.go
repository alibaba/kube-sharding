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
	"net"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	glog "k8s.io/klog"
)

func serveWatch(watcher watch.Interface, ns runtime.NegotiatedSerializer, gvk schema.GroupVersionKind, req *http.Request, w http.ResponseWriter, timeout time.Duration) {
	defer watcher.Stop()
	scope := handlers.RequestScope{
		Serializer: ns,
		Kind:       gvk,
	}
	serializer, err := negotiation.NegotiateOutputMediaTypeStream(req, ns, negotiation.DefaultEndpointRestrictions)
	if err != nil {
		responsewriters.ErrorNegotiated(err, scope.Serializer, scope.Kind.GroupVersion(), w, req)
		return
	}
	framer := serializer.StreamSerializer.Framer
	streamSerializer := serializer.StreamSerializer.Serializer
	embedded := serializer.Serializer
	if framer == nil {
		err := fmt.Errorf("no framer defined for %q available for embedded encoding", serializer.MediaType)
		responsewriters.ErrorNegotiated(err, scope.Serializer, scope.Kind.GroupVersion(), w, req)
		return
	}
	encoder := ns.EncoderForVersion(streamSerializer, scope.Kind.GroupVersion())

	// find the embedded serializer matching the media type
	embeddedEncoder := ns.EncoderForVersion(embedded, scope.Kind.GroupVersion())

	mediaType := serializer.MediaType
	if mediaType != runtime.ContentTypeJSON {
		mediaType += ";stream=watch"
	}
	server := &handlers.WatchServer{
		Watching:        watcher,
		Scope:           &scope,
		MediaType:       mediaType,
		Framer:          framer,
		Encoder:         encoder,
		EmbeddedEncoder: embeddedEncoder,
		Fixup: func(o runtime.Object) runtime.Object {
			return o
		},
		// Disable timeout watching for memkube because memkube can't provide a real increment resource version
		// issue #108763 (http://github.com/alibaba/kube-sharding/issues/108763)
		// The watcher will timeout here.
		TimeoutFactory: &realTimeoutFactory{0},
	}

	server.ServeHTTP(w, req)
	glog.Infof("Serve watching finished for: %s", getRequestHost(req))
}

// nothing will ever be sent down this channel
var neverExitWatch <-chan time.Time = make(chan time.Time)

// realTimeoutFactory implements timeoutFactory
type realTimeoutFactory struct {
	timeout time.Duration
}

// TimeoutCh returns a channel which will receive something when the watch times out,
// and a cleanup function to call when this happens.
func (w *realTimeoutFactory) TimeoutCh() (<-chan time.Time, func() bool) {
	if w.timeout == 0 {
		return neverExitWatch, func() bool { return false }
	}
	t := time.NewTimer(w.timeout)
	return t.C, t.Stop
}

func getRequestHost(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}
	return host
}
