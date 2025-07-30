package main

import (
	"context"
	"flag"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
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

	// 初始化 Manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{})
	if err != nil {
		panic(err)
	}

	// 初始化 controller
	err = ctrl.NewControllerManagedBy(mgr).
		Named("pod-controller").
		For(&corev1.Pod{}).
		Complete(&podReconciler{client: mgr.GetClient()})
	if err != nil {
		panic(err)
	}

	// 启动 Manager
	if err := mgr.Start(context.Background()); err != nil {
		panic(err)
	}
}
