---
title: 扩展 Kubernetes API
---

参考：https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/api-extension/

扩展 Kubernetes API 实际就是创建定制资源（Custom Resources，CR）

## kube-apiserver 的三个服务

- AggregatorServer：API 聚合服务。用于实现 Kubernetes API 聚合层的功能，当 AggregatorServer 接收到请求之后，如果发现对应的是一个
  APIService 的请求，则会直接转发到对应的服务上（自行编写和部署的扩展 API 服务器，称为 extension-apiserver ），否则则委托给
  KubeAPIServer 进行处理

- KubeAPIServer：API 核心服务。实现所有 Kubernetes 内置资源的 REST API 接口（诸如 Pod 和 Service
  等资源的接口），如果请求未能找到对应的处理，则委托给 APIExtensionsServer 进行处理

- APIExtensionsServer：API 扩展服务。处理 CustomResourceDefinitions（CRD）和 Custom Resource（CR）的 REST
  请求（自定义资源的通用处理接口），如果请求仍不能被处理则委托给 404 Handler 处理

## 方案一：定制资源定义（CustomResourceDefinitions，CRD）+ 定制控制器（Custom Controller）= Operator 模式

参考：https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/api-extension/custom-resources/

利用 kube-apiserver 的最后一个服务 APIExtensionsServer ，kube-apiserver 对 CRD 声明的 CR 有通用的 CRUD Handle 逻辑
，和内置资源一样，会存储到 etcd 中

创建 CRD 无需编码，但往往需要结合自定义 Controller 一起使用，即 Operator 模式

### 原理：APIExtensionsServer 的 API Discovery

APIExtensionsServer 用于处理 CustomResourceDefinitions（CRD）和 Custom Resource（CR）的 REST
请求（自定义资源的通用处理接口）

