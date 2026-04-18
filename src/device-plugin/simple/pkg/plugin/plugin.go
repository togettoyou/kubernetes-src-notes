package plugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	// ResourceName 是在 kube-apiserver 中注册的资源名称
	// 命名规范：<vendor-domain>/<resource-type>，不能以 kubernetes.io 开头
	ResourceName = "simple.io/fake-device"

	// DeviceCount 是本插件向 kubelet 上报的虚拟设备总数
	DeviceCount = 5

	// SocketName 是本插件在 device-plugins 目录下创建的 Unix Socket 文件名
	// kubelet 通过此 socket 调用插件的 gRPC 接口
	SocketName = "simple-device.sock"

	// SocketDir 是 kubelet 与 Device Plugin 通信的 socket 目录
	SocketDir = "/var/lib/kubelet/device-plugins"

	// KubeletSocket 是 kubelet 注册服务的 socket 文件名
	// 插件启动后通过此 socket 向 kubelet 完成注册
	KubeletSocket = "kubelet.sock"
)

// SimplePlugin 实现了 Device Plugin gRPC 服务
type SimplePlugin struct {
	server *grpc.Server
	stop   chan struct{}
}

func NewSimplePlugin() *SimplePlugin {
	return &SimplePlugin{
		stop: make(chan struct{}),
	}
}

// Start 启动 gRPC 服务并向 kubelet 完成注册
// 必须先启动 gRPC 服务，再向 kubelet 注册——kubelet 注册成功后会立即回调 ListAndWatch
func (p *SimplePlugin) Start() error {
	socketPath := filepath.Join(SocketDir, SocketName)

	// 清理可能残留的旧 socket 文件（插件崩溃或 kubelet 重启后可能存在）
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old socket %s: %v", socketPath, err)
	}

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", socketPath, err)
	}

	p.server = grpc.NewServer()
	pluginapi.RegisterDevicePluginServer(p.server, p)

	go func() {
		if err := p.server.Serve(lis); err != nil {
			klog.Errorf("gRPC server stopped: %v", err)
		}
	}()
	klog.Infof("Device plugin gRPC server started at %s", socketPath)

	// 向 kubelet 发起注册，告知资源名、socket 路径和 API 版本
	if err := p.register(); err != nil {
		return fmt.Errorf("failed to register with kubelet: %v", err)
	}

	// 后台监控 kubelet 是否重启，重启后需要重新注册
	go p.watchKubeletRestart()

	return nil
}

// Stop 停止 gRPC 服务和后台 goroutine
func (p *SimplePlugin) Stop() {
	close(p.stop)
	p.server.Stop()
}

// register 通过 kubelet.sock 向 kubelet 的 Registration gRPC 服务完成注册
func (p *SimplePlugin) register() error {
	kubeletSocket := filepath.Join(SocketDir, KubeletSocket)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		"unix://"+kubeletSocket,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to kubelet at %s: %v", kubeletSocket, err)
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	_, err = client.Register(context.Background(), &pluginapi.RegisterRequest{
		Version: pluginapi.Version,
		// Endpoint 只需文件名，kubelet 会在 SocketDir 下查找此 socket
		Endpoint: SocketName,
		// ResourceName 决定了 Node.status.capacity 中的资源键名
		ResourceName: ResourceName,
		Options:      &pluginapi.DevicePluginOptions{},
	})
	if err != nil {
		return fmt.Errorf("registration call failed: %v", err)
	}

	klog.Infof("Successfully registered with kubelet: resource=%s, devices=%d", ResourceName, DeviceCount)
	return nil
}

