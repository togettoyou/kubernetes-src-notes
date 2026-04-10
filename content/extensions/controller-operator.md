---
title: API 层 | 控制器和 Operator 模式
---

在 Kubernetes 的所有扩展机制中，**CRD + Controller（Operator 模式）** 是使用最广泛的一种。它的核心思想是：通过 CustomResourceDefinition 向集群声明一种新的资源类型，再配合一个自定义控制器，持续监听这个资源的状态，将实际状态驱动向期望状态收敛。

这种模式本质上是对 Kubernetes 自身设计哲学的一次复用——kube-controller-manager 里的 Deployment Controller、StatefulSet Controller，走的也是完全相同的路子。

参考：[extend-kubernetes/operator](https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/operator/)

## 控制循环：Operator 模式的核心

理解 Operator 模式，先要理解 **控制循环（Control Loop）**。

传统的命令式系统告诉你"怎么做"——你执行一条命令，它立刻执行一次操作，执行完就结束了。Kubernetes 的设计是声明式的：你只需要告诉系统"期望状态是什么"，系统会自己想办法让实际状态与期望状态保持一致，并且在出现偏差时自动修复。

实现这个目标的机制就是控制循环：

```
永远循环：
  观察（Observe）— 获取资源的当前实际状态
  比对（Diff）    — 与期望状态比较，找出差异
  执行（Act）     — 执行操作，让实际状态向期望状态靠拢
```

这也是为什么控制器的核心逻辑被称为 **Reconcile（调谐）**——每一次调谐，都是在修复实际状态与期望状态之间的偏差。

值得注意的是，控制循环采用的是 **水平触发（Level-Triggered）** 而非边缘触发（Edge-Triggered）：控制器不关心"发生了什么事件"，只关心"当前状态是什么"。这意味着即使某次事件被遗漏，下一次触发时控制器依然能观察到正确的状态并做出正确的动作——这天然保证了容错性。

## CRD：向集群声明自定义资源

控制器要管理的资源，首先需要在 Kubernetes 中有一个对应的 API 类型。对于内置资源（Pod、Deployment 等），这些类型是 kube-apiserver 硬编码的。对于自定义资源，则需要通过 **CustomResourceDefinition（CRD）** 来声明。

一旦创建了 CRD，kube-apiserver 的 APIExtensionsServer 组件会自动为这个新资源类型提供标准的 CRUD 接口，数据同样持久化到 etcd 中，`kubectl get/apply/delete` 全部开箱即用。

CRD 对应的 Go 类型定义遵循固定的结构，以 Kubebuilder 生成的 `MyPod` 为例：

```go
// api/v1/mypod_types.go

// MyPodSpec 定义 MyPod 的期望状态（用户声明的目标）
type MyPodSpec struct {
    Foo string `json:"foo,omitempty"`
}

// MyPodStatus 定义 MyPod 的实际状态（控制器观察并写回的结果）
type MyPodStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MyPod 是 mypods API 的核心类型
type MyPod struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   MyPodSpec   `json:"spec,omitempty"`
    Status MyPodStatus `json:"status,omitempty"`
}
```

这个结构有几个固定约定：

- **Spec**：期望状态，由用户填写，控制器只读
- **Status**：实际状态，由控制器写回，记录观察到的真实情况
- **TypeMeta + ObjectMeta**：每个 Kubernetes 资源都必须内嵌这两个字段，分别存储 `apiVersion/kind` 和 `name/namespace/labels` 等元信息

注释 `// +kubebuilder:object:root=true` 和 `// +kubebuilder:subresource:status` 是 Kubebuilder 的 **Marker 注解**，后面会详细介绍它们的作用。

## 方案一：直接使用 client-go

client-go 是 Kubernetes 官方的 Go 客户端库，kube-controller-manager 自身就是用它来实现所有内置控制器的。直接使用 client-go 灵活度最高，也最接近 Kubernetes 控制器的底层实现。

### Informer：带本地缓存的监听机制

直接用 `list/watch` 轮询 kube-apiserver 代价太高。client-go 提供了 **Informer** 机制来解决这个问题：

