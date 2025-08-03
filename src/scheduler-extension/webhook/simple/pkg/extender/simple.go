package extender

import (
	"context"
	"log"
	"sort"

	corev1 "k8s.io/api/core/v1"
	extenderapi "k8s.io/kube-scheduler/extender/v1"
)

type simpleHandler struct{}

func NewSimpleHandler() Handler {
	return &simpleHandler{}
}

// Filter 过滤节点，名字带 "bad" 的节点不通过
func (h *simpleHandler) Filter(ctx context.Context, args extenderapi.ExtenderArgs) (*extenderapi.ExtenderFilterResult, error) {
	var filtered []corev1.Node
	failed := make(map[string]string)

	log.Printf("[过滤节点] 收到调度请求，候选节点数：%d\n", len(args.Nodes.Items))

	for _, node := range args.Nodes.Items {
		if len(node.Name) >= 3 && node.Name[:3] == "bad" {
			failed[node.Name] = "节点名包含 'bad' 字符串"
			log.Printf("[过滤节点] 节点 %s 被过滤，原因：节点名包含 'bad'\n", node.Name)
			continue
		}
		filtered = append(filtered, node)
		log.Printf("[过滤节点] 节点 %s 通过过滤\n", node.Name)
	}

	log.Printf("[过滤节点] 过滤完成，剩余节点数：%d，过滤失败节点数：%d\n", len(filtered), len(failed))

	return &extenderapi.ExtenderFilterResult{
		Nodes:       &corev1.NodeList{Items: filtered},
		FailedNodes: failed,
	}, nil
}

// Prioritize 节点优先级，名字越短分数越高
func (h *simpleHandler) Prioritize(ctx context.Context, args extenderapi.ExtenderArgs) (*extenderapi.HostPriorityList, error) {
	var result extenderapi.HostPriorityList

	log.Printf("[节点优先级] 开始计算节点优先级，共 %d 个节点\n", len(args.Nodes.Items))

	for _, node := range args.Nodes.Items {
		score := 100 - len(node.Name)*10
		if score < 0 {
			score = 0
		}
		result = append(result, extenderapi.HostPriority{
			Host:  node.Name,
			Score: int64(score),
		})
		log.Printf("[节点优先级] 节点 %s 计算得分：%d\n", node.Name, score)
	}

	// 按分数倒序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	log.Println("[节点优先级] 节点优先级排序完成，排序结果：")
	for i, host := range result {
		log.Printf("  %d. 节点 %s, 分数 %d\n", i+1, host.Host, host.Score)
	}

	return &result, nil
}

// ProcessPreemption 抢占预处理
func (h *simpleHandler) ProcessPreemption(ctx context.Context, args extenderapi.ExtenderPreemptionArgs) (*extenderapi.ExtenderPreemptionResult, error) {
	log.Printf("[抢占处理] 开始处理抢占请求，候选节点数：%d", len(args.NodeNameToVictims))

	for node, victims := range args.NodeNameToVictims {
		log.Printf("[抢占处理] 节点 %s 有 %d 个被抢占 Pod：", node, len(victims.Pods))
		for _, pod := range victims.Pods {
			log.Printf("  - Pod: %s/%s", pod.Namespace, pod.Name)
		}
	}

	log.Printf("[抢占处理] 未做变更")
	return &extenderapi.ExtenderPreemptionResult{}, nil
}

// Bind 绑定操作
func (h *simpleHandler) Bind(ctx context.Context, args extenderapi.ExtenderBindingArgs) (*extenderapi.ExtenderBindingResult, error) {
	log.Printf("[绑定操作] 开始绑定 Pod %s/%s 到节点 %s\n", args.PodNamespace, args.PodName, args.Node)
	return &extenderapi.ExtenderBindingResult{Error: "这里演示绑定失败，真实绑定需调用 Kubernetes API"}, nil
}
