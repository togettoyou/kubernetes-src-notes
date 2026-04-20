---
title: 节点层 | DRA（动态资源分配）
weight: 6
---

Device Plugin 用一个整数描述节点上的设备数量，调度器只知道"这个节点有 3 块 GPU"，看不到它们的型号、显存大小、所在 NUMA 域。当一个 Pod 需要的是"显存 ≥ 40GB 的 A100"，Device Plugin 无能为力。**DRA（Dynamic Resource Allocation，动态资源分配）** 从根本上解决了这个问题：驱动通过 `ResourceSlice` 将设备的完整属性发布到 API Server，调度器可以用 CEL 表达式对属性做细粒度筛选，并原生支持多个 Pod 共享同一设备。DRA 于 Kubernetes 1.34 正式 GA，使用体验类似 PersistentVolumeClaim。

参考：[dynamic-resource-allocation](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/dynamic-resource-allocation/)

## Device Plugin 的局限

上一篇介绍的 Device Plugin 有三个根本局限：

- **只能按数量申请**：Pod 只能声明 `limits: vendor/gpu: 1`，无法表达"我要一块显存 ≥ 40GB 的 GPU"
- **无法共享设备**：一个设备 ID 在同一时刻只能分配给一个容器，即便设备本身支持多路复用
- **调度器感知不到属性**：`ListAndWatch` 上报的是设备 ID 列表，属性完全不可见，NUMA 亲和性等高级调度无从实现

DRA 针对性地解决了这三个问题：设备属性写进 `ResourceSlice`，调度器直接读取并用 CEL 做匹配；`ResourceClaim` 可以被多个 Pod 共享；整套调度决策在 API Server 层完成，不再绕过调度器在 kubelet 侧进行。

## 核心 API

DRA 围绕四个 API 资源构建：ResourceSlice、DeviceClass、ResourceClaim、ResourceClaimTemplate。

### ResourceSlice

ResourceSlice 是驱动发布到 API Server 的设备目录，集群级资源。每个 ResourceSlice 属于一个驱动（`spec.driver`）和一个资源池（`spec.pool`）。资源池通常对应一个节点，池名与节点名相同；驱动也可以将多个节点的设备聚合为一个共享池（如网络附加设备）。一个资源池可以拆分为多个 ResourceSlice 对象，`spec.pool.resourceSliceCount` 记录总片数，调度器收齐全部片才认为该池的设备信息完整。

每个设备（`spec.devices[]`）可以携带任意属性（`attributes`）和容量（`capacity`）：

- `attributes`：键值对，值类型支持 string、int、bool、version。调度器用 CEL 对这些属性做匹配，例如 `device.attributes["fake.dra.example.com"].model == "A100"`。键名不带域名前缀时默认归属驱动的域名。
- `capacity`：键值对，值为 `resource.Quantity`，表示该设备可提供的可量化资源，例如显存大小。

```yaml
spec:
  driver: fake.dra.example.com
  pool:
    name: node01
    generation: 1
    resourceSliceCount: 1
  nodeName: node01
  devices:
    - name: fake-0
      attributes:
        model: {string: "fake-v1"}
        index: {int: 0}
      capacity:
        memory: {value: 8Gi}
    - name: fake-1
      attributes:
        model: {string: "fake-v1"}
        index: {int: 1}
      capacity:
        memory: {value: 8Gi}
```

### DeviceClass

DeviceClass 是集群管理员定义的设备类型模板，集群级资源，类似 StorageClass。它的核心作用是预设一组 CEL 选择器，将"这类设备来自哪个驱动、满足什么基本条件"固化为一个可重用的名字，用户只需在 ResourceClaim 中引用名字，无需重复写驱动名或基础约束。

```yaml
apiVersion: resource.k8s.io/v1
kind: DeviceClass
metadata:
  name: fake-device
spec:
  selectors:
    - cel:
        expression: device.driver == "fake.dra.example.com"
```

调度器在匹配设备时，DeviceClass 的 selectors 与 ResourceClaim 中 `exactly.selectors` 取 AND 关系，两者均满足才算匹配成功。

