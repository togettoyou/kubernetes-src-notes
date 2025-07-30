package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type podReconciler struct {
	client client.Client
}

var _ reconcile.Reconciler = &podReconciler{}

func (r *podReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := klog.FromContext(ctx)

	pod := &corev1.Pod{}

	err := r.client.Get(ctx, request.NamespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(nil, "Could not find Pod", request.NamespacedName)
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	fmt.Printf("Sync/Add/Update/Delete for Pod %s/%s\n", pod.GetNamespace(), pod.GetName())

	return reconcile.Result{}, nil
}
