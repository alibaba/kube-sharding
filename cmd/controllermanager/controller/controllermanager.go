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

package controller

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/alibaba/kube-sharding/common"
	config2 "github.com/alibaba/kube-sharding/common/config"
	publisher "github.com/alibaba/kube-sharding/controller/service-publisher"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"

	"github.com/alibaba/kube-sharding/cmd/controllermanager/config"
	"github.com/alibaba/kube-sharding/cmd/controllermanager/options"
	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/shardgroup"
	aflag "github.com/alibaba/kube-sharding/pkg/ago-util/flag"
	logrotate "github.com/alibaba/kube-sharding/pkg/ago-util/logging/rotate"
	"github.com/alibaba/kube-sharding/pkg/ago-util/logging/zapm"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	carbonclientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	memclient "github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	"github.com/alibaba/kube-sharding/transfer"
	"github.com/docker/docker/pkg/term"
	"github.com/emicklei/go-restful"
	"github.com/gorilla/handlers"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	glog "k8s.io/klog"
)

var (
	zapLogConfig       = ""
	checkMemkubeConfig = true
	logFlags           = flag.NewFlagSet("log", flag.ExitOnError)
	globalKubeConfName = "hippo-c2-system/c2-global-config"
	accessLogFile      = ""
)

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.StringVar(&zapLogConfig, "zap-log-config", zapLogConfig, "zap log config")
	flagset.BoolVar(&checkMemkubeConfig, "check-memkube-config", checkMemkubeConfig, "check memkube config")
	flagset.StringVar(&globalKubeConfName, "global-kube-conf-name", globalKubeConfName, "global kube configmap name")
}

const (
	// ControllerManagerUserAgent is the userAgent name when starting controller managers.
	ControllerManagerUserAgent = "c2-scheduler"
	// AllNamespace express all name space
	AllNamespace = "all-namespaces"
	// EmptyNamespace express not set namespace
	EmptyNamespace = ""
	healthPath     = "/health_status"
	// MemKubeCheckerName MemKubeCheckerName
	MemKubeCheckerName = "memkube-config-checker"
)

// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand(controllers map[string]InitFunc) *cobra.Command {
	s, err := options.NewKubeControllerManagerOptions()
	if err != nil {
		panic("unable to initialize command options: " + err.Error())
	}
	cmd := &cobra.Command{
		Use:     ControllerManagerUserAgent,
		Long:    `C2 scheduler is a full set of controllers for Alibaba ASR services scheduling on kubernetes.`,
		Version: utils.GetVersion().String(),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := s.Config()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			if err := run(c, controllers, wait.NeverStop); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	initFlags(cmd, s, logFlags)
	return cmd
}

func initFlags(cmd *cobra.Command, s *options.KubeControllerManagerOptions, logFlags *flag.FlagSet) {
	namedFlagSets := s.Flags([]string{}, []string{})
	// NOTE: Put the controller specific flags into explicit flag set.
	logFlags.StringVar(&accessLogFile, "access-log-file", accessLogFile, "access log config file")
	glog.InitFlags(logFlags)
	logrotate.PatchKlogFlags(logFlags)
	namedFlagSets.FlagSet("log").AddGoFlagSet(logFlags)
	InitFlags(namedFlagSets.FlagSet("controllers"))
	common.InitFlags(namedFlagSets.FlagSet("common"))
	rollalgorithm.InitFlags(namedFlagSets.FlagSet("controllers"))
	carbonv1.InitFlags(namedFlagSets.FlagSet("controllers"))
	controller.InitFlags(namedFlagSets.FlagSet("controllers"))
	transfer.InitFlags(namedFlagSets.FlagSet("spec-transfer"))
	features.InitFlags(namedFlagSets.FlagSet("feature-gates"))
	publisher.InitFlags(namedFlagSets.FlagSet("publisher"))
	shardgroup.InitFlags(namedFlagSets.FlagSet("shardgroup"))

	options.PrintNamedFlagsetsSections(namedFlagSets, cmd, ControllerManagerUserAgent)
}

func terminalSize(w io.Writer) (int, int, error) {
	outFd, isTerminal := term.GetFdInfo(w)
	if !isTerminal {
		return 0, 0, fmt.Errorf("given writer is no terminal")
	}
	winsize, err := term.GetWinsize(outFd)
	if err != nil {
		return 0, 0, err
	}
	return int(winsize.Width), int(winsize.Height), nil
}