### ResourceClaim

ResourceClaim 是用户声明资源需求的对象，命名空间级资源，类似 PersistentVolumeClaim。`spec.devices.requests` 中每个 request 描述一项设备需求：

- `name`：本 Pod 内的引用名，Pod 的 `resources.claims[].request` 与此对应。
- `exactly.deviceClassName`：引用 DeviceClass，指定设备类型。
- `exactly.count`：需要的设备数量，默认为 1。
- `exactly.selectors`：在 DeviceClass 条件之上追加 CEL 筛选，例如进一步限定显存大小或 NUMA 域。

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: my-fake-device
spec:
  devices:
    requests:
      - name: my-request
        exactly:
          deviceClassName: fake-device
          # 可选：追加 CEL 条件进一步筛选设备
          # selectors:
          #   - cel:
          #       expression: device.capacity["fake.dra.example.com"].memory.compareTo(quantity("16Gi")) >= 0
```

调度器在 PreBind 阶段同时写入 `status.allocation` 和 `status.reservedFor`，ResourceClaim 直接进入 `allocated,reserved` 状态，设备被独占。ResourceClaim 可以被多个 Pod 共享，`status.reservedFor` 列出所有当前持有者。

### ResourceClaimTemplate

ResourceClaimTemplate 用于"每个 Pod 独享一个 ResourceClaim"的场景，适合不希望设备在多 Pod 间共享的情况。`spec.spec` 与 ResourceClaim 的 `spec` 字段完全一致；`spec.metadata` 中的 labels/annotations 会被复制到生成的 ResourceClaim 上。

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: my-fake-device-template
  namespace: default
spec:
  metadata:
    labels:
      app: my-app
  spec:
    devices:
      requests:
        - name: my-request
          exactly:
            deviceClassName: fake-device
```

Pod 通过 `resourceClaimTemplateName`（而非 `resourceClaimName`）引用 Template：

```yaml
spec:
  resourceClaims:
    - name: my-device
      resourceClaimTemplateName: my-fake-device-template
  containers:
    - name: consumer
      resources:
        claims:
          - name: my-device
            request: my-request
```

Pod 创建时 Kubernetes 自动生成一个独立的 ResourceClaim（名称格式为 `<pod-name>-<claim-name>-<suffix>`），Pod 删除时该 ResourceClaim 随之删除。与直接引用 ResourceClaim 的区别仅在于生命周期：前者与 Pod 同生共死，后者由用户手动管理，可在 Pod 删除后继续保留或被其他 Pod 复用。

## 整体架构

DRA 涉及五个角色：

- **DRA 驱动**：以 DaemonSet 部署到每个节点，分为两个逻辑组件。**Controller** 负责向 API Server 发布本节点的 ResourceSlice；**kubelet Plugin** 通过 gRPC 接收 kubelet 的调用，执行设备的实际初始化和释放。两者通常运行在同一进程内。
- **kube-apiserver**：存储 ResourceSlice、ResourceClaim、DeviceClass 等 DRA 对象，是各组件唯一的通信枢纽。
- **kube-scheduler**：从 API Server 读取 ResourceSlice，对设备属性跑 CEL 表达式完成匹配，将 `status.allocation` 和 `status.reservedFor` 写回 ResourceClaim，并为 Pod 绑定节点。整个过程纯粹在控制面完成，无需和节点通信。
- **kubelet**：检测到 Pod 调度到本节点后，通过 DRA gRPC 接口依次调用驱动的 `NodePrepareResources`（设备初始化）和 `NodeUnprepareResources`（设备释放）。同时负责发现并注册 DRA 驱动（plugin manager 机制）。
- **containerd**：从 kubelet 接收 CDI 设备 ID，读取 `/etc/cdi/` 下的描述文件，将设备注入容器（设置环境变量、挂载路径、设备节点等）。

资源的完整生命周期跨越四个阶段：驱动启动、调度、节点准备、Pod 删除：

**驱动启动 + 调度**

