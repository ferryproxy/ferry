package controller

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/ferry-proxy/ferry/pkg/utils/trybuffer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type EndpointWatcher struct {
	mut       sync.Mutex
	lastIPs   []string
	try       *trybuffer.TryBuffer
	name      string
	namespace string
	clientset kubernetes.Interface
	syncFunc  func(ips []string)
}

type EndpointWatcherConfig struct {
	Name      string
	Namespace string
	Clientset kubernetes.Interface
	SyncFunc  func(ips []string)
}

func NewEndpointWatcher(conf *EndpointWatcherConfig) *EndpointWatcher {
	return &EndpointWatcher{
		name:      conf.Name,
		namespace: conf.Namespace,
		clientset: conf.Clientset,
		syncFunc:  conf.SyncFunc,
	}
}

func (e *EndpointWatcher) Run(ctx context.Context) error {
	fieldSelector := fmt.Sprintf("metadata.name=%s", e.name)
	watch, err := e.clientset.
		CoreV1().
		Endpoints(e.namespace).
		Watch(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
	if err != nil {
		return fmt.Errorf("failed to watch service: %w", err)
	}

	e.try = trybuffer.NewTryBuffer(e.sync, time.Second/2)

	for {
		select {
		case <-ctx.Done():
			e.try.Close()
			return nil
		case event, ok := <-watch.ResultChan():
			if !ok {
				return nil
			}
			ep := event.Object.(*corev1.Endpoints)
			ips := getIPs(ep)
			if len(ips) == 0 {
				continue
			}
			sort.Strings(ips)

			e.mut.Lock()
			if !reflect.DeepEqual(e.lastIPs, ips) {
				e.lastIPs = ips
				e.try.Try()
			}
			e.mut.Unlock()
		}
	}
}

func (e *EndpointWatcher) sync() {
	e.mut.Lock()
	defer e.mut.Unlock()
	e.syncFunc(e.lastIPs)
}

func getIPs(e *corev1.Endpoints) []string {
	var ips []string
	for _, subset := range e.Subsets {
		for _, address := range subset.Addresses {
			ips = append(ips, address.IP)
		}
	}
	return ips
}