// resyncPeriod returns a function which generates a duration each time it is
// invoked; this is so that multiple controllers don't get into lock-step and all
// hammer the apiserver with list requests simultaneously.
func resyncPeriod(c *config.Config) func() time.Duration {
	return func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(c.Generic.MinResyncPeriod.Nanoseconds()) * factor)
	}
}

func newHealthHandler(mux *http.ServeMux) {
	h := handlers.LoggingHandler(logrotate.GetOrNop("access"), http.HandlerFunc(common.HealthCheck))
	mux.Handle(common.HealthCheckPath, h)
	mux.Handle(healthPath, h)
}

// run runs the KubeControllerManagerOptions.  This should never exit.
func run(c *config.Config, controllers map[string]InitFunc, stopCh <-chan struct{}) error {
	// To help debugging, immediately log version
	glog.Infof("%s", utils.GetVersion().String())

	if zapLogConfig != "" {
		err := zapm.InitDefault(zapLogConfig)
		if nil != err {
			return fmt.Errorf("init zap log failed: %v", err)
		}
	}
	if accessLogFile != "" {
		if err := logrotate.LoadFile(accessLogFile); err != nil {
			return fmt.Errorf("load access log file failed: %s, %v", accessLogFile, err)
		}
	}
	var mux = http.NewServeMux()
	newHealthHandler(mux)
	mux.HandleFunc("/flags", aflag.NewFlagHandler(logFlags))
	for p, h := range utils.PProfHandlers() {
		mux.Handle(p, h)
	}
	if 0 != c.Generic.Port {
		go func() {
			err := http.ListenAndServe(fmt.Sprintf("%s:%d", c.Generic.Address, c.Generic.Port), mux)
			if err != nil {
				glog.Fatalf("ListenAndServe error: %v", err)
				panic("ListenAndServe error")
			}
		}()
	}

	run := func(ctx context.Context) {
		glog.Infof("leaderelection started")
		h, ctx := utils.NewGracefulSigHandler(ctx)
		h.NotifySigs(syscall.SIGUSR1)
		Context, err := createControllerContext(c, ctx.Done())
		if err != nil {
			glog.Fatalf("error building controller context: %v", err)
		}

		if 0 != Context.DelayStart.Duration {
			time.Sleep(Context.DelayStart.Duration)
		}
		err = common.InitGlobalKubeConfiger(Context.Namespace, Context.KubeClient, stopCh)
		if err != nil {
			glog.Fatalf("error init global kubeconfiger: %v", err)
		}
		startLocalClient(Context, stopCh)
		if err := startControllers(Context, controllers, mux); err != nil {
			glog.Fatalf("error starting controllers: %v", err)
		}

		if err = startMonitor(Context, c, stopCh); err != nil {
			glog.Fatalf("error startMonitor: %v", err)
		}
		namespace := c.Namespace
		if namespace == "" {
			namespace = fmt.Sprintf("%s-%s", AllNamespace, c.Name)
		}
		metricgc.Start(stopCh)

		Context.InformerFactory.Start(Context.Stop)
		Context.CarbonInformerFactory.Start(Context.Stop)
		Context.SystemInformerFactory.Start(Context.Stop)

		close(Context.InformersStarted)

		ip, err := utils.GetLocalIP()
		if err != nil {
			panic("get local ip failed:" + err.Error())
		}
		startResourceServer(Context, fmt.Sprintf("%s:%d", ip, c.Generic.Port), mux, stopCh)

		h.Wait(time.Second*60*10, 0, Context.Waiter.Wait)
		os.Exit(0)
	}

	if !c.Generic.LeaderElection.LeaderElect {
		run(context.TODO())
		panic("unreachable")
	}

	id, err := os.Hostname()
	if err != nil {
		return err
	}

	restconfig := *c.Kubeconfig
	restconfig.Timeout = c.Generic.LeaderElection.RenewDeadline.Duration
	leaderElectionClient := clientset.NewForConfigOrDie(restclient.AddUserAgent(&restconfig, "leader-election"))

	rootClientBuilder := SimpleControllerClientBuilder{
		ClientConfig: c.Kubeconfig,
	}
	client := rootClientBuilder.ClientOrDie(ControllerManagerUserAgent)
	eventRecorder := createRecorder(client, ControllerManagerUserAgent)

	// add a uniquifier so that two processes on the same host don't accidentally both become active
	namespace := c.Namespace
	if AllNamespace == c.Namespace {
		namespace = "default"
	}
	id = getServerID(c)
	glog.Infof("Service register id: %s", id)
	name := ControllerManagerUserAgent + "-" + namespace
	if "" != c.Name {
		name = name + "-" + c.Name
	}
	rl, err := resourcelock.New(c.Generic.LeaderElection.ResourceLock,
		namespace,
		name,
		leaderElectionClient.CoreV1(),
		leaderElectionClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: eventRecorder,
		})
	if err != nil {
		glog.Fatalf("error creating lock: %s, %v", utils.ObjJSON(c.Generic.LeaderElection), err)
	}

	glog.Infof("elect info: %s", utils.ObjJSON(c.Generic.LeaderElection))
	var electionChecker *leaderelection.HealthzAdaptor
	electionChecker = leaderelection.NewLeaderHealthzAdaptor(time.Second * 20)
	leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: c.Generic.LeaderElection.LeaseDuration.Duration,
		RenewDeadline: c.Generic.LeaderElection.RenewDeadline.Duration,
		RetryPeriod:   c.Generic.LeaderElection.RetryPeriod.Duration,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				glog.Fatalf("leaderelection lost")
			},
		},
		WatchDog: electionChecker,
		Name:     ControllerManagerUserAgent + "-" + c.Name,
	})
	panic("unreachable")
}

