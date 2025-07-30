---
title: 使用 kubeadm 创建集群
outline: deep
---

# 使用 kubeadm 创建集群

所有节点执行：

```shell
curl -sSL https://raw.githubusercontent.com/togettoyou/kubernetes-src-notes/main/src/appendix/install/start.sh | bash
```

脚本执行成功后，使用 kubeadm 创建集群：

```shell
kubeadm init --pod-network-cidr=10.244.0.0/16 --kubernetes-version=v1.27.2 --image-repository registry.aliyuncs.com/google_containers --v=5
```

后续参考：https://kubernetes.io/zh-cn/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/

## 附录

1. 安装 Pod 网络附加组件：

    - calico

        ```shell
        kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/calico.yaml
        ```

    - flannel

        ```shell
        kubectl apply -f https://github.com/flannel-io/flannel/releases/download/v0.24.0/kube-flannel.yml
        ```

2. 安装 Helm ：

   ```shell
   curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
   ```

3. 安装 Kubernetes Metrics Server ：

   ```shell
   kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.6.4/components.yaml
   ```

   若无法拉取镜像：

   ```shell
   ctr -n k8s.io image pull registry.cn-hangzhou.aliyuncs.com/hubmirrorbytogettoyou/registry.k8s.io.metrics-server.metrics-server:v0.6.4 && ctr -n k8s.io image tag registry.cn-hangzhou.aliyuncs.com/hubmirrorbytogettoyou/registry.k8s.io.metrics-server.metrics-server:v0.6.4 registry.k8s.io/metrics-server/metrics-server:v0.6.4
   ```

   若 metrics-server 服务一直无法 ready ，需要编辑 Deployment 增加 `--kubelet-insecure-tls` 运行参数：

   ```yaml
         containers:
           - args:
               - --kubelet-insecure-tls
   ```

4. containerd 设置国内代理（以腾讯云代理为例）：

   ```toml
   [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
           [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
              endpoint = ["https://mirror.ccs.tencentyun.com"]
   ```

5. 启用 kubectl 自动补全功能

   ```shell
   # yum install bash-completion
   # apt-get install bash-completion
   echo 'source /usr/share/bash-completion/bash_completion' >>~/.bashrc
   echo 'source <(kubectl completion bash)' >>~/.bashrc
   echo 'alias k=kubectl' >>~/.bashrc
   echo 'complete -o default -F __start_kubectl k' >>~/.bashrc
   source ~/.bashrc
   ```

## 微信公众号

更多内容请关注微信公众号：gopher云原生

<img src="https://github.com/user-attachments/assets/ea93572c-6c05-4751-bde7-35a58fe083f1" width="520px" />
