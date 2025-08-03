package main

import (
	"context"
	"flag"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var (
	host  string
	token string
)

func init() {
	flag.StringVar(&host, "host", "", "连接自定义集群")
	flag.StringVar(&token, "token", "", "连接自定义集群的token")
}

func main() {
	flag.Parse()

	// 初始化集群配置
	var (
		cfg *rest.Config
		err error
	)
	if host != "" && token != "" {
		cfg = &rest.Config{
			Host:        host,
			BearerToken: token,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}

	// 连接集群
	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	// 创建 SharedInformerFactory 实例
	sharedInformerFactory := informers.NewSharedInformerFactory(clientSet, 0)

	// 创建 Pod Informer
	podInformer := sharedInformerFactory.Core().V1().Pods()

	// 创建工作队列
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// 添加监听事件，将key放置到工作队列
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	// 初始化控制器
	controller := NewController(queue, podInformer.Lister(), podInformer.Informer())

	ctx := context.Background()

	// 启动 Informer
	sharedInformerFactory.Start(ctx.Done())
	// 启动控制器
	go controller.Run(ctx, 1)

	select {}
}