// Context define the context of controller
type Context struct {
	// ClientBuilder will provide a client for this controller to use
	ClientBuilder ClientBuilder

	KubeClient    clientset.Interface
	CarbonClient  carbonclientset.Interface
	DynamicClient dynamic.Interface
	// InformerFactory gives access to informers for the controller.
	InformerFactory       informers.SharedInformerFactory
	ConfigInformerFactory informers.SharedInformerFactory
	SystemInformerFactory informers.SharedInformerFactory

	// CarbonInformerFactory gives access to carbon informers for the controller.
	CarbonInformerFactory carboninformers.SharedInformerFactory

	// Stop is the stop channel
	Stop   <-chan struct{}
	Waiter *sync.WaitGroup
	// 某些controller需要等到manager完全运行起来之后才能开始，InformersStarted 作为通知通道
	// InformersStarted is closed after all of the controllers have been initialized and are running.  After this point it is safe,
	// for an individual controller to start the shared informers. Before it is closed, they should not.
	InformersStarted chan struct{}

	// DeferredDiscoveryRESTMapper is a RESTMapper that will defer
	// initialization of the RESTMapper until the first mapping is
	// requested.
	RESTMapper *restmapper.DeferredDiscoveryRESTMapper
	// resyncPeriod generates a duration each time it is invoked; this is so that
	// multiple controllers don't get into lock-step and all hammer the apiserver
	// with list requests simultaneously.
	resyncPeriod func() time.Duration

	Controllers []string
	// LabelSelector and Namespace defines the scope of the controller
	LabelSelector string
	Namespace     string
	Cluster       string

	ConcurrentSyncs int32

	//ExtendConfig define the external config file path for controllers
	ExtendConfig string

	// delay duration before controllers start after became to leader
	DelayStart metav1.Duration

	LocalClient *memclient.LocalClient

	WriteLabels map[string]string
}

// InitFunc is used to launch a particular controller.  It may run additional "should I activate checks".
// Any error returned will cause the controller process to `Fatal`
// The bool indicates whether the controller was enabled.
type InitFunc func(ctx Context) (debuggingHandler http.Handler, enabled bool, err error)

