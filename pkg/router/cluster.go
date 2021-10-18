package router

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	ferry = "ferry-controller"
)

type ResourceBuilder interface {
	Build(proxy *Proxy, destinationServices []*corev1.Service) (Resourcer, error)
}

type ResourceBuilders []ResourceBuilder

func (r ResourceBuilders) Build(proxy *Proxy, destinationServices []*corev1.Service) (Resourcer, error) {
	var resourcers Resourcers
	for _, i := range r {
		resourcer, err := i.Build(proxy, destinationServices)
		if err != nil {
			return nil, err
		}
		resourcers = append(resourcers, resourcer)
	}
	return resourcers, nil
}

type Proxy struct {
	RemotePrefix string
	Labels       map[string]string
	EgressIPs    []string
	EgressPort   int32
	IngressIPs   []string
	IngressPort  int32
}

type Resourcer interface {
	Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error)
	Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error)
}

type Resourcers []Resourcer

func (r Resourcers) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	for _, i := range r {
		err = i.Apply(ctx, clientset)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r Resourcers) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	for _, i := range r {
		err = i.Delete(ctx, clientset)
		if err != nil {
			return err
		}
	}
	return nil
}

type Ingresses []Ingress

func (i Ingresses) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	for _, a := range i {
		err = a.Apply(ctx, clientset)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i Ingresses) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	for _, a := range i {
		err = a.Delete(ctx, clientset)
		if err != nil {
			return err
		}
	}
	return nil
}

type Ingress struct {
	Ingress *networkingv1.Ingress
}

func (i Ingress) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	_, err = clientset.NetworkingV1().Ingresses(i.Ingress.Namespace).
		Create(ctx, i.Ingress, metav1.CreateOptions{
			FieldManager: ferry,
		})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			i.Ingress.ResourceVersion = "0"
			_, err = clientset.NetworkingV1().Ingresses(i.Ingress.Namespace).
				Update(ctx, i.Ingress, metav1.UpdateOptions{
					FieldManager: ferry,
				})
			if err != nil {
				return fmt.Errorf("update ingrsss %s.%s: %w", i.Ingress.Name, i.Ingress.Namespace, err)
			}
		} else {
			return fmt.Errorf("create ingrsss %s.%s: %w", i.Ingress.Name, i.Ingress.Namespace, err)
		}
	}
	return nil
}

func (i Ingress) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	err = clientset.CoreV1().
		Services(i.Ingress.Namespace).
		Delete(ctx, i.Ingress.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete ingress %s.%s: %w", i.Ingress.Name, i.Ingress.Namespace, err)
	}
	return nil
}

type Backends []Backend

func (b Backends) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	for _, a := range b {
		err = a.Apply(ctx, clientset)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b Backends) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	for _, a := range b {
		err = a.Delete(ctx, clientset)
		if err != nil {
			return err
		}
	}
	return nil
}

type Backend struct {
	Service   *corev1.Service
	Endpoints *corev1.Endpoints
}

func (b Backend) Apply(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	_, err = clientset.CoreV1().
		Services(b.Service.Namespace).
		Create(ctx, b.Service, metav1.CreateOptions{
			FieldManager: ferry,
		})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			b.Service.ResourceVersion = "0"
			_, err = clientset.CoreV1().Services(b.Service.Namespace).
				Update(ctx, b.Service, metav1.UpdateOptions{
					FieldManager: ferry,
				})
			if err != nil {
				return fmt.Errorf("update service %s.%s: %w", b.Service.Name, b.Service.Namespace, err)
			}
		} else {
			return fmt.Errorf("create service %s.%s: %w", b.Service.Name, b.Service.Namespace, err)
		}
	}

	_, err = clientset.CoreV1().
		Endpoints(b.Endpoints.Namespace).
		Create(ctx, b.Endpoints, metav1.CreateOptions{
			FieldManager: ferry,
		})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			b.Endpoints.ResourceVersion = "0"
			_, err = clientset.CoreV1().Endpoints(b.Endpoints.Namespace).
				Update(ctx, b.Endpoints, metav1.UpdateOptions{
					FieldManager: ferry,
				})
			if err != nil {
				return fmt.Errorf("update endpoints %s.%s: %w", b.Endpoints.Name, b.Endpoints.Namespace, err)
			}
		} else {
			return fmt.Errorf("create endpoints %s.%s: %w", b.Endpoints.Name, b.Endpoints.Namespace, err)
		}
	}
	return nil
}

func (b Backend) Delete(ctx context.Context, clientset *kubernetes.Clientset) (err error) {
	err = clientset.CoreV1().
		Endpoints(b.Service.Namespace).
		Delete(ctx, b.Endpoints.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete endpoints %s.%s: %w", b.Endpoints.Name, b.Endpoints.Namespace, err)
	}
	err = clientset.CoreV1().
		Services(b.Service.Namespace).
		Delete(ctx, b.Service.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("delete service %s.%s: %w", b.Endpoints.Name, b.Endpoints.Namespace, err)
	}
	return nil
}

func BuildBackend(proxy *Proxy, name, ns string, ips []string, srcPort, destPort int32) Backend {
	meta := metav1.ObjectMeta{
		Name:      name,
		Namespace: ns,
		Labels:    proxy.Labels,
	}

	return Backend{
		Service: &corev1.Service{
			ObjectMeta: meta,
			Spec: corev1.ServiceSpec{
				// ClusterIP: corev1.ClusterIPNone,
				Ports: []corev1.ServicePort{
					{
						Port: srcPort,
						//	TargetPort: intstr.FromInt(int(destPort)),
					},
				},
			},
		},
		Endpoints: &corev1.Endpoints{
			ObjectMeta: meta,
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: buildIPToEndpointAddress(ips),
					Ports: []corev1.EndpointPort{
						{
							Port: destPort,
						},
					},
				},
			},
		},
	}
}

func buildIPToEndpointAddress(ips []string) []corev1.EndpointAddress {
	eps := []corev1.EndpointAddress{}
	for _, ip := range ips {
		eps = append(eps, corev1.EndpointAddress{
			IP: ip,
		})
	}
	return eps
}
