/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clientset

import (
	"context"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type Interface[T Object, L List] interface {
	// Get takes name of the resource, and returns the corresponding resource object, and an error if there is any.
	Get(ctx context.Context, name string, options metav1.GetOptions) (result T, err error)

	// List takes label and field selectors, and returns the list of resource that match those selectors.
	List(ctx context.Context, opts metav1.ListOptions) (result L, err error)

	// Watch returns a watch.Interface that watches the requested Clientset.
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)

	// Create takes the representation of a resource and creates it.  Returns the server's representation of the resource, and an error, if there is any.
	Create(ctx context.Context, cr T, opts metav1.CreateOptions) (result T, err error)

	// Update takes the representation of a resource and updates it. Returns the server's representation of the resource, and an error, if there is any.
	Update(ctx context.Context, cr T, opts metav1.UpdateOptions) (result T, err error)

	// UpdateStatus was generated because the type contains a Status member.
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, cr T, opts metav1.UpdateOptions) (result T, err error)

	// Delete takes name of the resource and deletes it. Returns an error if one occurs.
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error

	// DeleteCollection deletes a collection of objects.
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error

	// Patch applies the patch and returns the patched resource.
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result T, err error)

	Expansion[T, L]
}

type Object interface {
	runtime.Object
	metav1.Object
}

type List interface {
	runtime.Object
	metav1.ListMetaAccessor
}

// Clientset implements Clientset[Object, List]
type Clientset[T Object, L List] struct {
	scheme         *runtime.Scheme
	resource       string
	parameterCodec runtime.ParameterCodec
	client         rest.Interface
	ns             string

	tType reflect.Type
	lType reflect.Type
}

// NewClientset returns a Interface[T Object, L List]
func NewClientset[T Object, L List](scheme *runtime.Scheme, restConfig *rest.Config, resource, namespace string) (Interface[T, L], error) {
	httpClient, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return nil, err
	}
	codecs := serializer.NewCodecFactory(scheme)
	restConfig.NegotiatedSerializer = codecs.WithoutConversion()

	client, err := rest.RESTClientForConfigAndClient(restConfig, httpClient)
	if err != nil {
		return nil, err
	}
	return newClientset[T, L](scheme, client, resource, namespace), nil
}

// newClientset returns a Interface[T Object, L List]
func newClientset[T Object, L List](scheme *runtime.Scheme, client rest.Interface, resource, namespace string) Interface[T, L] {
	var (
		t T
		l L
	)
	parameterCodec := runtime.NewParameterCodec(scheme)

	return &Clientset[T, L]{
		scheme:         scheme,
		parameterCodec: parameterCodec,
		client:         client,
		resource:       resource,
		ns:             namespace,
		tType:          reflect.TypeOf(t).Elem(),
		lType:          reflect.TypeOf(l).Elem(),
	}
}

func (c *Clientset[T, L]) newT() (result T) {
	return reflect.New(c.tType).Interface().(T)
}

func (c *Clientset[T, L]) newL() (result L) {
	return reflect.New(c.lType).Interface().(L)
}

// Get takes name of the resource, and returns the corresponding resource object, and an error if there is any.
func (c *Clientset[T, L]) Get(ctx context.Context, name string, options metav1.GetOptions) (result T, err error) {
	result = c.newT()
	err = c.client.Get().
		Namespace(c.ns).
		Resource(c.resource).
		Name(name).
		VersionedParams(&options, c.parameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of resource that match those selectors.
func (c *Clientset[T, L]) List(ctx context.Context, opts metav1.ListOptions) (result L, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = c.newL()
	err = c.client.Get().
		Namespace(c.ns).
		Resource(c.resource).
		VersionedParams(&opts, c.parameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested Clientset.
func (c *Clientset[T, L]) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource(c.resource).
		VersionedParams(&opts, c.parameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a resource and creates it.  Returns the server's representation of the resource, and an error, if there is any.
func (c *Clientset[T, L]) Create(ctx context.Context, cr T, opts metav1.CreateOptions) (result T, err error) {
	result = c.newT()
	err = c.client.Post().
		Namespace(c.ns).
		Resource(c.resource).
		VersionedParams(&opts, c.parameterCodec).
		Body(cr).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a resource and updates it. Returns the server's representation of the resource, and an error, if there is any.
func (c *Clientset[T, L]) Update(ctx context.Context, cr T, opts metav1.UpdateOptions) (result T, err error) {
	result = c.newT()
	err = c.client.Put().
		Namespace(c.ns).
		Resource(c.resource).
		Name(cr.GetName()).
		VersionedParams(&opts, c.parameterCodec).
		Body(cr).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *Clientset[T, L]) UpdateStatus(ctx context.Context, cr T, opts metav1.UpdateOptions) (result T, err error) {
	result = c.newT()
	err = c.client.Put().
		Namespace(c.ns).
		Resource(c.resource).
		Name(cr.GetName()).
		SubResource("status").
		VersionedParams(&opts, c.parameterCodec).
		Body(cr).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the resource and deletes it. Returns an error if one occurs.
func (c *Clientset[T, L]) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource(c.resource).
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *Clientset[T, L]) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource(c.resource).
		VersionedParams(&listOpts, c.parameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched resource.
func (c *Clientset[T, L]) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result T, err error) {
	result = c.newT()
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource(c.resource).
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, c.parameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