```plantuml
@startuml
!theme plain
skinparam sequenceMessageAlign center
skinparam responseMessageBelowArrow true
skinparam defaultTextAlignment center
skinparam ParticipantPadding 20

participant "DRA 驱动" as drv #FFE0B2
participant "kube-apiserver" as api #E3F2FD
participant "kube-scheduler" as sch #E8F5E9
participant "kubelet" as k #BBDEFB
actor "用户" as user

== 驱动启动 ==
drv -> api : 创建 ResourceSlice（发布设备及属性）
drv -> k : 在 plugins_registry/ 创建注册 socket
k -> drv : GetInfo()
drv --> k : 类型=DRAPlugin，dra.sock 路径
k -> drv : NotifyRegistrationStatus(registered=true)

== 调度阶段 ==
user -> api : 创建 DeviceClass + ResourceClaim
user -> api : 创建 Pod（spec.resourceClaims 引用 ResourceClaim）
sch -> api : 读取 ResourceSlice，CEL 匹配设备（PreFilter/Filter）
sch -> api : 写入 status.allocation + reservedFor（PreBind）
note right of sch : ResourceClaim 状态变为 allocated,reserved
sch -> api : 绑定 Pod 到节点（写入 spec.nodeName）
@enduml
```

- **驱动启动**：Controller 先通过 API Server 发布 ResourceSlice，让调度器立即看到设备。Plugin 随后在 `plugins_registry/` 创建注册 socket，kubelet 的 plugin manager 检测到后主动调用 `GetInfo` 完成握手，握手成功后 kubelet 才会向该驱动发起 `NodePrepareResources`。
- **调度阶段**：调度器在 `PreFilter`/`Filter` 阶段筛选可行节点，`Reserve` 阶段确定设备分配方案，**`PreBind`** 阶段将 `status.allocation` 和 `status.reservedFor` 一次性写入 API Server，ResourceClaim 进入 `allocated,reserved` 状态，随后 `Bind` 阶段写入 `pod.spec.nodeName`。

**节点准备 + Pod 删除**

```plantuml
@startuml
!theme plain
skinparam sequenceMessageAlign center
skinparam responseMessageBelowArrow true
skinparam defaultTextAlignment center
skinparam ParticipantPadding 20

participant "DRA 驱动" as drv #FFE0B2
participant "kube-apiserver" as api #E3F2FD
participant "kubelet" as k #BBDEFB
participant "containerd" as cri #F3E5F5
actor "用户" as user

== 节点准备阶段 ==
k -> drv : NodePrepareResources(claims)
drv --> k : CDI 设备 ID 列表
k -> cri : RunPodSandbox / CreateContainer（携带 CDI ID）
cri -> cri : 按 CDI 规范将设备注入容器

== Pod 删除 ==
user -> api : 删除 Pod
k -> cri : StopContainer / RemovePodSandbox
k -> drv : NodeUnprepareResources(claims)
drv --> k : 成功（Claims map 中每个 claim 均有对应条目）
note right of k : resource claim controller 检测到 Pod 已删除\n清除 reservedFor 和 allocation，claim 回到未分配状态
@enduml
```

- **节点准备阶段**：kubelet 调用 `NodePrepareResources` 时传入已分配给该 Pod 的所有 ResourceClaim（含 namespace、name、uid），驱动执行设备初始化操作并返回 CDI 设备 ID。kubelet 将 CDI ID 传给 containerd，containerd 读取 `/etc/cdi/` 下对应的 JSON 描述文件，完成环境变量注入、设备节点挂载等操作。CDI 之于设备注入，类似 CNI 之于网络，是驱动与容器运行时之间的标准化协议。
- **Pod 删除**：kubelet 停止容器后调用 `NodeUnprepareResources` 释放设备。驱动必须在响应的 `Claims` map 中为每个传入的 claim 写入对应条目，否则 kubelet 认为释放未成功并持续重试。设备释放后，kube-controller-manager 的 resource claim controller 检测到 Pod 已删除，**一次性清除 `status.reservedFor` 和 `status.allocation`**，claim 回到完全未分配状态。这是 DRA 的"用时分配、用完即释放"设计：立即释放底层资源，避免下一个 Pod 被迫调度到同一节点。

