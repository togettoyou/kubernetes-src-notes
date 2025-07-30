package simple

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const Name = "SimplePlugin"

type plugin struct {
	handle framework.Handle
}

func (pl *plugin) Name() string {
	return Name
}

func New(_ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &plugin{handle: h}, nil
}

var (
	_ framework.PreFilterPlugin = &plugin{}
	_ framework.FilterPlugin    = &plugin{}
	_ framework.PreBindPlugin   = &plugin{}
	_ framework.BindPlugin      = &plugin{}
	_ framework.PostBindPlugin  = &plugin{}
)

// PreFilter 调度前对 Pod 进行预处理
func (pl *plugin) PreFilter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod) (*framework.PreFilterResult, *framework.Status) {
	if _, ok := pod.Labels["simple.io/required"]; !ok {
		fmt.Printf("[PreFilter] Pod %s/%s 缺少必要的标签 simple.io/required，调度失败\n", pod.Namespace, pod.Name)
		return nil, framework.NewStatus(framework.Unschedulable, "缺少必要的 pod 标签 simple.io/required")
	}
	fmt.Printf("[PreFilter] Pod %s/%s 标签检查通过\n", pod.Namespace, pod.Name)
	return &framework.PreFilterResult{}, framework.NewStatus(framework.Success)
}

// PreFilterExtensions 可选择返回 PreFilter 的扩展接口
func (pl *plugin) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// Filter 用于过滤节点
func (pl *plugin) Filter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	fmt.Printf("[Filter] Pod %s/%s 评估节点 %s\n", pod.Namespace, pod.Name, nodeInfo.Node().Name)
	return framework.NewStatus(framework.Success)
}

// PreBind 在绑定前最后校验
func (pl *plugin) PreBind(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) *framework.Status {
	fmt.Printf("[PreBind] Pod %s/%s 准备绑定到节点 %s\n", pod.Namespace, pod.Name, nodeName)
	return framework.NewStatus(framework.Success)
}

// Bind 执行绑定
func (pl *plugin) Bind(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) *framework.Status {
	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{Namespace: pod.Namespace, Name: pod.Name, UID: pod.UID},
		Target:     corev1.ObjectReference{Kind: "Node", Name: nodeName},
	}
	err := pl.handle.ClientSet().CoreV1().Pods(binding.Namespace).Bind(ctx, binding, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("[Bind] Pod %s/%s 绑定到节点 %s 失败: %v\n", pod.Namespace, pod.Name, nodeName, err)
		return framework.AsStatus(err)
	}
	fmt.Printf("[Bind] Pod %s/%s 成功绑定到节点 %s\n", pod.Namespace, pod.Name, nodeName)
	return framework.NewStatus(framework.Success)
}

// PostBind 绑定后执行
func (pl *plugin) PostBind(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) {
	fmt.Printf("[PostBind] Pod %s/%s 已成功绑定到节点 %s\n", pod.Namespace, pod.Name, nodeName)
}
