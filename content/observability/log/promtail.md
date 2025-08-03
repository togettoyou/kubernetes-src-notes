---
title: Promtail
---

## 在线Chart仓库方式

```shell
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
```

```shell
kubectl create ns monitoring
helm install promtail grafana/promtail -f values.yaml --version 6.16.4 -n monitoring
```

## 本地离线Chart方式

```shell
wget https://github.com/grafana/helm-charts/releases/download/promtail-6.16.4/promtail-6.16.4.tgz
```

```shell
kubectl create ns monitoring
helm install promtail promtail-6.16.4.tgz -f values.yaml -n monitoring
```

## 配置参考

https://github.com/grafana/helm-charts/tree/promtail-6.16.4/charts/promtail

`values.yaml` 配置文件：

```yaml
daemonset:
  enabled: true
resources:
  limits:
    cpu: 200m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
defaultVolumes:
  - name: run
    hostPath:
      path: /run/promtail
  - name: pods
    hostPath:
      path: /var/log/pods
defaultVolumeMounts:
  - name: run
    mountPath: /run/promtail
  - name: pods
    mountPath: /var/log/pods
    readOnly: true
config:
  enabled: true
  serverPort: 3101
  clients:
    - url: http://loki:3100/loki/api/v1/push
      tenant_id: k8s
      timeout: 10s
  positions:
    filename: /run/promtail/positions.yaml
    sync_period: 10s
```

## 微信公众号

更多内容请关注微信公众号：gopher云原生

<img src="https://github.com/user-attachments/assets/ea93572c-6c05-4751-bde7-35a58fe083f1" width="520px" />
