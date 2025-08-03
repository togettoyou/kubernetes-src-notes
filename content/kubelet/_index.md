---
linkTitle: kubelet
title: kubelet
cascade:
  type: docs
weight: 4
---

[kubelet](https://kubernetes.io/zh-cn/docs/concepts/architecture/#kubelet) 会在集群中每个节点（node）上运行。
它保证容器（containers）都运行在 Pod 中。

kubelet 接收一组通过各类机制提供给它的 PodSpec，确保这些 PodSpec 中描述的容器处于运行状态且健康。 kubelet 不会管理不是由
Kubernetes 创建的容器。
