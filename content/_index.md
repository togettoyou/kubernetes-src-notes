---
title: Kubernetes 源码阅读
breadcrumbs: false
cascade:
  type: docs
---

版本：Kubernetes [1.27.2](https://git.k8s.io/kubernetes/CHANGELOG/CHANGELOG-1.27.md#v1272)

代码结构：

![Untitled](/index/Untitled.png)

其中：

- `api`: 存放 OpenAPI 的 spec 文件
- `build`: 包含构建 Kubernetes 的工具和脚本
- `cluster`: 包含用于构建、测试和部署Kubernetes集群的工具和脚本
- `cmd`: 包含 Kubernetes 所有组件入口的源代码，例如
  kube-apiserver、kube-scheduler、kube-controller-manager、kubelet、kube-proxy、kubectl 等
- `hack`: 包含用于构建和测试 Kubernetes 的脚本和工具
- `pkg`: 包含 Kubernetes 的核心公共库和工具代码
- `plugin`: 包含 Kubernetes 插件的源代码，例如认证插件、授权插件等
- `staging`: 存放部分核心库的暂存代码，这些暂存代码会定期发布到各自的顶级 [k8s.io](http://k8s.io/) 存储库
- `test`: 包含 Kubernetes 测试的源代码和测试工具
- `third_party`: 包含 Kubernetes 使用的第三方工具代码
- `vendor`: 包含 Kubernetes 使用的所有依赖库代码

Kubernetes 组件架构：

![Untitled](/index/Untitled%201.png)

## 微信公众号

更多内容请关注微信公众号：gopher云原生

<img src="https://github.com/user-attachments/assets/ea93572c-6c05-4751-bde7-35a58fe083f1" width="520px" />