// watchKubeletRestart 通过 fsnotify 监听 device-plugins 目录的文件系统事件
// kubelet 重启后会删除并重建 kubelet.sock；插件检测到该文件被创建后重新注册，
// 否则 kubelet 不会再调用本插件的 ListAndWatch，节点资源会丢失
func (p *SimplePlugin) watchKubeletRestart() {
	kubeletSocket := filepath.Join(SocketDir, KubeletSocket)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		klog.Errorf("Failed to create fsnotify watcher: %v", err)
		return
	}
	defer watcher.Close()

	// 监听整个 device-plugins 目录，因为 kubelet.sock 被删除时无法直接 Watch 已不存在的文件
	if err := watcher.Add(SocketDir); err != nil {
		klog.Errorf("Failed to watch %s: %v", SocketDir, err)
		return
	}

	for {
		select {
		case <-p.stop:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// kubelet 重启时会先 Remove 再 Create kubelet.sock
			// 只关注 kubelet.sock 的 Create 事件，触发重新注册
			if event.Name == kubeletSocket && event.Has(fsnotify.Create) {
				klog.Info("kubelet.sock recreated, re-registering...")
				if err := p.register(); err != nil {
					klog.Errorf("Re-registration failed: %v", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			klog.Errorf("fsnotify error: %v", err)
		}
	}
}

// -------- 以下是 DevicePlugin gRPC 接口实现 --------

// GetDevicePluginOptions 声明本插件支持哪些可选能力
// 返回 false 表示不实现 GetPreferredAllocation 和 PreStartContainer
func (p *SimplePlugin) GetDevicePluginOptions(_ context.Context, _ *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		GetPreferredAllocationAvailable: false,
		PreStartRequired:                false,
	}, nil
}

// ListAndWatch 以流的形式向 kubelet 推送设备列表
// kubelet 注册成功后立即调用此方法，之后持续保持 stream 开启
// 当设备状态发生变化时（如设备故障），需要主动推送新列表（Health 改为 Unhealthy）
// kubelet 根据 Healthy 状态的设备数量更新 Node.status.allocatable 中的资源数量
func (p *SimplePlugin) ListAndWatch(_ *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	klog.Info("ListAndWatch called by kubelet")

	// 首次推送全部设备，kubelet 据此更新 Node 可分配资源数量
	devs := make([]*pluginapi.Device, DeviceCount)
	for i := 0; i < DeviceCount; i++ {
		devs[i] = &pluginapi.Device{
			ID:     fmt.Sprintf("fake-device-%d", i),
			Health: pluginapi.Healthy,
		}
	}
	if err := stream.Send(&pluginapi.ListAndWatchResponse{Devices: devs}); err != nil {
		return err
	}
	klog.Infof("Reported %d devices to kubelet", DeviceCount)

	// 保持 stream 开启，等待插件停止信号
	// 真实场景中：在这里监听硬件状态变化事件，设备故障时重新推送带有 Unhealthy 状态的列表
	<-p.stop
	return nil
}

// Allocate 在每个容器创建前由 kubelet 调用
// req 中包含本次分配的设备 ID 列表（一个 ContainerRequests 对应一个容器）
// 返回的 ContainerAllocateResponse 告诉容器运行时如何让容器访问到设备：
//   - Envs：注入环境变量（适合传递设备标识符、配置参数）
//   - Mounts：挂载宿主机路径（适合传递设备文件、共享库）
//   - Devices：暴露设备节点（适合真实硬件，如 /dev/gpu0）
func (p *SimplePlugin) Allocate(_ context.Context, req *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var responses pluginapi.AllocateResponse

	for _, r := range req.ContainerRequests {
		klog.Infof("Allocate: container requested devices %v", r.DevicesIDs)

		// 通过环境变量将分配的设备 ID 传入容器
		// 真实场景中可能还需要通过 Mounts 挂载驱动库，或通过 Devices 暴露设备节点
		resp := &pluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				"FAKE_DEVICE_IDS": fmt.Sprintf("%v", r.DevicesIDs),
			},
		}
		responses.ContainerResponses = append(responses.ContainerResponses, resp)
	}

	return &responses, nil
}

// GetPreferredAllocation 返回推荐的设备分配方案（可选接口，本插件未实现）
// 调度器在有多个设备可选时调用此接口，插件可据此给出最优分配建议（如 NUMA 亲和性）
func (p *SimplePlugin) GetPreferredAllocation(_ context.Context, _ *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

// PreStartContainer 在容器启动前调用（可选接口，本插件未实现）
// 可用于设备重置、状态清理等初始化操作
func (p *SimplePlugin) PreStartContainer(_ context.Context, _ *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}