// createControllerContext creates a context struct containing references to resources needed by the
// controllers such as the cloud provider and clientBuilder. rootClientBuilder is only used for
// the shared-informers client and token controller.
func createControllerContext(s *config.Config, stop <-chan struct{}) (Context, error) {
	if "" == s.Namespace {
		glog.Fatal("namespace can't be empty")
	}
	if AllNamespace == s.Namespace {
		s.Namespace = ""
	}
	if "" == s.Cluster {
		glog.Fatal("cluster should be hippo cluster name. ex: hippo_daily_7u")
	}
	// To make mem-kube rpc support corev1.Pod resource
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1.Pod{}, &v1.PodList{})

	var localClient *memclient.LocalClient
	if len(s.MemObjs) > 0 {
		var err error
		localClient, err = CreateLocalClient(s, stop, false)
		if err != nil {
			glog.Fatalf("Create localClient fatal %v", err)
			panic(err)
		}
	}
	writeLabels := map[string]string{}
	if s.WriteLabels != "" {
		for _, label := range strings.Split(s.WriteLabels, ",") {
			split := strings.Split(label, "=")
			if len(split) != 2 || len(validation.IsQualifiedName(split[0])) != 0 || len(validation.IsValidLabelValue(split[1])) != 0 {
				glog.Fatalf("invalid label %q", label)
				panic(fmt.Errorf("invalid label %q", label))
			}
			writeLabels[split[0]] = split[1]
		}
	}
	rootClientBuilder := SimpleControllerClientBuilder{
		ClientConfig: s.Kubeconfig,
		localClient:  localClient,
		memObjs:      s.MemObjs,
	}
	var clientBuilder = rootClientBuilder

	kubeClient := clientBuilder.ClientOrDie("shared-informers")
	carbonClient := clientBuilder.CarbonClientOrDie("shared-informers")
	dynamicClient := dynamic.NewForConfigOrDie(s.Kubeconfig)
	err := checkMemKubeConfig(s, kubeClient)
	if nil != err {
		return Context{}, err
	}

	sharedInformers := informers.NewFilteredSharedInformerFactory(kubeClient, resyncPeriod(s)(), s.Namespace, s.TweakListOptionsFunc)
	carbonSharedInformers := carboninformers.NewFilteredSharedInformerFactory(carbonClient, resyncPeriod(s)(), s.Namespace, s.TweakListOptionsFunc)
	systemSharedInformers := informers.NewFilteredSharedInformerFactory(kubeClient, resyncPeriod(s)(), carbonv1.SystemNamespace, s.TweakListOptionsFunc)

	// Use a discovery client capable of being refreshed.
	discoveryClient := rootClientBuilder.ClientOrDie("controller-discovery")
	cachedClient := cacheddiscovery.NewMemCacheClient(discoveryClient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClient)

	go wait.Until(func() {
		restMapper.Reset()
	}, 30*time.Second, stop)
	// If apiserver is not running we should wait for some time and fail only then. This is particularly
	// important when we start apiserver and controller manager at the same time.
	if err := waitForAPIServer(kubeClient, 10*time.Second); err != nil {
		return Context{}, fmt.Errorf("failed to wait for apiserver being healthy: %v", err)
	}

	ctx := Context{
		RESTMapper:            restMapper,
		ClientBuilder:         clientBuilder,
		KubeClient:            kubeClient,
		CarbonClient:          carbonClient,
		DynamicClient:         dynamicClient,
		InformerFactory:       sharedInformers,
		CarbonInformerFactory: carbonSharedInformers,
		SystemInformerFactory: systemSharedInformers,
		Stop:                  stop,
		InformersStarted:      make(chan struct{}),
		resyncPeriod:          resyncPeriod(s),
		Controllers:           s.Generic.Controllers,
		ConcurrentSyncs:       s.ConcurrentSyncs,
		ExtendConfig:          s.ExtendConfig,
		Waiter:                &sync.WaitGroup{},
		DelayStart:            s.DelayStart,
		Cluster:               s.Cluster,
		Namespace:             s.Namespace,
		LocalClient:           localClient,
		WriteLabels:           writeLabels,
	}
	return ctx, nil
}

func startControllers(ctx Context, controllers map[string]InitFunc, mux *http.ServeMux) error {
	for i := range ctx.Controllers {
		controllerName := ctx.Controllers[i]
		if initFn, ok := controllers[ctx.Controllers[i]]; ok {
			glog.Infof("Starting %q", controllerName)
			debugHandler, started, err := initFn(ctx)
			if err != nil {
				glog.Errorf("Error starting %q", controllerName)
				return err
			}
			if !started {
				glog.Warningf("Skipping %q", controllerName)
				continue
			}

			if debugHandler != nil && mux != nil {
				basePath := "/debug/controllers/" + controllerName
				mux.Handle(basePath+"/", http.StripPrefix(basePath, debugHandler))
			}
			glog.Infof("Started %q", controllerName)
		}
	}

	return nil
}

func initGlobalKubeConfiger(ctx Context, stopCh <-chan struct{}) error {
	conf := strings.Split(globalKubeConfName, "/")
	if len(conf) != 2 || conf[0] == "" || conf[1] == "" {
		return fmt.Errorf("invalid global-kube-conf-name, it should be $namespace/$config")
	}
	factory := config2.NewKubeConfigerFactory(ctx.KubeClient, conf[0], nil)
	factory.Start(stopCh)
	common.SetGlobalConfiger(factory.Configer(conf[1]), ctx.Namespace)
	return nil
}

