---
linkTitle: kube-controller-manager
title: kube-controller-manager
cascade:
  type: docs
weight: 5
---

[kube-controller-manager](https://kubernetes.io/zh-cn/docs/concepts/architecture/#kube-controller-manager) 是控制平面的组件，
负责运行控制器进程。

从逻辑上讲， 每个控制器都是一个单独的进程， 但是为了降低复杂性，它们都被编译到同一个可执行文件，并在同一个进程中运行。

控制器有许多不同类型。以下是一些例子：

- Node 控制器：负责在节点出现故障时进行通知和响应
- Job 控制器：监测代表一次性任务的 Job 对象，然后创建 Pod 来运行这些任务直至完成
- EndpointSlice 控制器：填充 EndpointSlice 对象（以提供 Service 和 Pod 之间的链接）。
- ServiceAccount 控制器：为新的命名空间创建默认的 ServiceAccount。

以上并不是一个详尽的列表。
