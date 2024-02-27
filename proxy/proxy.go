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

package proxy

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba/kube-sharding/common"
	aflag "github.com/alibaba/kube-sharding/pkg/ago-util/flag"
	logrotate "github.com/alibaba/kube-sharding/pkg/ago-util/logging/rotate"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils/restutil"
	"github.com/alibaba/kube-sharding/proxy/httph"
	"github.com/alibaba/kube-sharding/proxy/service"
	restful "github.com/emicklei/go-restful"
	"github.com/gorilla/handlers"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	glog "k8s.io/klog"
)

var opts struct {
	port           string
	cluster        string
	kubeConfigFile string
	hippoApiserver string
	contentType    string
	disableKmon    bool
	kubeOnly       bool
	slotService    bool
	logConfig      string
	authPort       string
	proxiedGVRList []string
}

const (
	kmonDefaultAddress = "localhost"
	kmonDefaultPort    = "4141"
)

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.StringVar(&opts.port, "port", "", "Service listen port.")
	flagset.StringVar(&opts.cluster, "cluster", "", "Hippo cluster id.")
	flagset.StringVar(&opts.hippoApiserver, "hippo-apiserver", "", "hippo apiserver.")
	flagset.StringVar(&opts.kubeConfigFile, "kubeconfig", "", "kube apiserver client config file")
	flagset.StringVar(&opts.contentType, "kube-api-content-type", "application/vnd.kubernetes.protobuf", "Content type of requests sent to apiserver.")
	flagset.StringVar(&opts.logConfig, "log-config", "", "Log config file")
	flagset.BoolVar(&opts.disableKmon, "disable-kmon", false, "disable kmonitor report")
	flagset.BoolVar(&opts.kubeOnly, "kube-only", false, "Cache kube resource only")
	flagset.StringVar(&opts.authPort, "auth-port", "", "Service auth listen port.")
	flagset.BoolVar(&opts.slotService, "slot-service", false, "serve as hippo adapter")
	flagset.StringSliceVar(&opts.proxiedGVRList, "proxied-gvr", []string{}, "gvr list allowed be proxied to k8s api-server, example: --proxied-gvr='scales.app.scaler.io/v1,groups.app.scaler.io/v1'")
	service.InitFlags(flagset)
}

// Run start the service
func Run() error {
	glog.Infof("carbon-proxy version: %s", utils.GetVersion().String())
	if err := logrotate.LoadFile(opts.logConfig); err != nil && opts.logConfig != "" {
		panic("load log config failed:" + err.Error())
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	metricgc.Start(stopCh)
	server, err := startHTTPServer(opts.port, false, stopCh)
	if err != nil {
		return err
	}
	var serverAuth *http.Server
	if opts.authPort != "" {
		glog.Infof("regist auth http server. port:%s", opts.authPort)
		serverAuth, err = startHTTPServer(opts.authPort, true, stopCh)
		if err != nil {
			return err
		}
	}

	h, ctx := utils.NewGracefulSigHandler(context.Background())
	h.NotifySigs()
	<-ctx.Done()
	glog.Info("Shutdown server")
	server.Shutdown(context.Background())
	if serverAuth != nil {
		glog.Info("Shutdown server with auth")
		serverAuth.Shutdown(context.Background())
	}
	return nil
}

func startHTTPServer(port string, auth bool, stopCh <-chan struct{}) (*http.Server, error) {
	wsContainer := restful.NewContainer()
	wsContainer.DoNotRecover(false)

	if err := registerService(wsContainer, auth, stopCh); err != nil {
		return nil, err
	}
	var mux http.ServeMux
	mux.Handle("/", handlers.LoggingHandler(logrotate.GetOrNop("access"), wsContainer))
	mux.Handle(common.HealthCheckPath, handlers.LoggingHandler(logrotate.GetOrNop("access"), http.HandlerFunc(common.HealthCheck)))
	if !auth {
		mux.HandleFunc("/flags", aflag.NewFlagHandler(flag.CommandLine))
		for p, h := range utils.PProfHandlers() {
			mux.Handle(p, h)
		}
	}

	if !strings.Contains(port, ":") {
		port = ":" + port
	}
	server := &http.Server{Addr: port, Handler: &mux}
	go func() {
		glog.Infof("Start listen %s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			panic("Listen error: " + err.Error())
		}
	}()
	return server, nil
}

func registerService(wsContainer *restful.Container, auth bool, stopCh <-chan struct{}) error {
	wsContainer.Router(restful.CurlyRouter{})
	restutil.PatchContainer(wsContainer, &restutil.PatchContainerOptions{
		LoggingFunc: globalLogging,
	})
	httph.InitRoute(opts.cluster)

	serviceFac, err := service.NewFactory(opts.cluster, opts.kubeConfigFile, opts.contentType, stopCh)
	if err != nil {
		return err
	}

	// groups
	groupHandlers, err := httph.NewGroupHandler(serviceFac.GroupService)
	if err != nil {
		return err
	}
	wsContainer.Add(groupHandlers.NewWebService())
	return nil
}

func globalLogging(req *restful.Request, resp *restful.Response, latency time.Duration) {
	reportQueryCounter(opts.cluster, req.Request.URL.Path, req.Request.Method, resp.StatusCode())
	reportQueryLatency(opts.cluster, req.Request.URL.Path, req.Request.Method, resp.StatusCode(), latency.Nanoseconds())
	if glog.V(4) {
		glog.Infof("process request %s, %s, %s, %d ms", opts.cluster, req.Request.URL.String(), req.Request.Method, latency.Nanoseconds()/1e6)
	}
}

func parseGVRs(rawGVR []string) []schema.GroupVersionResource {
	ret := make([]schema.GroupVersionResource, 0, len(rawGVR))
	for i := range rawGVR {
		if gvr, err := parseGVR(rawGVR[i]); err == nil {
			ret = append(ret, gvr)
		} else {
			glog.Errorf("parse gvr err:%v, raw:%s", err, rawGVR[i])
		}
	}
	return ret
}

// pattern: resouce.group/version, example: scales.app.scaler.io/v1
func parseGVR(rawGVR string) (schema.GroupVersionResource, error) {
	gvr := schema.GroupVersionResource{}
	segs := strings.Split(rawGVR, "/")
	if len(segs) != 2 {
		return gvr, errors.New("invalid gvr raw string")
	}
	grRaw, versionRaw := segs[0], segs[1]
	gr := schema.ParseGroupResource(grRaw)
	return schema.GroupVersionResource{
		Group:    gr.Group,
		Resource: gr.Resource,
		Version:  versionRaw,
	}, nil
}
