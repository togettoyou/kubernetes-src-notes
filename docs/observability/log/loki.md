---
title: Loki
outline: deep
---

# Loki

## 在线Chart仓库方式

```shell
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
```

```shell
kubectl create ns monitoring
helm install loki grafana/loki -f values.yaml --version 6.7.3 -n monitoring
```

## 本地离线Chart方式

```shell
wget https://github.com/grafana/helm-charts/releases/download/helm-loki-6.7.3/loki-6.7.3.tgz
```

```shell
kubectl create ns monitoring
helm install loki loki-6.7.3.tgz -f values.yaml -n monitoring
```

## 配置参考

https://github.com/grafana/loki/tree/helm-loki-6.7.3/production/helm/loki

### ALL 单体模式

`values.yaml` 配置文件：

```yaml
deploymentMode: SingleBinary
singleBinary:
  replicas: 1
  resources:
    limits:
      cpu: 2
      memory: 4Gi
    requests:
      cpu: 1
      memory: 1Gi
  autoscaling:
    enabled: false
  persistence:
    enableStatefulSetAutoDeletePVC: true
    enabled: true
    size: 10Gi
    storageClass: local-storage
  service:
    annotations:
      prometheus.io/scrape: "true"
      prometheus.io/port: "3100"
backend:
  replicas: 0
read:
  replicas: 0
write:
  replicas: 0
test:
  enabled: false
lokiCanary:
  enabled: false
gateway:
  enabled: false
chunksCache:
  enabled: false
resultsCache:
  enabled: false
loki:
  auth_enabled: true
  server:
    http_listen_port: 3100
    grpc_listen_port: 9095
    http_server_read_timeout: 600s
    http_server_write_timeout: 600s
    grpc_server_max_recv_msg_size: 4194304
    grpc_server_max_send_msg_size: 4194304
  limits_config:
    ingestion_rate_mb: 16
    ingestion_burst_size_mb: 32
    reject_old_samples: true
    reject_old_samples_max_age: 168h
    split_queries_by_interval: 15m
    query_timeout: 2m
  commonConfig:
    replication_factor: 1
    path_prefix: /var/loki
  ingester:
    max_chunk_age: 2m
    chunk_idle_period: 30s
    flush_check_period: 5s
  storage:
    type: filesystem
    filesystem:
      chunks_directory: /var/loki/chunks
      rules_directory: /var/loki/rules
  compactor:
    working_directory: /var/loki/compactor/retention
    compaction_interval: 10m
    retention_enabled: true
    retention_delete_delay: 2h
    delete_request_cancel_period: 1h
    delete_request_store: filesystem
  storage_config:
    tsdb_shipper:
      active_index_directory: /var/loki/tsdb_shipper-active
      cache_location: /var/loki/tsdb_shipper-cache
      cache_ttl: 24h
      resync_interval: 5m
  schemaConfig:
    configs:
      - from: 2024-07-01
        store: tsdb
        object_store: filesystem
        schema: v13
        index:
          prefix: index_
          period: 24h
```

### SSD 简单可扩展部署模式

`values.yaml` 配置文件：

```yaml
# TODO
```

## 附录

1. 日志推送

   ```bash
   curl -H "Content-Type: application/json" -XPOST -s "http://loki.monitoring.svc.cluster.local:3100/loki/api/v1/push"  \
   --data "{\"streams\": [{\"stream\": {\"job\": \"test\"}, \"values\": [[\"$(date +%s)000000000\", \"hello loki\"]]}]}" \
   -H X-Scope-OrgId:test
   ```

2. 日志查询

   ```bash
   curl "http://loki.monitoring.svc.cluster.local:3100/loki/api/v1/query_range" --data-urlencode 'query={job="test"}' -H X-Scope-OrgId:test | jq .data.result
   ```

3. LocalPV 参考：

   ```yaml
   apiVersion: v1
   kind: PersistentVolume
   metadata:
     name: local-pv-loki-node1
   spec:
     capacity:
       storage: 10Gi
     volumeMode: Filesystem
     accessModes:
       - ReadWriteOnce
     persistentVolumeReclaimPolicy: Retain
     storageClassName: local-storage
     local:
       path: /mnt/localpv/loki
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