其中的 [DiscoveryController](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_discovery_controller.go#L45)
会监听 CRD
资源的变化，动态注册 [/apis/\<group>](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_discovery_controller.go#L246)
和 [/apis/\<group>/\<version>](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_discovery_controller.go#L259-L261)
路由

- `/apis/<group>`
  ：返回的是一个 [APIGroup](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L1057-L1076)
  对象

- `/apis/<group>/<version>`
  ：返回的是一个 [APIResourceList](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L1148-L1154)
  对象

并且还会将 GroupVersion 和 Resources
信息通过 [AddGroupVersion](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_discovery_controller.go#L267-L271)
方法添加到全局的 [AggregatedDiscoveryGroupManager](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apiserver/pkg/server/config.go#L278)
内存对象中，以此聚合到 `/apis`
路由返回的 [APIGroupList](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L1047-L1051)
或 [APIGroupDiscoveryList](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/api/apidiscovery/v2beta1/types.go#L33-L41)
对象中

> APIGroupDiscoveryList 是 [1.26 新增的 API](https://github.com/kubernetes/enhancements/issues/3352) （默认关闭）, 在 1.27
> 默认开启
>>
> APIGroupDiscoveryList = APIGroupList + APIResourceList
>>
> 作用：减少请求次数，直接请求 `/apis` 端点一次性获取到 APIGroupDiscoveryList 对象
>>
> v1.26 或之前版本需要请求 `/apis` 获取 APIGroupList 对象，随后再继续请求 `/apis/<group>` 和 `/apis/<group>/<version>`
> 端点获取到所有的 APIResourceList 对象
>>
> 可以通过判断 header 是否有 `Accept: application/json;as=APIGroupDiscoveryList;v=v2beta1;g=apidiscovery.k8s.io` 来区分是请求
> APIGroupDiscoveryList 还是 APIGroupList 对象

### 流程演示

1. 创建 `crd.yaml` :

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: crontabs.simple.extension.io
spec:
  # 可以是 Namespaced 或 Cluster
  scope: Namespaced
  names:
    # 名称的复数形式，用于 URL：/apis/<组>/<版本>/<名称的复数形式>
    plural: crontabs
    # 名称的单数形式，作为命令行使用时和显示时的别名
    singular: crontab
    # kind 通常是单数形式的驼峰命名（CamelCased）形式。你的资源清单会使用这一形式。
    kind: CronTab
    # shortNames 允许你在命令行使用较短的字符串来匹配资源
    shortNames:
      - ct
  # 组名称，用于 REST API: /apis/<组>/<版本>
  group: simple.extension.io
  # 列举此 CustomResourceDefinition 所支持的版本
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                cronSpec:
                  type: string
                image:
                  type: string
                replicas:
                  type: integer
---
#apiVersion: simple.extension.io/v1
#kind: CronTab
#metadata:
#  name: test-cron
#spec:
#  cronSpec: "* * * * */5"
#  image: "hello-world"
#  replicas: 1
```

创建资源：

```shell
[root@node1 ~]# k apply -f crd.yaml 
customresourcedefinition.apiextensions.k8s.io/crontabs.simple.extension.io created
[root@node1 ~]# 
```

2. 查看 CR ，同时调整日志级别显示所请求的资源

```shell
[root@node1 ~]# k get CronTab -v 6
I0207 15:20:44.379006   13196 loader.go:373] Config loaded from file:  /root/.kube/config
I0207 15:20:44.387614   13196 discovery.go:214] Invalidating discovery information
I0207 15:20:44.396388   13196 round_trippers.go:553] GET https://10.0.8.17:6443/api?timeout=32s 200 OK in 8 milliseconds
I0207 15:20:44.399674   13196 round_trippers.go:553] GET https://10.0.8.17:6443/apis?timeout=32s 200 OK in 1 milliseconds
I0207 15:20:44.407886   13196 round_trippers.go:553] GET https://10.0.8.17:6443/apis/simple.extension.io/v1/namespaces/default/crontabs?limit=500 200 OK in 1 milliseconds
No resources found in default namespace.
[root@node1 ~]# 
```

可以看到，首先会请求 `/api` 路由（核心 API ，没有 G 组的概念，只有 V 版本和 K 资源），返回的同样是 `APIGroupList` 或
`APIGroupDiscoveryList` 对象（这里是 `APIGroupDiscoveryList` ），对于 K 为 `CronTab` 的 CR 资源，肯定无法在此发现

所以会接着继续请求 `/apis` 路由，从这里就可以找到 K 为 `CronTab` 所对应的 G 和 V
了，即最终请求 `/apis/simple.extension.io/v1/namespaces/default/crontabs`
路由（ [CR 通用的 CRUD Handle 逻辑](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L225-L360)）

如果想显示详细的请求内容，可以调整日志级别为 `-v 9`

## 方案二：API 聚合（API Aggregation，AA）

参考：https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/

利用 kube-apiserver 的第一个服务 AggregatorServer ，kube-apiserver 发现收到自定义 APIService 的请求时，会转发到对应的自行编写和部署的扩展
API 服务器（Extension API Server），相比方案一，有更强扩展性：

- 可以采用除了 etcd 之外，其它不同的存储

- 可以扩展长时间运行的子资源/端点，例如 websocket

- 可以与任何其它外部系统集成

但也有缺点，需要自行实现 REST API ：

- API Discovery

- OpenAPI v2/v3 Specification（非必须）

- CR 的 CRUD Handle

因此，若无特殊需求，推荐直接使用方案一

### 原理：AggregatorServer 的 API Discovery

当 AggregatorServer 接收到请求之后，如果发现对应的是一个 APIService 的请求，则会直接转发到对应的扩展 API 服务器上

和 [APIExtensionsServer 的 API Discovery](#原理-apiextensionsserver-的-api-discovery)
类似，AggregatorServer
也有一个 [DiscoveryAggregationController](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/kube-aggregator/pkg/apiserver/handler_discovery.go#L50-L64)
会监听 APIService
资源的变化，[调用 AA 服务的 /apis 接口](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/kube-aggregator/pkg/apiserver/handler_discovery.go#L192-L207)
，然后将 AA 服务的 APIGroupDiscoveryList
对象 [添加到 kube-apiserver 全局的 AggregatedDiscoveryGroupManager](https://github.com/kubernetes/kubernetes/blob/v1.27.2/staging/src/k8s.io/kube-aggregator/pkg/apiserver/handler_discovery.go#L384)
内存对象中，以此聚合到 kube-apiserver 的 `/apis` 端点

因此，对于 AA 服务，我们至少需要自行实现以下接口用于 API Discovery ：

- `/apis` ：用于给 AggregatorServer 获取 AA 的 APIGroupDiscoveryList 或 APIGroupList 对象

- `/apis/<group>` ：CRD 会动态注册，但 AA 需要自行实现，返回 APIGroup 对象

- `/apis/<group>/<version>` ：CRD 会动态注册，但 AA 需要自行实现，返回 APIResourceList 对象

> 其中 `/apis` 返回的 APIGroupList 对象，以及 `/apis/<group>` 和 `/apis/<group>/<version>` 路由是为了兼容 1.27 之前版本

另外，对于 CRD 声明的 CR 会有通用的 CRUD Handle ，但对于 AA 所创建的 CR 是需要自行实现逻辑的

### 开发方案

- [apiserver-runtime](https://github.com/kubernetes-sigs/apiserver-runtime) （不推荐）

  apiserver-runtime 是专门开发 AA 服务的 SDK 框架，但是 apiserver-runtime
  是[不稳定](https://github.com/kubernetes-sigs/apiserver-builder-alpha/issues/541)的，目前对于 AA
  服务的开发，社区并没有一个较流行的库支持（类似 [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
  那样）

- [k8s.io/apiserver](https://github.com/kubernetes/apiserver)

  apiserver-runtime 实际是基于 kube-apiserver 组件的 k8s.io/apiserver 库提供扩展。建议直接学习使用该库，可以保证最大的灵活定制，不过难度也相应较大

- [Kubernetes API](https://kubernetes.io/zh-cn/docs/reference/using-api/api-concepts/)

  理论上，对于简单的需求，对照着 kube-apiserver
  的 [API 规范](https://kubernetes.io/zh-cn/docs/concepts/overview/kubernetes-api/)，直接手写也是可以的，重点是需要实现
  API Discovery ，使 kube-apiserver 可以知道 AA 服务实现了什么 CR ，从而将请求转发过来

### Kubernetes API Server 必须通过 HTTPS 访问扩展 API 服务器（Extension API Server）

参考：[准入 Webhook](/extensions/admission-webhook)

其中 `tls.crt` 和 `tls.key` 将用于 Extension API Server 启动 HTTPS 服务， `$(cat ca.crt | base64 | tr -d '\n')` 需作为创建
APIService 资源时的 caBundle 字段值，例如：

```yaml
apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata:
  name: <注释对象名称>
spec:
  group: <扩展 Apiserver 的 API 组名>
  version: <扩展 Apiserver 的 API 版本>
  groupPriorityMinimum: <APIService 对应组的优先级, 参考 API 文档>
  versionPriority: <版本在组中的优先排序, 参考 API 文档>
  service:
    namespace: <拓展 Apiserver 服务的名字空间>
    name: <拓展 Apiserver 服务的名称>
  caBundle: <PEM 编码的 CA 证书，用于对 Webhook 服务器的证书签名>
```

但 `caBundle` 不是必须的，可以通过 `insecureSkipTLSVerify: true` 禁用 TLS 证书验证（不建议）

### 代码示例

[api-extension/AA/simple](https://github.com/togettoyou/kubernetes-src-notes/tree/main/src/api-extension/AA/simple)

## 微信公众号

更多内容请关注微信公众号：gopher云原生

<img src="https://github.com/user-attachments/assets/ea93572c-6c05-4751-bde7-35a58fe083f1" width="520px" />
