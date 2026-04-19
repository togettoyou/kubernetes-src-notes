package controller

import (
	"context"
	"fmt"
	"os"

	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	// DriverName 是本驱动在集群中的唯一标识，格式为反向域名
	// DeviceClass、ResourceSlice 等资源中的 spec.driver 字段均引用此名称
	DriverName = "fake.dra.example.com"

	// DeviceCount 是本驱动在每个节点上发布的虚拟设备数量
	DeviceCount = 5
)

// Controller 负责将节点上的可用设备发布到 API Server（ResourceSlice）
// ResourceSlice 是调度器进行设备调度的数据来源
type Controller struct {
	client   kubernetes.Interface
	nodeName string
}

func NewController(client kubernetes.Interface) *Controller {
	return &Controller{
		client:   client,
		nodeName: os.Getenv("NODE_NAME"),
	}
}

// PublishResourceSlice 向 API Server 创建或更新本节点的 ResourceSlice
// ResourceSlice 描述本节点上可用的设备列表及每个设备的属性
// 调度器通过读取 ResourceSlice 决定将 Pod 调度到哪个节点
func (c *Controller) PublishResourceSlice(ctx context.Context) error {
	sliceName := fmt.Sprintf("%s-%s", DriverName, c.nodeName)

	// k8s 1.34（resource/v1）中 Device 的 Attributes 和 Capacity 直接在 Device 上，
	// 不再通过 Basic *BasicDevice 包裹
	devices := make([]resourcev1.Device, DeviceCount)
	for i := 0; i < DeviceCount; i++ {
		devices[i] = resourcev1.Device{
			Name: fmt.Sprintf("fake-%d", i),
			// Attributes 是可供 ResourceClaim CEL 选择器查询的键值对
			// 调度器会将 ResourceClaim 的 selectors 与这些属性做匹配
			Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
				"model": {StringValue: strPtr("fake-v1")},
				"index": {IntValue: int64Ptr(int64(i))},
			},
			// Capacity 描述设备提供的可量化资源，供调度器计算资源余量
			Capacity: map[resourcev1.QualifiedName]resourcev1.DeviceCapacity{
				"memory": {Value: resource.MustParse("8Gi")},
			},
		}
	}

	// NodeName 在 resource/v1 中变为 *string
	slice := &resourcev1.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name: sliceName,
		},
		Spec: resourcev1.ResourceSliceSpec{
			Driver: DriverName,
			// Pool 是资源池，通常以节点名命名；调度器通过 Pool 将设备与节点关联
			Pool: resourcev1.ResourcePool{
				Name:               c.nodeName,
				Generation:         0,
				ResourceSliceCount: 1,
			},
			NodeName: &c.nodeName,
			Devices:  devices,
		},
	}

	existing, err := c.client.ResourceV1().ResourceSlices().Get(ctx, sliceName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = c.client.ResourceV1().ResourceSlices().Create(ctx, slice, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create ResourceSlice: %v", err)
		}
		klog.Infof("Created ResourceSlice %s with %d devices", sliceName, DeviceCount)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get ResourceSlice: %v", err)
	}

	// 更新已有 ResourceSlice（设备列表变化时使用）
	existing.Spec = slice.Spec
	existing.Spec.Pool.Generation++
	_, err = c.client.ResourceV1().ResourceSlices().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ResourceSlice: %v", err)
	}
	klog.Infof("Updated ResourceSlice %s", sliceName)
	return nil
}

// DeleteResourceSlice 在驱动退出时删除本节点的 ResourceSlice
func (c *Controller) DeleteResourceSlice(ctx context.Context) error {
	sliceName := fmt.Sprintf("%s-%s", DriverName, c.nodeName)
	err := c.client.ResourceV1().ResourceSlices().Delete(ctx, sliceName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ResourceSlice: %v", err)
	}
	klog.Infof("Deleted ResourceSlice %s", sliceName)
	return nil
}

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }
