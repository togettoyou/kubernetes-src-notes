package plugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	drapb "k8s.io/kubelet/pkg/apis/dra/v1"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

const (
	// DriverName 与 controller 中保持一致，用于构造 socket 路径和注册信息
	DriverName = "fake.dra.example.com"

	// PluginsDir 是 kubelet 约定的驱动 socket 根目录
	// 每个驱动在此目录下创建以驱动名命名的子目录，并在其中放置 dra.sock
	PluginsDir = "/var/lib/kubelet/plugins"

	// RegistryDir 是 kubelet plugin manager 监听的注册目录
	// 驱动在此目录下创建注册 socket，kubelet 检测到后主动来调用 GetInfo
	RegistryDir = "/var/lib/kubelet/plugins_registry"

	// DRASocketName 是驱动 gRPC 服务的 socket 文件名
	DRASocketName = "dra.sock"
)

// DRAPlugin 实现了两个 gRPC 服务：
//  1. DRAPlugin（DRA 业务接口）：kubelet 通过此接口调用 NodePrepareResources / NodeUnprepareResources
//  2. Registration（注册接口）：kubelet plugin manager 通过此接口完成驱动注册握手
type DRAPlugin struct {
	drapb.UnimplementedDRAPluginServer
	registerapi.UnimplementedRegistrationServer

	draServer *grpc.Server
	regServer *grpc.Server

	// draSocketPath 是 DRA gRPC 服务的完整 socket 路径，注册时告知 kubelet
	draSocketPath string
	nodeName      string
}

func NewDRAPlugin() *DRAPlugin {
	return &DRAPlugin{
		nodeName: os.Getenv("NODE_NAME"),
	}
}

// Start 按顺序完成：
//  1. 在 PluginsDir/<driver-name>/dra.sock 启动 DRA gRPC 服务
//  2. 在 RegistryDir/<driver-name>.sock 启动注册 gRPC 服务
//
// kubelet plugin manager 持续监听 RegistryDir，发现新 socket 后调用 GetInfo 完成握手
func (p *DRAPlugin) Start() error {
	pluginDir := filepath.Join(PluginsDir, DriverName)
	if err := os.MkdirAll(pluginDir, 0750); err != nil {
		return fmt.Errorf("failed to create plugin dir: %v", err)
	}

	p.draSocketPath = filepath.Join(pluginDir, DRASocketName)

	if err := p.startDRAServer(); err != nil {
		return err
	}
	if err := p.startRegistrationServer(); err != nil {
		return err
	}
	return nil
}

// startDRAServer 在 dra.sock 上启动 DRA gRPC 服务
func (p *DRAPlugin) startDRAServer() error {
	if err := os.Remove(p.draSocketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old dra socket: %v", err)
	}

	lis, err := net.Listen("unix", p.draSocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", p.draSocketPath, err)
	}

	p.draServer = grpc.NewServer()
	drapb.RegisterDRAPluginServer(p.draServer, p)

	go func() {
		if err := p.draServer.Serve(lis); err != nil {
			klog.Errorf("DRA gRPC server stopped: %v", err)
		}
	}()
	klog.Infof("DRA plugin gRPC server started at %s", p.draSocketPath)
	return nil
}

// startRegistrationServer 在 RegistryDir/<driver-name>.sock 上启动注册 gRPC 服务
// kubelet plugin manager 检测到此 socket 后会调用 GetInfo 获取驱动类型和 dra.sock 路径
func (p *DRAPlugin) startRegistrationServer() error {
	regSocketPath := filepath.Join(RegistryDir, DriverName+".sock")

	if err := os.Remove(regSocketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old reg socket: %v", err)
	}

	lis, err := net.Listen("unix", regSocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", regSocketPath, err)
	}

	p.regServer = grpc.NewServer()
	registerapi.RegisterRegistrationServer(p.regServer, p)

	go func() {
		if err := p.regServer.Serve(lis); err != nil {
			klog.Errorf("Registration gRPC server stopped: %v", err)
		}
	}()
	klog.Infof("Registration server started at %s", regSocketPath)
	return nil
}

