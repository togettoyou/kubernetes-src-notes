---
title: Prometheus
---

## 在线Chart仓库方式

```shell
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
```

```shell
kubectl create ns monitoring
helm install prometheus prometheus-community/prometheus -f values.yaml --version 25.24.1 -n monitoring
```

## 本地离线Chart方式

```shell
wget https://github.com/prometheus-community/helm-charts/releases/download/prometheus-25.24.1/prometheus-25.24.1.tgz
```

```shell
kubectl create ns monitoring
helm install prometheus prometheus-25.24.1.tgz -f values.yaml -n monitoring
```

其中 `values.yaml` 配置文件：

```yaml
server:
  persistentVolume:
    enabled: true
    accessModes:
      - ReadWriteOnce
    size: 8Gi
    storageClass: local-storage
kube-state-metrics:
  enabled: true
prometheus-node-exporter:
  enabled: true
alertmanager:
  enabled: false
prometheus-pushgateway:
  enabled: false
```

配置参考：https://github.com/prometheus-community/helm-charts/tree/prometheus-25.24.1/charts/prometheus

## 附录

1. 国内无法拉取 `registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.12.0` 镜像，可以在节点执行：

   ```shell
   ctr -n k8s.io image pull registry.cn-hangzhou.aliyuncs.com/hubmirrorbytogettoyou/registry.k8s.io.kube-state-metrics.kube-state-metrics:v2.12.0 && ctr -n k8s.io image tag registry.cn-hangzhou.aliyuncs.com/hubmirrorbytogettoyou/registry.k8s.io.kube-state-metrics.kube-state-metrics:v2.12.0 registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.12.0
   ```

2. LocalPV 参考：

    ```yaml
    apiVersion: v1
    kind: PersistentVolume
    metadata:
      name: local-pv-prometheus-node1
    spec:
      capacity:
        storage: 8Gi
      volumeMode: Filesystem
      accessModes:
        - ReadWriteOnce
      persistentVolumeReclaimPolicy: Retain
      storageClassName: local-storage
      local:
        path: /mnt/localpv/prometheus
      nodeAffinity:
        required:
          nodeSelectorTerms:
            - matchExpressions:
                - key: kubernetes.io/hostname
                  operator: In
                  values:
                    - node1
    ```

## 微信公众号

更多内容请关注微信公众号：gopher云原生

<img src="https://github.com/user-attachments/assets/ea93572c-6c05-4751-bde7-35a58fe083f1" width="520px" />