Informer 在本地内存中维护一份资源的完整缓存（Store），启动时先通过 List 接口全量同步，之后通过 Watch 接口接收增量事件，保持缓存与 etcd 的实时一致。控制器所有的读操作都命中本地缓存，不会给 kube-apiserver 带来压力。

**SharedInformerFactory** 是批量管理 Informer 的工厂，它确保同一种资源的 Informer 只创建一个实例，在多个控制器之间共享：

```go
// main.go

// 连接集群
clientSet, err := kubernetes.NewForConfig(cfg)

// 创建 SharedInformerFactory，resync 周期为 0（不强制全量 re-list）
sharedInformerFactory := informers.NewSharedInformerFactory(clientSet, 0)

// 获取 Pod 的 Informer
podInformer := sharedInformerFactory.Core().V1().Pods()
```

### WorkQueue：可靠的事件分发

Informer 捕获到资源变更事件后，不应该直接在事件回调里执行业务逻辑——那样如果处理失败就没有重试机会，并发也难以控制。

标准做法是将变更对象的 key（格式为 `namespace/name`）放入一个 **工作队列（WorkQueue）**，由独立的 worker goroutine 从队列中取出并处理：

```go
// main.go

// 创建限速工作队列，内置重试退避，避免频繁失败后的无限重试
queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

// 注册事件回调，将 key 推入队列
podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(obj)
        if err == nil {
            queue.Add(key)
        }
    },
    UpdateFunc: func(oldObj, newObj interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(newObj)
        if err == nil {
            queue.Add(key)
        }
    },
    DeleteFunc: func(obj interface{}) {
        key, err := cache.MetaNamespaceKeyFunc(obj)
        if err == nil {
            queue.Add(key)
        }
    },
})
```

WorkQueue 有一个关键特性：**去重**，如果同一个 key 在处理完成之前被多次入队，队列中只会保留一份。这与水平触发的设计理念一脉相承——不管触发了多少次事件，最终只需要处理一次当前状态即可。

### Controller：组装控制循环

有了 Informer 缓存和 WorkQueue，控制器的主体逻辑就是标准的"从队列取 key → 从缓存读状态 → 执行调谐"循环：

```go
// controller.go

type Controller struct {
    workqueue workqueue.RateLimitingInterface // 工作队列
    lister    v1.PodLister                   // 从 Informer 本地缓存读取 Pod
    informer  cache.Controller               // 用于等待缓存同步完成
}

func (c *Controller) Run(ctx context.Context, workers int) {
    defer c.workqueue.ShutDown()

    // 等待 Informer 完成首次全量同步，确保本地缓存已就绪
    if ok := cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced); !ok {
        return
    }

    // 启动指定数量的 worker goroutine 并发处理队列
    for i := 0; i < workers; i++ {
        go wait.UntilWithContext(ctx, c.runWorker, time.Second)
    }
    <-ctx.Done()
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
    key, shutdown := c.workqueue.Get()
    if shutdown {
        return false
    }
    defer c.workqueue.Done(key) // 标记本次处理完成

    err := c.syncHandler(key.(string))
    c.handleErr(ctx, err, key)
    return true
}

func (c *Controller) syncHandler(key string) error {
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return nil
    }

    // 从 Informer 本地缓存读取最新状态（不发起网络请求）
    pod, err := c.lister.Pods(namespace).Get(name)
    if err != nil {
        if errors.IsNotFound(err) {
            // 对象已删除，key 已无效，忽略即可
            return nil
        }
        return err
    }

    // 在这里实现调谐逻辑：比对期望与实际状态，执行相应操作
    fmt.Printf("Reconciling Pod %s/%s\n", pod.GetNamespace(), pod.GetName())
    return nil
}

func (c *Controller) handleErr(ctx context.Context, err error, key interface{}) {
    if err == nil {
        c.workqueue.Forget(key) // 处理成功，清除重试计数
        return
    }
    if c.workqueue.NumRequeues(key) < 3 {
        // 失败次数未达上限，重新入队（限速退避）
        c.workqueue.AddRateLimited(key)
        return
    }
    // 超过重试上限，放弃处理
    c.workqueue.Forget(key)
}
```

可以看到，直接使用 client-go 需要自己组装 Informer、WorkQueue、Controller 这三个核心组件，并手动处理缓存同步、重试退避、优雅关闭等细节。这正是 controller-runtime 出现的动机。

