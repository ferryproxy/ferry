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

package encoding

import (
	"bytes"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/yaml"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(v1alpha2.AddToScheme(scheme))
}

func MarshalYAML(objs ...runtime.Object) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	for i, obj := range objs {
		if i != 0 {
			buf.Write([]byte("---\n"))
		}
		gvks, _, err := scheme.ObjectKinds(obj)
		if err != nil {
			return nil, err
		}
		b, ok := obj.(interface {
			SetGroupVersionKind(gvk schema.GroupVersionKind)
		})
		if ok {
			b.SetGroupVersionKind(gvks[0])
		}
		data, err := yaml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	return buf.Bytes(), nil
}
