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

package router

import (
	"testing"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/google/go-cmp/cmp"
	"github.com/wzshiming/sshproxy/permissions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestManualBuildResource(t *testing.T) {
	tests := []struct {
		name    string
		args    ManualConfig
		wantOut map[string][]objref.KMetadata
		wantErr bool
	}{
		{
			args: ManualConfig{
				RouteName:        "manual",
				ExportHubName:    "export-hub",
				ExportName:       "export-name",
				ExportNamespace:  "export-namespace",
				ExportAuthorized: "export-authorized",
				ExportGateway: v1alpha2.HubSpecGateway{
					Address:   "export-address",
					Reachable: true,
				},
				ImportHubName:    "import-hub",
				ImportName:       "import-name",
				ImportNamespace:  "import-namespace",
				ImportAuthorized: "import-authorized",
				ImportGateway: v1alpha2.HubSpecGateway{
					Address:   "",
					Reachable: false,
				},
				Port:     80,
				BindPort: 10000,
			},
			wantOut: map[string][]objref.KMetadata{
				"export-hub": {
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "import-hub-authorized",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"tunnel.ferryproxy.io/config": "authorized",
							},
						},
						Data: map[string]string{
							"authorized_keys": "import-authorized import-hub@ferryproxy.io",
							"user":            "import-hub",
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "manual-allows-80-10000",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"tunnel.ferryproxy.io/config": "allows",
							},
						},
						Data: map[string]string{
							consts.TunnelRulesAllowKey: toJson(
								map[string]AllowList{
									"import-hub": {
										DirectTcpip: permissions.Permission{
											Allows: []string{
												"export-name.export-namespace.svc:80",
											},
										},
									},
								},
							),
						},
					},
				},
				"import-hub": {
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "manual-service",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"tunnel.ferryproxy.io/config": "service",
							},
						},
						Data: map[string]string{
							"export_hub_name":          "export-hub",
							"export_service_name":      "export-name",
							"export_service_namespace": "export-namespace",
							"import_service_name":      "import-name",
							"import_service_namespace": "import-namespace",
							"ports":                    `[{"protocol":"TCP","port":80,"targetPort":10000}]`,
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "manual-tunnel-80-10000",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"tunnel.ferryproxy.io/config": "rules",
							},
						},
						Data: map[string]string{
							consts.TunnelRulesKey: toJson(
								[]Chain{
									{
										Bind: []string{
											":10000",
										},
										Proxy: []string{
											"export-name.export-namespace.svc:80",
											"ssh://import-hub@export-address?identity_file=/var/ferry/ssh/identity&target_hub=export-hub",
										},
									},
								},
							),
						},
					},
				},
			},
		},
		{
			args: ManualConfig{
				RouteName:        "manual",
				ExportHubName:    "export-hub",
				ExportName:       "export-name",
				ExportNamespace:  "export-namespace",
				ExportAuthorized: "export-authorized",
				ExportGateway: v1alpha2.HubSpecGateway{
					Address:   "",
					Reachable: false,
				},
				ImportHubName:    "import-hub",
				ImportName:       "import-name",
				ImportNamespace:  "import-namespace",
				ImportAuthorized: "import-authorized",
				ImportGateway: v1alpha2.HubSpecGateway{
					Address:   "import-address",
					Reachable: true,
				},
				Port:     80,
				BindPort: 10000,
			},
			wantOut: map[string][]objref.KMetadata{
				"export-hub": {
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "manual-tunnel-80-10000",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"tunnel.ferryproxy.io/config": "rules",
							},
						},
						Data: map[string]string{
							consts.TunnelRulesKey: toJson(
								[]Chain{
									{
										Bind: []string{
											":10000",
											"ssh://export-hub@import-address?identity_file=/var/ferry/ssh/identity&target_hub=import-hub",
										},
										Proxy: []string{
											"export-name.export-namespace.svc:80",
										},
									},
								},
							),
						},
					},
				},
				"import-hub": {
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "export-hub-authorized",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"tunnel.ferryproxy.io/config": "authorized",
							},
						},
						Data: map[string]string{
							"authorized_keys": "export-authorized export-hub@ferryproxy.io",
							"user":            "export-hub",
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "manual-allows-80-10000",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"tunnel.ferryproxy.io/config": "allows",
							},
						},
						Data: map[string]string{
							consts.TunnelRulesAllowKey: toJson(
								map[string]AllowList{
									"export-hub": {
										TcpipForward: permissions.Permission{
											Allows: []string{
												":10000",
											},
										},
									},
								},
							),
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "manual-service",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"tunnel.ferryproxy.io/config": "service",
							},
						},
						Data: map[string]string{
							"export_hub_name":          "export-hub",
							"export_service_name":      "export-name",
							"export_service_namespace": "export-namespace",
							"import_service_name":      "import-name",
							"import_service_namespace": "import-namespace",
							"ports":                    `[{"protocol":"TCP","port":80,"targetPort":10000}]`,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewManual(tt.args)
			gotOut, err := f.BuildResource()
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(gotOut, tt.wantOut); diff != "" {
				t.Errorf("BuildResource(): got - want + \n%s", diff)
			}
		})
	}
}
