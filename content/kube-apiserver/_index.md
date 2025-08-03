---
linkTitle: kube-apiserver
title: kube-apiserver
cascade:
  type: docs
weight: 1
---

API 服务器是 Kubernetes 控制平面的组件， 该组件负责公开了 Kubernetes API，负责处理接受请求的工作。 API 服务器是 Kubernetes
控制平面的前端。

Kubernetes API
服务器的主要实现是 [kube-apiserver](https://kubernetes.io/zh-cn/docs/concepts/architecture/#kube-apiserver)。
kube-apiserver 设计上考虑了水平扩缩，也就是说，它可通过部署多个实例来进行扩缩。
你可以运行 kube-apiserver 的多个实例，并在这些实例之间平衡流量。
