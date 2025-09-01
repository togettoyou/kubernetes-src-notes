---
title: 在 Kubernetes 环境中 debug go 程序
---

### 一、利用 sa token 直连方式

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: local-dev
rules:
  - apiGroups: [ "" ]
    resources:
      - pods
    verbs: [ "get", "list", "watch" ]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: local-dev
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: local-dev
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: local-dev
subjects:
  - kind: ServiceAccount
    name: local-dev
    namespace: default
---
apiVersion: v1
kind: Secret
metadata:
  annotations:
    kubernetes.io/service-account.name: local-dev
  name: local-dev
  namespace: default
type: kubernetes.io/service-account-token
```

查看 token ：

```bash
kubectl get secret local-dev -n default -o jsonpath="{.data.token}" | base64 --decode
```

适合使用方式如下：

```go
var cfg *rest.Config
cfg = &rest.Config{
    Host:        host,
    BearerToken: token,
    TLSClientConfig: rest.TLSClientConfig{
        Insecure: true,
    },
}
```

### 二、在 Pod 中运行

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: golang-debug
rules:
  - apiGroups: [ "" ]
    resources:
      - pods
    verbs: [ "get", "list", "watch" ]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: golang-debug
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: golang-debug
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: golang-debug
  #  或者直接绑定集群管理员
  #  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: golang-debug
    namespace: default
---
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: golang-debug
  labels:
    app: golang-debug
spec:
  serviceAccountName: golang-debug
  containers:
    - name: golang-debug
      image: registry.cn-hangzhou.aliyuncs.com/hubmirrorbytogettoyou/golang-debug:1.24.6
      imagePullPolicy: IfNotPresent
      env:
        - name: GOPROXY
          value: "https://goproxy.cn,direct"
        # 记得修改密码
        - name: ROOT_PASSWORD
          value: "123456"
      ports:
        - name: ssh
          containerPort: 22
      command:
        - sh
        - -c
        - |
          echo "root:${ROOT_PASSWORD}" | chpasswd && \
          unset ROOT_PASSWORD && \
          printenv > /etc/environment && \
          /usr/sbin/sshd -D
      resources:
        requests:
          memory: "256Mi"
          cpu: "200m"
        limits:
          memory: "2Gi"
          cpu: "2"
---
apiVersion: v1
kind: Service
metadata:
  namespace: default
  name: golang-debug
spec:
  selector:
    app: golang-debug
  type: NodePort
  ports:
    - name: ssh
      port: 22
      targetPort: 22
      nodePort: 32222
```

其中使用到的镜像 Dockerfile 参考如下：

```dockerfile
FROM golang:1.24.6

RUN apt-get update && \
    apt-get install -y openssh-server && \
    rm -rf /var/lib/apt/lists/*

RUN echo "root:123456" | chpasswd && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    mkdir -p /var/run/sshd

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D"]
```

适合使用方式如下：

```go
var cfg *rest.Config
cfg, err = rest.InClusterConfig()
```
