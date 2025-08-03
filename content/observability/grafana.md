---
title: Grafana
---

## 在线Chart仓库方式

```shell
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
```

```shell
kubectl create ns monitoring
helm install grafana grafana/grafana -f values.yaml --version 8.3.6 -n monitoring
```

## 本地离线Chart方式

```shell
wget https://github.com/grafana/helm-charts/releases/download/grafana-8.3.6/grafana-8.3.6.tgz
```

```shell
kubectl create ns monitoring
helm install grafana grafana-8.3.6.tgz -f values.yaml -n monitoring
```

其中 `values.yaml` 配置文件：

```yaml
service:
  enabled: true
  type: NodePort
  nodePort: 30080
persistence:
  type: pvc
  enabled: true
  storageClassName: local-storage
  accessModes:
    - ReadWriteOnce
  size: 10Gi
grafana.ini:
  metrics:
    enabled: false
```

配置参考：https://github.com/grafana/helm-charts/tree/grafana-8.3.6/charts/grafana

## 附录

1. 查看 admin 密码：

    ```shell
    kubectl get secret --namespace monitoring grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
    ```

2. LocalPV 参考：

   ```yaml
   apiVersion: v1
   kind: PersistentVolume
   metadata:
     name: local-pv-grafana-node1
   spec:
     capacity:
       storage: 10Gi
     volumeMode: Filesystem
     accessModes:
       - ReadWriteOnce
     persistentVolumeReclaimPolicy: Retain
     storageClassName: local-storage
     local:
       path: /mnt/localpv/grafana
     nodeAffinity:
       required:
         nodeSelectorTerms:
           - matchExpressions:
               - key: kubernetes.io/hostname
                 operator: In
                 values:
                   - node1
   ```

## My All dashboards

20356 [Kubernetes 节点资源监控](https://grafana.com/grafana/dashboards/20356-kubernetes/)

## 微信公众号

更多内容请关注微信公众号：gopher云原生

<img src="https://github.com/user-attachments/assets/ea93572c-6c05-4751-bde7-35a58fe083f1" width="520px" />