// Stop 停止两个 gRPC 服务
func (p *DRAPlugin) Stop() {
	if p.draServer != nil {
		p.draServer.Stop()
	}
	if p.regServer != nil {
		p.regServer.Stop()
	}
}

// -------- Registration gRPC 接口实现 --------

// GetInfo 是 kubelet plugin manager 握手的第一步
// kubelet 通过返回值确认：这是一个 DRAPlugin，DRA socket 在哪里
func (p *DRAPlugin) GetInfo(_ context.Context, _ *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	return &registerapi.PluginInfo{
		Type:              registerapi.DRAPlugin,
		Name:              DriverName,
		Endpoint:          p.draSocketPath,
		SupportedVersions: []string{"v1"},
	}, nil
}

// NotifyRegistrationStatus 是 kubelet plugin manager 握手的第二步
// kubelet 通知驱动注册是否成功
func (p *DRAPlugin) NotifyRegistrationStatus(_ context.Context, status *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
	if !status.PluginRegistered {
		klog.Errorf("Plugin registration failed: %s", status.Error)
	} else {
		klog.Info("Plugin successfully registered with kubelet")
	}
	return &registerapi.RegistrationStatusResponse{}, nil
}

// -------- DRA gRPC 接口实现 --------

// NodePrepareResources 在 Pod 调度到本节点后由 kubelet 调用
// kubelet 传入已分配给 Pod 的 ResourceClaim 列表，驱动负责准备好对应设备，
// 并返回 CDI 设备 ID，kubelet 将这些 ID 传给容器运行时（containerd），
// 容器运行时通过 CDI 规范将设备注入容器
//
// Claim 中只含 namespace/uid/name，驱动需自行从 ResourceClaim 的
// status.allocation 字段读取已分配设备信息（生产实现中通过 lister 完成）
// 此示例为演示流程，直接返回一个固定的虚拟设备
func (p *DRAPlugin) NodePrepareResources(ctx context.Context, req *drapb.NodePrepareResourcesRequest) (*drapb.NodePrepareResourcesResponse, error) {
	resp := &drapb.NodePrepareResourcesResponse{
		Claims: make(map[string]*drapb.NodePrepareResourceResponse),
	}

	for _, claim := range req.Claims {
		klog.Infof("NodePrepareResources: claim=%s/%s uid=%s", claim.Namespace, claim.Name, claim.UID)

		// CDI 设备 ID 格式：<vendor>/<class>=<name>
		// 容器运行时通过此 ID 在 /etc/cdi/ 中查找设备描述文件，完成设备注入
		cdiID := fmt.Sprintf("%s/device=fake-0", DriverName)

		// 真实场景：在这里执行设备初始化操作（如创建软链接、分配显存分区等）
		klog.Infof("  prepared device: cdi=%s", cdiID)

		resp.Claims[claim.UID] = &drapb.NodePrepareResourceResponse{
			Devices: []*drapb.Device{
				{
					// RequestNames 对应 ResourceClaim.spec.devices.requests[*].name
					// 此示例固定填写，生产实现中应从 ResourceClaim 读取
					RequestNames: []string{"my-request"},
					PoolName:     p.nodeName,
					DeviceName:   "fake-0",
					CDIDeviceIDs: []string{cdiID},
				},
			},
		}
	}

	return resp, nil
}

// NodeUnprepareResources 在 Pod 删除后由 kubelet 调用
// 驱动负责释放此前为这些 ResourceClaim 准备的设备资源
func (p *DRAPlugin) NodeUnprepareResources(ctx context.Context, req *drapb.NodeUnprepareResourcesRequest) (*drapb.NodeUnprepareResourcesResponse, error) {
	for _, claim := range req.Claims {
		klog.Infof("NodeUnprepareResources: claim=%s/%s uid=%s", claim.Namespace, claim.Name, claim.UID)
		// 真实场景：在这里释放设备资源（如删除软链接、归还显存分区等）
	}
	return &drapb.NodeUnprepareResourcesResponse{}, nil
}
