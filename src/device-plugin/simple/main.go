package main

import (
	"os"
	"os/signal"
	"syscall"

	"simple/pkg/plugin"

	"k8s.io/klog/v2"
)

func main() {
	klog.Info("Starting simple device plugin")

	p := plugin.NewSimplePlugin()

	// Start 依次完成：
	//   1. 在 /var/lib/kubelet/device-plugins/simple-device.sock 启动 gRPC 服务
	//   2. 通过 kubelet.sock 向 kubelet 完成注册
	//   3. 后台启动 kubelet 重启监控 goroutine
	if err := p.Start(); err != nil {
		klog.Fatalf("Failed to start device plugin: %v", err)
	}

	// 等待终止信号（SIGTERM 由 DaemonSet 滚动升级时发送，SIGINT 用于本地调试）
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	klog.Info("Shutting down simple device plugin")
	p.Stop()
}
