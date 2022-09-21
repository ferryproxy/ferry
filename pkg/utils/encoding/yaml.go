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

	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

func MarshalYAML(objs ...objref.KMetadata) ([]byte, error) {
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