## 实现 DRA 驱动

DRA 驱动由两部分组成：**Controller** 向 API Server 发布 ResourceSlice，**kubelet Plugin** 处理 kubelet 的 NodePrepareResources 调用。两者通常打包在同一个进程里，以 DaemonSet 部署到每个节点。

下面实现一个最简单的 DRA 驱动，在每个节点上发布 5 个虚拟设备（驱动名 `fake.dra.example.com`）。

### Controller：发布 ResourceSlice

Controller 在驱动启动时向 API Server 创建本节点的 ResourceSlice，将节点上可用的设备及属性告知调度器（`pkg/controller/controller.go`）：

```go
func (c *Controller) PublishResourceSlice(ctx context.Context) error {
    sliceName := fmt.Sprintf("%s-%s", DriverName, c.nodeName)

    // k8s 1.34（resource/v1）中 Device 的 Attributes 和 Capacity 直接在 Device 上，
    // v1beta1 时通过 Basic *BasicDevice 包裹，升级时需注意
    devices := make([]resourcev1.Device, DeviceCount)
    for i := 0; i < DeviceCount; i++ {
        devices[i] = resourcev1.Device{
            Name: fmt.Sprintf("fake-%d", i),
            // Attributes 是可供 ResourceClaim CEL 选择器查询的键值对
            // 调度器会将 ResourceClaim 的 selectors 与这些属性做匹配
            Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
                "model": {StringValue: strPtr("fake-v1")},
                "index": {IntValue: int64Ptr(int64(i))},
            },
            // Capacity 描述设备提供的可量化资源，供调度器计算资源余量
            Capacity: map[resourcev1.QualifiedName]resourcev1.DeviceCapacity{
                "memory": {Value: resource.MustParse("8Gi")},
            },
        }
    }

    slice := &resourcev1.ResourceSlice{
        ObjectMeta: metav1.ObjectMeta{Name: sliceName},
        Spec: resourcev1.ResourceSliceSpec{
            Driver: DriverName,
            // Pool 是资源池，通常以节点名命名；调度器通过 Pool 将设备与节点关联
            Pool: resourcev1.ResourcePool{
                Name:               c.nodeName,
                Generation:         0,
                ResourceSliceCount: 1,
            },
            // NodeName 在 resource/v1 中为 *string
            NodeName: &c.nodeName,
            Devices:  devices,
        },
    }

    existing, err := c.client.ResourceV1().ResourceSlices().Get(ctx, sliceName, metav1.GetOptions{})
    if errors.IsNotFound(err) {
        _, err = c.client.ResourceV1().ResourceSlices().Create(ctx, slice, metav1.CreateOptions{})
        ...
        return nil
    }
    // 设备列表变化时更新已有的 ResourceSlice，同时递增 Generation
    existing.Spec = slice.Spec
    existing.Spec.Pool.Generation++
    _, err = c.client.ResourceV1().ResourceSlices().Update(ctx, existing, metav1.UpdateOptions{})
    ...
}
```

`ResourceSlice` 是集群级资源，驱动的 ServiceAccount 需要 `resource.k8s.io/resourceslices` 的 `get/list/watch/create/update/delete` 权限（见 `deploy/driver.yaml` 中的 ClusterRole）。

### kubelet Plugin：注册握手

DRA 的注册协议与 Device Plugin 截然不同。Device Plugin 是驱动主动调用 kubelet 的注册接口；DRA 驱动是被动等待 kubelet 来询问。

做法是在 `/var/lib/kubelet/plugins_registry/` 目录下创建注册 socket，kubelet 的 plugin manager 持续监听该目录，发现新 socket 后主动调用驱动的 `GetInfo` 完成握手（`pkg/plugin/plugin.go`）：

