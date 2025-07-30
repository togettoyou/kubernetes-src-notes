package main

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api/proto"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/config"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog"
	klogapi "sigs.k8s.io/kube-scheduler-wasm-extension/guest/klog/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/plugin"
)

func main() {
	p, err := New(klog.Get(), config.Get())
	if err != nil {
		panic(err)
	}
	plugin.Set(p)
}

func New(klog klogapi.Klog, jsonConfig []byte) (api.Plugin, error) {
	var args simpleArgs
	if jsonConfig != nil {
		if err := json.Unmarshal(jsonConfig, &args); err != nil {
			panic(fmt.Errorf("decode arg into args: %w", err))
		}
		klog.Info("args is successfully applied")
	}
	return &Simple{args: args}, nil
}

type Simple struct {
	args simpleArgs
}

type simpleArgs struct {
}

var (
	_ api.PreFilterPlugin = &Simple{}
)

func (s *Simple) PreFilter(state api.CycleState, pod proto.Pod) (nodeNames []string, status *api.Status) {
	if _, ok := pod.GetLabels()["simple.io/required"]; !ok {
		fmt.Printf("[PreFilter] Pod %s/%s 缺少必要的标签 simple.io/required，调度失败\n", pod.GetNamespace(), pod.GetName())

		return nil, &api.Status{
			Code:   api.StatusCodeUnschedulable,
			Reason: "缺少必要的 pod 标签 simple.io/required",
		}
	}
	fmt.Printf("[PreFilter] Pod %s/%s 标签检查通过\n", pod.GetNamespace(), pod.GetName())
	return []string{}, &api.Status{Code: api.StatusCodeSuccess}
}
