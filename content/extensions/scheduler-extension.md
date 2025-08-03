---
title: 调度扩展
---

参考：[scheduling-extensions](https://kubernetes.io/zh-cn/docs/concepts/extend-kubernetes/#scheduling-extensions)

## scheduler extender 调度扩展 （Webhook）

参考：[scheduler_extender](https://github.com/kubernetes/design-proposals-archive/blob/main/scheduling/scheduler_extender.md)

调度扩展实际是一种 Webhook ，kube-scheduler 通过 http 调用

只能作用于节点过滤（filter）、节点优先级排序（prioritize）、抢占/驱逐Pod（preempt）和节点绑定（bind）操作

代码示例：[scheduler-extension/webhook/simple](https://github.com/togettoyou/kubernetes-src-notes/tree/main/src/scheduler-extension/webhook/simple)

## scheduler framework 调度框架

参考：[scheduling-framework](https://kubernetes.io/zh-cn/docs/concepts/scheduling-eviction/scheduling-framework/)

代码示例：[scheduler-extension/framework/simple](https://github.com/togettoyou/kubernetes-src-notes/tree/main/src/scheduler-extension/framework/simple)

## WebAssembly (wasm) 自定义插件

参考：[kube-scheduler-wasm-extension](https://github.com/kubernetes-sigs/kube-scheduler-wasm-extension/blob/main/docs/tutorial.md)

代码示例：[scheduler-extension/wasm-extension/simple](https://github.com/togettoyou/kubernetes-src-notes/tree/main/src/scheduler-extension/wasm-extension/simple)

## 微信公众号

更多内容请关注微信公众号：gopher云原生

<img src="https://github.com/user-attachments/assets/ea93572c-6c05-4751-bde7-35a58fe083f1" width="520px" />