```go
// Start 按顺序完成两件事：
//  1. 在 plugins/<driver>/dra.sock 启动 DRA gRPC 服务
//  2. 在 plugins_registry/<driver>.sock 启动注册 gRPC 服务，触发 kubelet 检测
//
// 顺序不能反：kubelet 检测到注册 socket 后立即调用 GetInfo，
// GetInfo 返回的 Endpoint 是 dra.sock，kubelet 随即连接该 socket，
// 所以 dra.sock 必须在注册 socket 创建之前就绪
func (p *DRAPlugin) Start() error {
    pluginDir := filepath.Join(PluginsDir, DriverName)
    os.MkdirAll(pluginDir, 0750)
    p.draSocketPath = filepath.Join(pluginDir, DRASocketName)

    p.startDRAServer()
    p.startRegistrationServer()
    return nil
}
```

`GetInfo` 向 kubelet 说明这是一个 DRAPlugin 以及 DRA socket 的位置；`NotifyRegistrationStatus` 是握手完成后 kubelet 的回调：

```go
// GetInfo 是 kubelet plugin manager 握手的第一步
// kubelet 通过返回值确认：这是一个 DRAPlugin，DRA socket 在哪里
func (p *DRAPlugin) GetInfo(_ context.Context, _ *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
    return &registerapi.PluginInfo{
        Type:              registerapi.DRAPlugin,
        Name:              DriverName,
        Endpoint:          p.draSocketPath,
        SupportedVersions: []string{"v1.DRAPlugin"},
    }, nil
}

// NotifyRegistrationStatus 是 kubelet plugin manager 握手的第二步
// kubelet 通知驱动注册是否成功
func (p *DRAPlugin) NotifyRegistrationStatus(_ context.Context, status *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
    if !status.PluginRegistered {
        klog.Errorf("Plugin registration failed: %s", status.Error)
    } else {
        klog.Info("Plugin successfully registered with kubelet")
    }
    return &registerapi.RegistrationStatusResponse{}, nil
}
```

### kubelet Plugin：准备与释放设备

注册完成后，kubelet 会在两个时机调用驱动：Pod 调度到本节点后调用 `NodePrepareResources`，Pod 删除后调用 `NodeUnprepareResources`。

`NodePrepareResources` 接收的每个 `Claim` 只含 `namespace/uid/name`，驱动需要自行从 ResourceClaim 的 `status.allocation` 字段读取调度器记录的具体分配设备（生产实现中通过 informer lister 完成）。准备好设备后，驱动返回 CDI 设备 ID，kubelet 将其传给 containerd。示例中跳过了读取 allocation 的步骤，直接返回一个固定的虚拟设备：

```go
func (p *DRAPlugin) NodePrepareResources(ctx context.Context, req *drapb.NodePrepareResourcesRequest) (*drapb.NodePrepareResourcesResponse, error) {
    resp := &drapb.NodePrepareResourcesResponse{
        Claims: make(map[string]*drapb.NodePrepareResourceResponse),
    }

    for _, claim := range req.Claims {
        klog.Infof("NodePrepareResources: claim=%s/%s uid=%s", claim.Namespace, claim.Name, claim.UID)

        // CDI 设备 ID 格式：<vendor>/<class>=<name>
        // 容器运行时通过此 ID 在 /etc/cdi/ 中查找设备描述文件，完成设备注入
        cdiID := fmt.Sprintf("%s/device=fake-0", DriverName)

        // 真实场景：在这里执行设备初始化操作（如创建软链接、分配显存分区等）
        klog.Infof("  prepared device: cdi=%s", cdiID)

        resp.Claims[claim.UID] = &drapb.NodePrepareResourceResponse{
            Devices: []*drapb.Device{
                {
                    // RequestNames 对应 ResourceClaim.spec.devices.requests[*].name
                    // 此示例固定填写，生产实现中应从 ResourceClaim.status.allocation 读取
                    RequestNames: []string{"my-request"},
                    PoolName:     p.nodeName,
                    DeviceName:   "fake-0",
                    CDIDeviceIDs: []string{cdiID},
                },
            },
        }
    }
    return resp, nil
}

func (p *DRAPlugin) NodeUnprepareResources(ctx context.Context, req *drapb.NodeUnprepareResourcesRequest) (*drapb.NodeUnprepareResourcesResponse, error) {
    resp := &drapb.NodeUnprepareResourcesResponse{
        Claims: make(map[string]*drapb.NodeUnprepareResourceResponse),
    }

    for _, claim := range req.Claims {
        klog.Infof("NodeUnprepareResources: claim=%s/%s uid=%s", claim.Namespace, claim.Name, claim.UID)
        // 真实场景：在这里释放设备资源（如删除软链接、归还显存分区等）
        // 每个 claim 必须在 Claims map 中有对应条目，否则 kubelet 认为释放未成功，会持续重试
        resp.Claims[claim.UID] = &drapb.NodeUnprepareResourceResponse{}
    }

    return resp, nil
}
```

