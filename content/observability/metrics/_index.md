---
linkTitle: metrics
title: 指标
cascade:
  type: docs
weight: 8
---

一个 [metric](https://opentelemetry.io/zh/docs/concepts/signals/metrics/) 是在运行时捕获的服务的测量值。捕获测量值的时刻称为
metric 事件，它不仅包括测量值本身，还包括捕获它的时间和相关的元数据。

应用和请求的 metrics 是可用性和性能的重要指标。自定义 metric
可以在‘可用性因素是如何影响到用户体验和业务’方面提供见解。收集的数据可以用于异常警告或触发调度决策，以在高要求时自动扩展部署。
