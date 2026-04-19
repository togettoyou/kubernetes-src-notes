package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"simple-dra/pkg/controller"
	"simple-dra/pkg/plugin"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func main() {
	klog.Info("Starting simple DRA driver")

	// 在集群内通过 ServiceAccount 获取访问 API Server 的凭证
	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to get in-cluster config: %v", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Failed to create kubernetes client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动 Controller：向 API Server 发布本节点的 ResourceSlice
	// ResourceSlice 是调度器看到设备的唯一途径，必须在 kubelet plugin 之前就绪
	ctrl := controller.NewController(client)
	if err := ctrl.PublishResourceSlice(ctx); err != nil {
		klog.Fatalf("Failed to publish ResourceSlice: %v", err)
	}

	// 启动 kubelet Plugin：监听 dra.sock，处理 NodePrepareResources / NodeUnprepareResources
	p := plugin.NewDRAPlugin()
	if err := p.Start(); err != nil {
		klog.Fatalf("Failed to start DRA plugin: %v", err)
	}

	// 等待终止信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	klog.Info("Shutting down simple DRA driver")
	p.Stop()

	// 删除 ResourceSlice，让调度器感知本节点设备已下线
	if err := ctrl.DeleteResourceSlice(context.Background()); err != nil {
		klog.Errorf("Failed to delete ResourceSlice: %v", err)
	}
}