代码示例：[controller-operator/client-go/simple](https://github.com/togettoyou/kubernetes-src-notes/tree/main/src/controller-operator/client-go/simple)

## 方案二：使用 controller-runtime

[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) 是社区当前的主流选择，Kubebuilder 和 Operator SDK 生成的项目都基于它。它在 client-go 之上提供了更高层次的抽象，将开发者从繁琐的基础设施搭建中解放出来，专注于业务逻辑。

controller-runtime 的核心概念只有三个：**Manager**、**Controller**、**Reconciler**，分别对应生命周期管理、资源监听和调谐逻辑。

### Manager：统一的生命周期管理

Manager 是整个 Operator 的入口，负责管理所有 Controller 的生命周期，并内置了以下能力：

- **Leader Election**：多副本部署时，确保同一时刻只有一个 Pod 在主动调谐，其余处于热备状态
- **健康检查端点**：自动暴露 `/healthz` 和 `/readyz` 接口，供 Kubernetes 探针使用
- **Metrics 接口**：自动暴露 Prometheus 格式的指标
- **共享 Cache**：内部维护 Informer 缓存，所有 Controller 共用，避免重复 Watch

```go
// main.go

// 创建 Manager，传入集群配置和选项
mgr, err := ctrl.NewManager(cfg, ctrl.Options{})
if err != nil {
    panic(err)
}

// 向 Manager 注册控制器，声明关注 Pod 资源
err = ctrl.NewControllerManagedBy(mgr).
    Named("pod-controller").
    For(&corev1.Pod{}).         // 监听 Pod 的增删改事件
    Complete(&podReconciler{client: mgr.GetClient()})

// 启动 Manager（阻塞，直到收到终止信号）
mgr.Start(context.Background())
```

### Reconciler：只需关注"当前状态是什么"

开发者唯一需要实现的接口是 `Reconciler`，它只有一个方法：

```go
// controller.go

type podReconciler struct {
    client client.Client // controller-runtime 提供的统一客户端（读走缓存，写走 API Server）
}

// 确保 podReconciler 实现了 reconcile.Reconciler 接口（编译期检查）
var _ reconcile.Reconciler = &podReconciler{}

func (r *podReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
    pod := &corev1.Pod{}

    // 从缓存读取目标 Pod 的当前状态
    err := r.client.Get(ctx, request.NamespacedName, pod)
    if err != nil {
        if errors.IsNotFound(err) {
            // Pod 已删除，无需处理
            return reconcile.Result{}, nil
        }
        return reconcile.Result{}, err
    }

    // 在这里实现调谐逻辑
    fmt.Printf("Reconciling Pod %s/%s\n", pod.GetNamespace(), pod.GetName())

    // reconcile.Result{} 表示调谐成功，无需重新入队
    // reconcile.Result{RequeueAfter: time.Minute} 表示 1 分钟后重新调谐
    return reconcile.Result{}, nil
}
```

对比 client-go 方案，差异一目了然：Informer 的创建、WorkQueue 的管理、缓存同步等待、重试逻辑——这些全部由 controller-runtime 内部处理，开发者只需要关注 `Reconcile` 方法里"拿到资源对象后做什么"。

代码示例：[controller-operator/controller-runtime/simple](https://github.com/togettoyou/kubernetes-src-notes/tree/main/src/controller-operator/controller-runtime/simple)

## 方案三：使用 Kubebuilder / Operator SDK

[Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) 和 [Operator SDK](https://github.com/operator-framework/operator-sdk) 是基于 controller-runtime 的代码脚手架工具，它们的核心价值在于 **代码生成**：通过分析 Go 类型上的 Marker 注解，自动生成 CRD YAML、RBAC 规则、Webhook 配置等大量重复性文件。

### 初始化项目

```bash
# 初始化项目，指定 domain（CRD group 的后缀）和 module 名
kubebuilder init --project-name simple --domain controller.io --repo simple

# 创建一个新的 API 类型和对应的 Controller 骨架
kubebuilder create api --group simple --version v1 --kind MyPod
```

执行上述命令后，Kubebuilder 会自动生成以下关键文件：

```
simple/
├── api/v1/
│   ├── mypod_types.go          # CRD 类型定义（开发者填写 Spec/Status 字段）
│   └── zz_generated.deepcopy.go  # 自动生成的 DeepCopy 方法，无需手动维护
├── internal/controller/
│   └── mypod_controller.go     # Controller 骨架（开发者在此实现 Reconcile 逻辑）
├── config/
│   ├── crd/                    # 由 make generate/manifests 自动生成的 CRD YAML
│   └── rbac/                   # 由 Marker 注解自动生成的 RBAC 规则
└── cmd/main.go                 # Manager 入口，通常无需修改
```

### Marker 注解：用注释驱动代码生成

Kubebuilder 最有特色的设计是 **Marker 注解**，即写在 Go 代码注释里的特殊指令，`controller-gen` 工具会读取这些指令并生成对应的文件。

**类型级 Marker**（写在结构体前）：

```go
// +kubebuilder:object:root=true       — 声明此类型是 CRD 根对象，生成 DeepCopyObject 接口实现
// +kubebuilder:subresource:status     — 为此资源启用 status 子资源（status 更新走独立接口，避免冲突）
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//                                     — 定制 kubectl get 输出的列
type MyPod struct { ... }
```

**Controller 级 Marker**（写在 Reconcile 方法前）：

```go
// +kubebuilder:rbac:groups=simple.controller.io,resources=mypods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=simple.controller.io,resources=mypods/status,verbs=get;update;patch
func (r *MyPodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    ...
}
```

执行 `make manifests` 后，这些 Marker 会被 `controller-gen` 解析，自动生成 CRD YAML 和 RBAC ClusterRole，开发者不再需要手写这些枯燥的配置。

### main.go：自动注册 CRD Scheme

使用 Kubebuilder 时，`main.go` 中有一个重要步骤——将自定义资源类型注册到 Scheme：

```go
// cmd/main.go

var scheme = runtime.NewScheme()

func init() {
    // 注册 Kubernetes 内置类型（Pod、Deployment 等）
    utilruntime.Must(clientgoscheme.AddToScheme(scheme))
    // 注册自定义类型（MyPod 等）
    utilruntime.Must(simplev1.AddToScheme(scheme))
}

func main() {
    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
        Scheme:                 scheme,
        MetricsBindAddress:     ":8080",
        HealthProbeBindAddress: ":8081",
        LeaderElection:         enableLeaderElection,
        LeaderElectionID:       "1666807f.controller.io",
    })
    // ...
    mgr.Start(ctrl.SetupSignalHandler())
}
```

Scheme 是 controller-runtime 的类型注册表，它负责在 Go 类型和 Kubernetes API 的 `apiVersion/kind` 字符串之间做双向映射。只有注册到 Scheme 的类型，client 才能正确地序列化和反序列化。

代码示例：[controller-operator/kubebuilder/simple](https://github.com/togettoyou/kubernetes-src-notes/tree/main/src/controller-operator/kubebuilder/simple)

## 三种方案横向对比

| 维度 | client-go | controller-runtime | Kubebuilder / Operator SDK |
|------|-----------|-------------------|---------------------------|
| **抽象层次** | 底层，手动组装 Informer + WorkQueue | 中层，专注实现 Reconcile | 高层，脚手架生成大量样板代码 |
| **代码量** | 最多 | 中等 | 最少 |
| **灵活性** | 最高 | 高 | 中（基于 controller-runtime） |
| **CRD 支持** | 需手写 YAML | 需手写 YAML | 自动从 Go 类型生成 |
| **RBAC 生成** | 需手写 | 需手写 | Marker 注解自动生成 |
| **典型使用者** | kube-controller-manager 等核心组件 | 社区主流 Operator | 快速搭建新 Operator 项目 |
| **代表项目** | kube-scheduler、kube-controller-manager | cert-manager、Argo CD | 大量企业内部 Operator |

理解了 controller-runtime 的核心概念，Kubebuilder 和 Operator SDK 就很容易上手——它们不引入新的运行时概念，只是让项目初始化和配置生成更省力。

## 微信公众号

更多内容请关注微信公众号：gopher的Infra修行

<img src="https://github.com/user-attachments/assets/df5cfc9a-a33e-4471-83b5-e3e3999d0530" width="520px" />
