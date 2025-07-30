import {defineConfig} from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
    title: "Kubernetes 源码阅读",
    description: "Kubernetes 源码阅读|更多内容请关注微信公众号：gopher云原生",
    lang: 'zh-CN',
    cleanUrls: true,
    base: '/kubernetes-src-notes/',
    lastUpdated: true,
    head: [['link', {rel: 'icon', href: 'https://kubernetes.io/icons/favicon-32.png'}]],
    themeConfig: {
        logo: 'https://kubernetes.io/images/kubernetes.png',
        nav: [
            {text: '首页', link: '/'},
            {text: '关于我', link: '/about'}
        ],
        sidebar: [
            {
                text: '序言',
                link: '/'
            },
            {
                text: 'kube-apiserver',
                collapsed: true,
                items: [
                    {text: '01. apiserver 启动参数和调试准备', link: '/kube-apiserver/01'},
                    {text: '02. apiserver 的 HTTP Server 的初始化', link: '/kube-apiserver/02'},
                    {text: '03. KubeAPIServer 路由注册', link: '/kube-apiserver/03'},
                    {text: '04. KubeAPIServer 的存储接口实现', link: '/kube-apiserver/04'},
                    {text: '05. kube-apiserver 启动 HTTP Server', link: '/kube-apiserver/05'},
                    {text: '06. kube-apiserver handler 处理流程', link: '/kube-apiserver/06'},
                    {text: '07. kube-apiserver API 认证和鉴权', link: '/kube-apiserver/07'},
                    {text: '08. kube-apiserver API 准入控制', link: '/kube-apiserver/08'},
                    {text: '09. kube-apiserver 准入 Webhook', link: '/kube-apiserver/09'},
                    {text: '10. kube-apiserver 初始命名空间', link: '/kube-apiserver/10'},
                ]
            },
            {
                text: 'client-go',
                collapsed: true,
                items: [
                    {text: '01. client-go 四种客户端', link: '/client-go/01'},
                    {text: '02. client-go 的 Informer 机制', link: '/client-go/02'},
                ]
            },
            {
                text: 'kube-scheduler',
                collapsed: true,
                items: [
                    {text: '01. kube-scheduler 启动及前期调试准备', link: '/kube-scheduler/01'},
                    {text: '02. kube-scheduler 整体架构', link: '/kube-scheduler/02'},
                ]
            },
            {
                text: 'kubelet',
                collapsed: true,
                items: [
                    {text: '01. TODO', link: '/kubelet/01'},
                ]
            },
            {
                text: 'kube-controller-manager',
                collapsed: true,
                items: [
                    {text: '01. TODO', link: '/kube-controller-manager/01'},
                ]
            },
            {
                text: 'kube-proxy',
                collapsed: true,
                items: [
                    {text: '01. TODO', link: '/kube-proxy/01'},
                ]
            },
            {
                text: '扩展',
                collapsed: true,
                items: [
                    {text: 'CRI 容器运行时接口', link: '/extensions/cri'},
                    {text: 'CNI 容器网络接口', link: '/extensions/cni'},
                    {text: 'CSI 容器存储接口', link: '/extensions/csi'},
                    {text: '扩展 Kubernetes API', link: '/extensions/api-extension'},
                    {text: '准入 Webhook', link: '/extensions/admission-webhook'},
                    {text: '准入策略 CEL', link: '/extensions/admission-policy'},
                    {text: '控制器和 Operator 模式', link: '/extensions/controller-operator'},
                    {text: '调度扩展', link: '/extensions/scheduler-extension'},
                    {text: 'kubectl 插件', link: '/extensions/kubectl-plugin'},
                    {text: 'kubelet 设备插件', link: '/extensions/kubelet-device-plugin'},
                ]
            },
            {
                text: '可观测性',
                collapsed: true,
                items: [
                    {text: 'Grafana', link: '/observability/grafana'},
                    {
                        text: '指标',
                        collapsed: false,
                        items: [
                            {text: 'Prometheus', link: '/observability/metrics/prometheus'},
                        ]
                    },
                    {
                        text: '日志',
                        collapsed: false,
                        items: [
                            {text: 'Loki', link: '/observability/log/loki'},
                            {text: 'Promtail', link: '/observability/log/promtail'},
                        ]
                    },
                ]
            },
            {
                text: '附录',
                collapsed: true,
                items: [
                    {text: '使用 kubeadm 创建集群', link: '/appendix/install'},
                ]
            }
        ],
        socialLinks: [
            {icon: 'github', link: 'https://github.com/togettoyou/kubernetes-src-notes'}
        ],
    }
})