### main.go

入口按顺序启动 Controller 和 kubelet Plugin，等待退出信号后清理 ResourceSlice。启动顺序有意义：先发布 ResourceSlice，调度器才能感知设备；再启动 kubelet Plugin，处理后续的 NodePrepareResources 调用（`main.go`）：

```go
func main() {
    klog.Info("Starting simple DRA driver")

    // 在集群内通过 ServiceAccount 获取访问 API Server 的凭证
    cfg, _ := rest.InClusterConfig()
    client, _ := kubernetes.NewForConfig(cfg)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 先发布 ResourceSlice，让调度器能感知到本节点的设备
    // ResourceSlice 是调度器看到设备的唯一途径，必须在 kubelet plugin 之前就绪
    ctrl := controller.NewController(client)
    ctrl.PublishResourceSlice(ctx)

    // 再启动 kubelet Plugin，处理 NodePrepareResources / NodeUnprepareResources
    p := plugin.NewDRAPlugin()
    p.Start()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh

    klog.Info("Shutting down simple DRA driver")
    p.Stop()
    // 删除 ResourceSlice，调度器随即感知本节点设备已下线
    ctrl.DeleteResourceSlice(context.Background())
}
```

完整代码见：[dra/simple](https://github.com/togettoyou/kubernetes-src-notes/tree/main/src/dra/simple)

## 部署与演示

### 部署驱动

驱动以 DaemonSet 部署，需挂载两个宿主机目录，并通过 ClusterRole 授权读写 ResourceSlice（`deploy/driver.yaml`）：

```yaml
containers:
  - name: simple-dra-driver
    image: togettoyou/simple-dra-driver:latest
    env:
      # NODE_NAME 注入当前节点名，Controller 用此值命名 ResourceSlice 和 Pool
      - name: NODE_NAME
        valueFrom:
          fieldRef:
            fieldPath: spec.nodeName
    volumeMounts:
      - name: plugins-dir
        mountPath: /var/lib/kubelet/plugins       # 驱动在此创建 dra.sock
      - name: registry-dir
        mountPath: /var/lib/kubelet/plugins_registry  # 驱动在此创建注册 socket
volumes:
  - name: plugins-dir
    hostPath:
      path: /var/lib/kubelet/plugins
  - name: registry-dir
    hostPath:
      path: /var/lib/kubelet/plugins_registry
```

DaemonSet 启动后，每个节点上的驱动实例各自发布本节点的 ResourceSlice，并向本节点的 kubelet 完成注册握手。查看驱动日志可以看到完整流程：

```bash
$ kubectl -n dra-system logs simple-dra-driver-x7bxr
I0420 13:55:52.218622       1 main.go:18] Starting simple DRA driver
I0420 13:55:52.277375       1 controller.go:102] Updated ResourceSlice fake.dra.example.com-node01
I0420 13:55:52.277867       1 plugin.go:94] DRA plugin gRPC server started at /var/lib/kubelet/plugins/fake.dra.example.com/dra.sock
I0420 13:55:52.278284       1 plugin.go:120] Registration server started at /var/lib/kubelet/plugins_registry/fake.dra.example.com.sock
I0420 13:55:53.149397       1 plugin.go:153] Plugin successfully registered with kubelet
```

### 创建 DeviceClass 和 ResourceClaim

管理员先创建 DeviceClass，限定只有本驱动的设备才能被申请（`deploy/deviceclass.yaml`）：

```yaml
apiVersion: resource.k8s.io/v1
kind: DeviceClass
metadata:
  name: fake-device
spec:
  selectors:
    - cel:
        expression: device.driver == "fake.dra.example.com"
```

用户创建 ResourceClaim 申请一个设备（`deploy/claim.yaml`）：

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: my-fake-device
  namespace: default
spec:
  devices:
    requests:
      - name: my-request
        exactly:
          deviceClassName: fake-device
```

### Pod 申请设备

Pod 通过 `spec.resourceClaims` 引用 ResourceClaim，容器通过 `resources.claims` 声明自己要使用哪个 claim。这是 DRA 与 Device Plugin 在 Pod 写法上最直观的区别：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: fake-device-consumer
  namespace: default
spec:
  # resourceClaims 声明此 Pod 要使用的 ResourceClaim
  resourceClaims:
    - name: my-device
      resourceClaimName: my-fake-device
  containers:
    - name: consumer
      image: busybox
      command: ["sh", "-c", "sleep 3600"]
      resources:
        # claims 声明容器使用 Pod 级别 resourceClaims 中的哪一项
        claims:
          - name: my-device
            request: my-request
```

Pod 创建后，调度器读取 ResourceSlice，找到满足 DeviceClass CEL 条件的设备并选定节点，在 PreBind 阶段将分配结果写入 ResourceClaim 的 `status.allocation` 和 `status.reservedFor`，随后将 `spec.nodeName` 写入 Pod 完成绑定。kubelet 检测到 Pod 调度到本节点，调用驱动的 `NodePrepareResources`，驱动返回 CDI 设备 ID，containerd 完成设备注入，容器启动。

ResourceClaim 被分配并绑定到 Pod 后，状态变为 `allocated,reserved`：

```bash
$ kubectl get resourceclaim my-fake-device
NAME             ALLOCATION-MODE   STATE                AGE
my-fake-device                     allocated,reserved   16s
```

驱动日志同步记录了 NodePrepareResources 和 NodeUnprepareResources 的调用（Pod 删除时触发释放）：

```bash
# Pod 调度到节点后，kubelet 调用 NodePrepareResources
I0420 13:56:32.563769       1 plugin.go:174] NodePrepareResources: claim=default/my-fake-device uid=a52cda2c-26a3-497c-9db5-779f48593f2c
I0420 13:56:32.563840       1 plugin.go:181]   prepared device: cdi=fake.dra.example.com/device=fake-0

# Pod 删除后，kubelet 调用 NodeUnprepareResources 释放设备
I0420 13:57:27.932697       1 plugin.go:208] NodeUnprepareResources: claim=default/my-fake-device uid=a52cda2c-26a3-497c-9db5-779f48593f2c
```

## 总结

DRA 把"设备是什么"和"设备怎么用"分离到两个层面。驱动通过 ResourceSlice 告诉控制面设备有哪些属性，调度器在控制面完成匹配并记录分配结果，kubelet 最后驱动设备初始化并将 CDI ID 交给容器运行时，各层职责清晰，互不感知彼此的细节。

实现一个 DRA 驱动需要做三件事：Controller 发布 ResourceSlice，kubelet Plugin 处理 NodePrepareResources（设备准备）和 NodeUnprepareResources（设备释放），注册 gRPC 服务让 kubelet plugin manager 能发现驱动。与 Device Plugin 的 ListAndWatch 上报模型相比，DRA 换来的是调度器对设备属性的完整可见性、原生的设备共享支持，以及与 PVC 一致的用户体验。

## 微信公众号

更多内容请关注微信公众号：gopher的Infra修行

<img src="https://github.com/user-attachments/assets/df5cfc9a-a33e-4471-83b5-e3e3999d0530" width="520px" />
