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
	"reflect"
	"testing"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
)

func TestMarshalJSON(t *testing.T) {
	type args struct {
		objs []objref.KMetadata
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			args: args{
				objs: []objref.KMetadata{
					&corev1.Pod{},
					&v1alpha2.Hub{},
				},
			},
			want: []byte(`{"kind":"Pod","apiVersion":"v1","metadata":{"creationTimestamp":null},"spec":{"containers":null},"status":{}}
{"kind":"Hub","apiVersion":"traffic.ferryproxy.io/v1alpha2","metadata":{"creationTimestamp":null},"spec":{"gateway":{"reachable":false}},"status":{"lastSynchronizationTimestamp":null}}
`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalJSON(tt.args.objs...)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %s, want %s", got, tt.want)
			}
		})
	}
}
