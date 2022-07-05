package controller

import (
	"context"
	"reflect"
	"sync"

	"github.com/ferry-proxy/ferry/pkg/utils/objref"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type ServiceSyncer struct {
	mut           sync.Mutex
	ctx           context.Context
	ips           []string
	clientset     *kubernetes.Clientset
	labelSelector string
	cache         map[objref.ObjectRef]*corev1.Service
	logger        logr.Logger
}

type ServiceSyncerConfig struct {
	LabelSelector string
	Logger        logr.Logger
	Clientset     *kubernetes.Clientset
}

func NewServiceSyncer(conf *ServiceSyncerConfig) *ServiceSyncer {
	return &ServiceSyncer{
		labelSelector: conf.LabelSelector,
		cache:         map[objref.ObjectRef]*corev1.Service{},
		logger:        conf.Logger,
		clientset:     conf.Clientset,
	}
}

func (s *ServiceSyncer) Run(ctx context.Context) error {
	informer := informers.NewSharedInformerFactoryWithOptions(s.clientset, 0,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = s.labelSelector
		})).
		Core().
		V1().
		Services().
		Informer()
	s.ctx = ctx
	informer.AddEventHandler(s)
	informer.Run(ctx.Done())
	return nil
}

func (s *ServiceSyncer) UpdateIPs(ips []string) {
	s.mut.Lock()
	defer s.mut.Unlock()
	s.logger.Info("update ips", "old", s.ips, "ips", ips)
	s.ips = ips
	for _, svc := range s.cache {
		s.Update(svc)
	}
}

func (s *ServiceSyncer) Update(svc *corev1.Service) {
	svc = svc.DeepCopy()
	s.cache[objref.KObj(svc)] = svc
	if len(s.ips) == 0 {
		return
	}

	ep := toEndpoint(svc, s.ips)

	ori, err := s.clientset.
		CoreV1().
		Endpoints(ep.Namespace).
		Get(s.ctx, ep.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			s.logger.Error(err, "get endpoints failed")
			return
		}

		_, err = s.clientset.
			CoreV1().
			Endpoints(ep.Namespace).
			Create(s.ctx, ep, metav1.CreateOptions{})
		if err != nil {
			s.logger.Error(err, "create endpoints failed")
			return
		}
	} else {
		if reflect.DeepEqual(ori.Subsets, ep.Subsets) {
			return
		}

		ori.Subsets = ep.Subsets
		_, err = s.clientset.
			CoreV1().
			Endpoints(ep.Namespace).
			Update(s.ctx, ori, metav1.UpdateOptions{})
		if err != nil {
			s.logger.Error(err, "update endpoints failed")
			return
		}
	}
}

func (s *ServiceSyncer) Delete(svc *corev1.Service) {
	delete(s.cache, objref.KObj(svc))

	err := s.clientset.
		CoreV1().
		Endpoints(svc.Namespace).
		Delete(s.ctx, svc.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		s.logger.Error(err, "failed to delete endpoints")
	}
}

func (s *ServiceSyncer) OnAdd(obj interface{}) {
	svc := obj.(*corev1.Service)
	if len(svc.Spec.Selector) != 0 {
		return
	}
	s.logger.Info("add service", "service", objref.KObj(svc))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Update(svc)
}

func (s *ServiceSyncer) OnUpdate(oldObj, newObj interface{}) {
	svc := newObj.(*corev1.Service)
	if len(svc.Spec.Selector) != 0 {
		return
	}
	s.logger.Info("update service", "service", objref.KObj(svc))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Update(svc)
}

func (s *ServiceSyncer) OnDelete(obj interface{}) {
	svc := obj.(*corev1.Service)
	if len(svc.Spec.Selector) != 0 {
		return
	}
	s.logger.Info("delete service", "service", objref.KObj(svc))
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Delete(svc)
}

func toEndpoint(ori *corev1.Service, ips []string) *corev1.Endpoints {
	ep := corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ori.Name,
			Namespace:   ori.Namespace,
			Labels:      ori.Labels,
			Annotations: ori.Annotations,
		},
	}
	addresses := buildIPToEndpointAddress(ips)
	for _, p := range ori.Spec.Ports {
		port := p.TargetPort.IntVal
		if port == 0 {
			port = p.Port
		}
		ep.Subsets = append(ep.Subsets, corev1.EndpointSubset{
			Addresses: addresses,
			Ports: []corev1.EndpointPort{
				{
					Name:     p.Name,
					Port:     port,
					Protocol: p.Protocol,
				},
			},
		})
	}
	return &ep
}

func buildIPToEndpointAddress(ips []string) []corev1.EndpointAddress {
	eps := make([]corev1.EndpointAddress, 0, len(ips))
	for _, ip := range ips {
		eps = append(eps, corev1.EndpointAddress{
			IP: ip,
		})
	}
	return eps
}
