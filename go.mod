module github.com/alibaba/kube-sharding

go 1.21

require (
	github.com/bluele/gcache v0.0.0-20190518031135-bc40bd653833
	github.com/deckarep/golang-set v0.0.0-20180927150649-699df6a3acf6
	github.com/docker/docker v20.10.12+incompatible
	github.com/emicklei/go-restful v2.12.0+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang/glog v1.0.0
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.3
	github.com/gorilla/handlers v1.5.1
	github.com/ivpusic/grpool v1.0.0
	github.com/openkruise/kruise-api v0.8.0-1.18
	github.com/pborman/uuid v1.2.0
	github.com/prometheus/client_golang v1.12.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.0
	go.uber.org/zap v1.19.0
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8
	k8s.io/apimachinery v0.24.16
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/component-base v0.24.16
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.60.1
	k8s.io/kubernetes v1.18.9
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/gomega v1.27.4 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/samuel/go-zookeeper v0.0.0-20180130194729-c4fab1ac1bec // indirect
	github.com/stretchr/objx v0.4.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220107163113-42d7afdf6368 // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/component-helpers v0.0.0 // indirect
	k8s.io/kube-openapi v0.0.0-20220328201542-3ee0da9b0b42 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0
	github.com/abronan/valkeyrie v0.0.0-20190802193736-ed4c4a229894
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-zookeeper/zk v1.0.3
	github.com/hashicorp/go-version v1.6.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jinzhu/copier v0.4.0
	github.com/jinzhu/gorm v1.9.16
	github.com/kevinlynx/log4go v0.0.0-20190406053902-f281083bbfae
	github.com/lib/pq v1.3.0 // indirect
	github.com/liip/sheriff v0.0.0-00010101000000-000000000000
	github.com/toolkits/file v0.0.0-20160325033739-a5b3c5147e07 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/oauth2 v0.0.0-20220608161450-d0670ef3b1eb // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	k8s.io/api v0.24.16
	k8s.io/apiserver v0.24.16
)

replace go.etcd.io/etcd => go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.7.0
	github.com/abronan/valkeyrie => github.com/kevinlynx/valkeyrie v0.0.0-20200617114326-b19ce8bcb189
	github.com/golang/mock => github.com/golang/mock v1.4.4
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.1.0
	github.com/kubernetes-incubator/reference-docs => github.com/kubernetes-sigs/reference-docs v0.0.0-20170929004150-fcf65347b256
	github.com/liip/sheriff => github.com/kevinlynx/sheriff v0.0.0-20210729085801-ca229732c4e1
	github.com/markbates/inflect => github.com/markbates/inflect v1.0.4
	github.com/onsi/ginkgo => github.com/phil9909/ginkgo v1.16.6-0.20220211153547-67da0e38b07d
	github.com/spf13/cobra v1.0.0 => github.com/spf13/cobra v0.0.5
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.0.0-20190226205417-e64efc72b421
)

// Kubernetes 0.24.16
replace (
	github.com/docker/docker => github.com/docker/docker v20.10.6+incompatible
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
	k8s.io/api => k8s.io/api v0.24.16
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.24.16
	k8s.io/apimachinery => k8s.io/apimachinery v0.24.16
	k8s.io/apiserver => k8s.io/apiserver v0.24.16
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.24.16
	k8s.io/client-go => k8s.io/client-go v0.24.16
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.24.16
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.24.16
	k8s.io/code-generator => k8s.io/code-generator v0.24.16
	k8s.io/component-base => k8s.io/component-base v0.24.16
	k8s.io/component-helpers => k8s.io/component-helpers v0.24.16
	k8s.io/controller-manager => k8s.io/controller-manager v0.24.16
	k8s.io/cri-api => k8s.io/cri-api v0.24.16
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.24.16
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.24.16
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.24.16
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.24.16
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.24.16
	k8s.io/kubectl => k8s.io/kubectl v0.24.16
	k8s.io/kubelet => k8s.io/kubelet v0.24.16
	k8s.io/kubernetes => k8s.io/kubernetes v1.24.16
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.24.16
	k8s.io/metrics => k8s.io/metrics v0.24.16
	k8s.io/mount-utils => k8s.io/mount-utils v0.24.16
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.24.16
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.24.16
)