func startLocalClient(ctx Context, stopCh <-chan struct{}) {
	if ctx.LocalClient == nil {
		return
	}
	err := ctx.LocalClient.Start(stopCh)
	if err != nil {
		glog.Fatalf("LocalClient start err %v", err)
		panic("LocalClient start err")
	}
	glog.Info("Started local Client")
}

func startMonitor(ctx Context, c *config.Config, stopCh <-chan struct{}) error {
	var selector labels.Selector
	if c.LabelSelector != "" {
		if s, err := labels.Parse(c.LabelSelector); err != nil {
			return err
		} else {
			selector = s
		}
	}
	go controller.StartMonitor(ctx.CarbonInformerFactory, ctx.InformerFactory, c.Namespace, selector)
	return nil
}

func startResourceServer(ctx Context, spec string, mux *http.ServeMux, stopCh <-chan struct{}) error {
	var webSvc *restful.WebService
	var resScheme *mem.ResourceScheme
	if ctx.LocalClient != nil {
		glog.Info("Open memkube service")
		resScheme = mem.NewResourceScheme(scheme.Scheme)
		service := rpc.NewService(ctx.LocalClient, resScheme, scheme.Codecs, &rpc.ServiceOptions{})
		webSvc = service.GetHandlerService()
	}
	wsContainer := restful.NewContainer()
	wsContainer.EnableContentEncoding(true)
	if webSvc != nil {
		wsContainer.Add(webSvc)
	}
	mux.Handle("/", handlers.LoggingHandler(logrotate.GetOrNop("access"), wsContainer))
	glog.Info("Started resource server")
	return nil
}

func isControllerEnable(ctx Context, controllerName string) bool {
	for _, curControllerName := range ctx.Controllers {
		if curControllerName == controllerName {
			return true
		}
	}

	return false
}

// waitForAPIServer waits for the API Server's /healthz endpoint to report "ok" with timeout.
func waitForAPIServer(client clientset.Interface, timeout time.Duration) error {
	var lastErr error

	err := wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		healthStatus := 0
		result := client.Discovery().RESTClient().Get().AbsPath("/healthz").Do(context.Background()).StatusCode(&healthStatus)
		if result.Error() != nil {
			lastErr = fmt.Errorf("failed to get apiserver /healthz status: %v", result.Error())
			return false, nil
		}
		if healthStatus != http.StatusOK {
			content, _ := result.Raw()
			lastErr = fmt.Errorf("APIServer isn't healthy: %v", string(content))
			glog.Warningf("APIServer isn't healthy yet: %v. Waiting a little while.", string(content))
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return fmt.Errorf("%v: %v", err, lastErr)
	}

	return nil
}

func createRecorder(kubeClient clientset.Interface, userAgent string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	return eventBroadcaster.NewRecorder(legacyscheme.Scheme, v1.EventSource{Component: userAgent})
}

func checkMemKubeConfig(s *config.Config, kubeClient clientset.Interface) error {
	if !checkMemkubeConfig {
		return nil
	}
	name := MemKubeCheckerName
	if "" != s.Name {
		name = MemKubeCheckerName + "-" + s.Name
	}
	// memkube checker must set namespace, use system-namespace if not exist
	checkedNamespace := s.Namespace
	if s.Namespace == "" {
		checkedNamespace = carbonv1.SystemNamespace
	}
	configmap, err := kubeClient.CoreV1().ConfigMaps(checkedNamespace).Get(context.Background(), name, metav1.GetOptions{})
	if nil != err {
		if errors.IsNotFound(err) {
			var configmap = &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: checkedNamespace,
				},
				Data: map[string]string{
					"MemFsURL":           s.MemFsURL,
					"MemObjs":            strings.Join(s.MemObjs, ","),
					"MemBatcherGroupKey": s.MemBatcherGroupKey,
				},
			}
			_, err := kubeClient.CoreV1().ConfigMaps(checkedNamespace).Create(context.Background(), configmap, metav1.CreateOptions{})
			return err
		}
		return err
	}

	if configmap.Data["MemFsURL"] != s.MemFsURL || configmap.Data["MemObjs"] != strings.Join(s.MemObjs, ",") || configmap.Data["MemBatcherGroupKey"] != s.MemBatcherGroupKey {
		err = fmt.Errorf("memkubeconfig changed MemFsURL: %s, %s, MemObjs: %s, %s,MemBatcherGroupKey: %s,%s", configmap.Data["MemFsURL"], s.MemFsURL, configmap.Data["MemObjs"], s.MemObjs, configmap.Data["MemBatcherGroupKey"], s.MemBatcherGroupKey)
		glog.Error(err)
		return err
	}
	return nil
}
