# DEV

```
cd $GOPATH/src/

mkdir -p gitlab.alibaba-inc.com/kubeshard

cd github.com/alibaba/kube-sharding.git

git clone <http://github.com/alibaba/kube-sharding.git>

```

## 简介

参考k8s controller实现方式编写。

新增crd定义在 pkg/apis/carbon/v1

此项目包括多个controller: shardgroup,rollingset,publisher,worker,healthchecker。 添加controller和crd(customer resource define)

rollingset 参考carbon rollingset，调度replica(基本调度单元)，实现原地rolling。

replica 参考carbon replicaNode，作为被rollingset调度的基本单元，封装具体资源申请/释放/recover/offline等功能。

publisher 负责服务挂载。

healthchecker 负责健康检查。

captain/carbon/fiber等不同的调度器通过各自实现replica controller，实现各自调度replica的管理，并且共享rollingset controller。

## make

在根目录下执行 make

## 运行

```
./target/c2/c2-linux-amd64 --kubeconfig="{}" --concurrent=10 --delay-start=10s --controllers shardgroup,rollingset,publisher,healthcheck,worker --namespace test-namespace --spec:scheduler-name=ack --port 8989 --kube-api-qps=100 --kube-api-burst=300 --log_dir=./logs --leader-elect-resource-lock=leases
```

