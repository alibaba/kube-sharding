# kube-sharding 的职责和架构
![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2024/png/97471/1708941065617-77d81ab4-01ef-48c9-8a15-4581e901d81f.png#clientId=u47d5363b-c298-4&from=paste&height=540&id=ufc7431b1&originHeight=1080&originWidth=1920&originalType=binary&ratio=2&rotation=0&showTitle=false&size=239872&status=done&style=none&taskId=u0a29c5f0-9ec3-41c3-88b3-a11538e80b0&title=&width=960)
kube-sharding是一个业务编排controller，编排对象是需要加载大规模数据的搜索推荐引擎，或者大模型推理业务。支持业务进行分片以及分片后的对齐发布。支持业务单独发布索引/算子/模型数据，使数据变更在满足maxUnavailable的条件下安全的进行。
![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2024/png/97471/1708324573688-63f4d376-cf96-4b46-9d4a-6b64b7f79802.png#clientId=ucaed3f0e-093a-4&from=paste&height=540&id=u610fec37&originHeight=1080&originWidth=1920&originalType=binary&ratio=2&rotation=0&showTitle=false&size=630922&status=done&style=none&taskId=u7e536b9a-d6bd-4c2d-b1dd-e2815ea0c28&title=&width=960)
在k8s的架构下，kube-sharding也是由多个controller组成，通过不同controller的组成来实现一个整体的调度目标。
# kube-sharding的调度策略
## **rollingset 单列业务发布**
#### **社区的deployment的调度模式**
![](https://intranetproxy.alipay.com/skylark/lark/0/2024/webp/97471/1708324912041-bddcb40b-8805-441c-8545-1b5b4b1216c4.webp#clientId=ucaed3f0e-093a-4&from=paste&height=229&id=ud58c4d04&originHeight=618&originWidth=1286&originalType=url&ratio=2&rotation=0&showTitle=false&status=done&style=none&taskId=u132b1c6e-f3b3-4e27-93ce-790a78b449c&title=&width=477)

1. deployment 控制不同版本的replicaset的replicas数量。
2. replicaset对自己控制的pod进行增删来实现pod版本的替换。

这种模式实现比较简单，优雅，但是也存在着一些不足：没有对调度对象进行抽象，从而只能调度pod这一种资源，同时难以实现pod原地升级。
#### **c2的rollingset的调度模式**
![](https://intranetproxy.alipay.com/skylark/lark/0/2024/webp/97471/1708324912226-250a2de8-b1ca-4a84-bcf3-9efaa233aa20.webp#clientId=ucaed3f0e-093a-4&from=paste&height=241&id=u32347a9d&originHeight=652&originWidth=1452&originalType=url&ratio=2&rotation=0&showTitle=false&status=done&style=none&taskId=ue9a1aa21-00c9-419a-80a2-9a767583062&title=&width=537)
和deployment有两点显著不同

1. 没有直接调度pod资源，而是调度了一个抽象的replica资源，这样实现非常有利于调度能力的扩展：
   1. 对于搜推引擎来说，replica里包括了pod和引擎的索引数据，可以分别发布pod或者索引，从而轻量化的实现索引数据的发布。
   2. 对于普通单列大模型来说，索引数据变成了LoRA和模型，c2可以单独发布LoRA，从而轻松实现大模型的微调。
   3. 对于分布式推理，replica又可以实现为包含多列的业务的gang replica，这一点后续会详细介绍。
2. rollingset没有通过ReplicaSet的方式来控制具体replica的版本，而是通过一套算法直接输出其管理的所有replica的版本。这样实现有助于replica的抽象，以及原地升级的实现。由于gpu资源紧张，以及模型下载慢导致启动慢，原地升级是十分有必要的，c2 rollingset在发布业务过程中可以支持：索引/模型/LoRA等业务数据变更，image/command/args等容器层变更，以及cpu/gpu/mem等资源变更在内的几乎全变部更场景的原地升级，助力业务平滑发布。
#### **kube-sharding对于普通的单列无状态业务的发布过程**
![](https://intranetproxy.alipay.com/skylark/lark/0/2024/webp/97471/1708325024722-caaaf4f1-95f2-42fd-93f7-22feca62c56c.webp#clientId=ucaed3f0e-093a-4&from=paste&id=u583db3ca&originHeight=900&originWidth=1600&originalType=url&ratio=2&rotation=0&showTitle=false&status=done&style=none&taskId=u2c38915a-1aac-4cb2-9607-a95bc6853f8&title=)
如图所示：c2 rollingset对于普通单列无状态业务的调度也是基于maxUnavailable对业务进行分批发布。 不同于其它controller，c2并没有直接调度pod，而是调度了一个抽象的Replica资源，这个Replica可以代表一个pod，也可以代表其它资源。基于这个特点，我们对c2 rollingset进行了简单的扩展：
## **rollingset 多列分布式业务发布**
#### **行列式服务**
首先我们提出一个行列式服务的概念
当一个业务消耗的资源足够大，大到单节点放不下时就需要做分布式服务。分布式服务有两个维度：

1. 内存/显存/磁盘等资源不足时需要做partition，拆为多列。如图所示就是我们把一个完整的单节点服务做partition，拆为多列。拆分后我们把这一组节点称之为1行n列。对应到大模型中，如果模型参数太大，单节点显存不足时就需要做partition拆为多列。

![](https://intranetproxy.alipay.com/skylark/lark/0/2024/webp/97471/1708325041582-1d0adc57-8d6b-491e-a2d7-61b74f483855.webp#clientId=ucaed3f0e-093a-4&from=paste&height=47&id=u7b27e05c&originHeight=127&originWidth=1600&originalType=url&ratio=2&rotation=0&showTitle=false&status=done&style=none&taskId=u7e8c3135-2f6d-43c7-9c32-433e6fc6b8e&title=&width=586)

2. 当cpu/gpu等计算资源不足时就需要扩容为多个同构replica来分担计算需求，我们称之为扩行。比如社区的deployment就是部署同构无状态服务的，deployment中每一个节点都是一行。再带回到我们的场景下，我们将一个row复制为多row，就形成了如下图所示的行列式结构。图中一个row就是一行，对应到大模型中就是一个完整的pipeline，加载一整个模型，包含多个partition。对row扩容就形成了多行多列：

![](https://intranetproxy.alipay.com/skylark/lark/0/2024/webp/97471/1708325041777-50df32e6-3be7-4cee-b953-06c54377dc9f.webp#clientId=ucaed3f0e-093a-4&from=paste&height=229&id=u549dcc6a&originHeight=617&originWidth=1600&originalType=url&ratio=2&rotation=0&showTitle=false&status=done&style=none&taskId=ueebf0fe6-d39e-4e53-985d-481bf6ba1b9&title=&width=594)
**行列式调度，给了业务无限的服务扩展能力。**
#### **支持分布式推理的gang rolling**
![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2024/png/97471/1708325069403-7ea7629b-f8bd-49c3-b8e6-ffec803b435a.png#clientId=ucaed3f0e-093a-4&from=paste&height=540&id=u98da5b8d&originHeight=1080&originWidth=1920&originalType=binary&ratio=2&rotation=0&showTitle=false&size=321372&status=done&style=none&taskId=ua7295252-3ca5-4cb0-b772-c8e5c2b44ec&title=&width=960)
    我们将上面的一个row称为一个gang replica，这里的gang和资源分配中的gang有重名，但含义是不同的，这里的gang replica代表一个多partition组成的，需要同时进行部署/发布的多个节点的集合。
    然后我们将rollingset调度的replica实现为一个gang replica，具体到大模型场景下，它就代表了一个pipeline。此时我们再对这种gang replica进行调度，就自然支持了多列业务严格对齐的部署/发布/扩缩容。
    gang rolling适用于每一行都必须严格对齐发布的场景，是因为类似大模型分布式推理这种业务是有状态的，一个gang内的节点必须一起发布。但是有很多需要分片的业务是无状态的，比如搜索引擎，他们不需要严格对齐发布，严格的对齐反而会影响其发布效率。无状态的分片业务只需要保证在发布过程中，多列业务满足一个整体的可用度即可。
#### **无状态多列业务对齐发布的group rolling**
![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2024/png/97471/1708325328636-e2678770-01fc-458f-a97f-a1c0befd4a54.png#clientId=ucaed3f0e-093a-4&from=paste&height=540&id=u69c70b78&originHeight=1080&originWidth=1920&originalType=binary&ratio=2&rotation=0&showTitle=false&size=330030&status=done&style=none&taskId=u1c89c0c9-20b4-481e-82dd-7b3c2aa3b30&title=&width=960)
对齐发布过程中不需要保持一个严格对齐的行概念，只需要整体可用度比例满足maxUnavailable限制
![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2024/png/97471/1708325756272-6c7cd864-18d2-421b-a317-df6f4daf52ec.png#clientId=ucaed3f0e-093a-4&from=paste&height=540&id=ucab3de97&originHeight=1080&originWidth=1920&originalType=binary&ratio=2&rotation=0&showTitle=false&size=550443&status=done&style=none&taskId=ucddf2ec3-62aa-4674-9d58-d43b2ae158d&title=&width=960)
算法简单来说就是maxUnavailable步长当作一个滑动窗，在所有列整理计算可用度的情况下向前滑动升级。
## 面向错误的设计：Recover策略
![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2024/png/97471/1708325928772-3779f815-f6be-4d3d-8d9e-672481447df5.png#clientId=ucaed3f0e-093a-4&from=paste&height=540&id=u83a00cf0&originHeight=1080&originWidth=1920&originalType=binary&ratio=2&rotation=0&showTitle=false&size=264239&status=done&style=none&taskId=u140ee337-86dc-4f85-b79b-01b674f5aab&title=&width=960)
对“错误节点”进行起新下老的替换，其中有两个关键点：

1. 新节点完全启动成功后才会删除老节点，从而保证在线服务的安全。
2. 对于“错误节点”定义，c2中把所有不能正常启动的节点都当成“错误节点”，这样节约了在巨大集群下必然会出现的人工替换坏节点的运维成本。
