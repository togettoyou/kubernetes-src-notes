---
linkTitle: kube-proxy
title: kube-proxy
cascade:
  type: docs
weight: 6
---

[kube-proxy](https://kubernetes.io/zh-cn/docs/concepts/architecture/#kube-proxy) 是集群中每个节点（node）上所运行的网络代理，
实现 Kubernetes 服务（Service） 概念的一部分。

kube-proxy 维护节点上的一些网络规则， 这些网络规则会允许从集群内部或外部的网络会话与 Pod 进行网络通信。

如果操作系统提供了可用的数据包过滤层，则 kube-proxy 会通过它来实现网络规则。 否则，kube-proxy 仅做流量转发。

如果你使用网络插件为 Service 实现本身的数据包转发， 并提供与 kube-proxy 等效的行为，那么你不需要在集群中的节点上运行
kube-proxy。
