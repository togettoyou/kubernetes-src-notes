package main

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type Controller struct {
	workqueue workqueue.RateLimitingInterface
	lister    v1.PodLister
	informer  cache.Controller
}

func NewController(
	workqueue workqueue.RateLimitingInterface,
	lister v1.PodLister,
	informer cache.Controller) *Controller {
	return &Controller{
		workqueue: workqueue,
		lister:    lister,
		informer:  informer,
	}
}

func (c *Controller) Run(ctx context.Context, workers int) {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	logger := klog.FromContext(ctx)
	logger.Info("Starting controller")
	logger.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced); !ok {
		utilruntime.HandleError(fmt.Errorf("failed to wait for caches to sync"))
		return
	}

	logger.Info("Starting workers", "count", workers)

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	key, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	defer c.workqueue.Done(key)

	err := c.syncHandler(key.(string))
	c.handleErr(ctx, err, key)
	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	pod, err := c.lister.Pods(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("pod '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	fmt.Printf("Sync/Add/Update/Delete for Pod %s/%s\n", pod.GetNamespace(), pod.GetName())

	return nil
}

func (c *Controller) handleErr(ctx context.Context, err error, key interface{}) {
	if err == nil {
		c.workqueue.Forget(key)
		return
	}

	logger := klog.FromContext(ctx)

	if c.workqueue.NumRequeues(key) < 3 {
		logger.Info("Error syncing", key, err)
		c.workqueue.AddRateLimited(key)
		return
	}

	c.workqueue.Forget(key)
	utilruntime.HandleError(err)
}
