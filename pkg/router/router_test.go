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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/wzshiming/sshproxy/permissions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name string
		args fakeRouter
		want map[string][]resource.Resourcer
	}{
		{
			name: "export reachable",
			args: fakeRouter{
				Services: []*corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:     "http",
									Port:     80,
									Protocol: corev1.ProtocolTCP,
								},
							},
						},
					},
				},
				Hubs: []*v1alpha2.Hub{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "export",
						},
						Spec: v1alpha2.HubSpec{
							Gateway: v1alpha2.HubSpecGateway{
								Reachable: true,
								Address:   "10.0.0.1:8080",
								ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://reception2",
									},
									{
										Proxy: "socks5://reception1",
									},
								},
								NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://navigation2",
									},
									{
										Proxy: "socks5://navigation1",
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "import",
						},
						Spec: v1alpha2.HubSpec{
							Gateway: v1alpha2.HubSpecGateway{
								Reachable: false,
								Address:   "10.0.0.2:8080",
							},
						},
					},
				},
				Routes: []*v1alpha2.Route{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
						},
						Spec: v1alpha2.RouteSpec{
							Import: v1alpha2.RouteSpecRule{
								HubName: "export",
								Service: v1alpha2.RouteSpecRuleService{
									Name:      "svc1",
									Namespace: "test",
								},
							},
							Export: v1alpha2.RouteSpecRule{
								HubName: "export",
								Service: v1alpha2.RouteSpecRuleService{
									Name:      "svc1",
									Namespace: "test",
								},
							},
						},
					},
				},
			},

			want: map[string][]resource.Resourcer{
				"export": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "import-authorized",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "authorized",
								},
							},
							Data: map[string]string{
								"authorized_keys": "import-authorized import@ferryproxy.io",
								"user":            "import",
							},
						},
					},
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-allows-80-10001",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "allows",
								},
							},
							Data: map[string]string{
								consts.TunnelRulesAllowKey: toJson(
									map[string]AllowList{
										"import": {
											DirectTcpip: permissions.Permission{
												Allows: []string{
													"svc1.test.svc:80",
												},
											},
										},
									},
								),
							},
						},
					},
				},
				"import": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-service",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "service",
								},
							},
							Data: map[string]string{
								"export_hub_name":          "export",
								"export_service_name":      "svc1",
								"export_service_namespace": "test",
								"import_service_name":      "svc1",
								"import_service_namespace": "test",
								"ports":                    `[{"name":"http","protocol":"TCP","port":80,"targetPort":10001}]`,
							},
						},
					},
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-tunnel-80-10001",
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
												":10001",
											},
											Proxy: []string{
												"svc1.test.svc:80",
												"ssh://import@10.0.0.1:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
												"socks5://reception2",
												"socks5://reception1",
											},
										},
									},
								),
							},
						},
					},
				},
			},
		},
		{
			name: "import reachable",
			args: fakeRouter{
				Services: []*corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:     "http",
									Port:     80,
									Protocol: corev1.ProtocolTCP,
								},
							},
						},
					},
				},
				Hubs: []*v1alpha2.Hub{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "export",
						},
						Spec: v1alpha2.HubSpec{
							Gateway: v1alpha2.HubSpecGateway{
								Reachable: false,
								Address:   "10.0.0.1:8080",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "import",
						},
						Spec: v1alpha2.HubSpec{
							Gateway: v1alpha2.HubSpecGateway{
								Reachable: true,
								Address:   "10.0.0.2:8080",
								ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://reception2",
									},
									{
										Proxy: "socks5://reception1",
									},
								},
								NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://navigation2",
									},
									{
										Proxy: "socks5://navigation1",
									},
								},
							},
						},
					},
				},
				Routes: []*v1alpha2.Route{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
						},
						Spec: v1alpha2.RouteSpec{
							Import: v1alpha2.RouteSpecRule{
								HubName: "export",
								Service: v1alpha2.RouteSpecRuleService{
									Name:      "svc1",
									Namespace: "test",
								},
							},
							Export: v1alpha2.RouteSpecRule{
								HubName: "export",
								Service: v1alpha2.RouteSpecRuleService{
									Name:      "svc1",
									Namespace: "test",
								},
							},
						},
					},
				},
			},
			want: map[string][]resource.Resourcer{
				"import": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "export-authorized",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "authorized",
								},
							},
							Data: map[string]string{
								"authorized_keys": "export-authorized export@ferryproxy.io",
								"user":            "export",
							},
						},
					},
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-allows-80-10001",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "allows",
								},
							},
							Data: map[string]string{
								consts.TunnelRulesAllowKey: toJson(
									map[string]AllowList{
										"export": {
											TcpipForward: permissions.Permission{
												Allows: []string{
													":10001",
												},
											},
										},
									},
								),
							},
						},
					},
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-service",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "service",
								},
							},
							Data: map[string]string{
								"export_hub_name":          "export",
								"export_service_name":      "svc1",
								"export_service_namespace": "test",
								"import_service_name":      "svc1",
								"import_service_namespace": "test",
								"ports":                    `[{"name":"http","protocol":"TCP","port":80,"targetPort":10001}]`,
							},
						},
					},
				},
				"export": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-tunnel-80-10001",
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
												":10001",
												"ssh://export@10.0.0.2:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
												"socks5://reception2",
												"socks5://reception1",
											},
											Proxy: []string{
												"svc1.test.svc:80",
											},
										},
									},
								),
							},
						},
					},
				},
			},
		},
		{
			name: "proxy reachable",
			args: fakeRouter{
				Services: []*corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:     "http",
									Port:     80,
									Protocol: corev1.ProtocolTCP,
								},
							},
						},
					},
				},
				Hubs: []*v1alpha2.Hub{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "export",
						},
						Spec: v1alpha2.HubSpec{
							Gateway: v1alpha2.HubSpecGateway{
								Reachable: false,
								Address:   "10.0.0.1:8080",
								NavigationWay: []v1alpha2.HubSpecGatewayWay{
									{
										HubName: "import",
									},
									{
										HubName: "proxy",
									},
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "import",
						},
						Spec: v1alpha2.HubSpec{
							Gateway: v1alpha2.HubSpecGateway{
								Reachable: false,
								Address:   "10.0.0.2:8080",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "proxy",
						},
						Spec: v1alpha2.HubSpec{
							Gateway: v1alpha2.HubSpecGateway{
								Reachable: true,
								Address:   "10.0.0.3:8080",
							},
						},
					},
				},
				Routes: []*v1alpha2.Route{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
						},
						Spec: v1alpha2.RouteSpec{
							Import: v1alpha2.RouteSpecRule{
								HubName: "export",
								Service: v1alpha2.RouteSpecRuleService{
									Name:      "svc1",
									Namespace: "test",
								},
							},
							Export: v1alpha2.RouteSpecRule{
								HubName: "export",
								Service: v1alpha2.RouteSpecRuleService{
									Name:      "svc1",
									Namespace: "test",
								},
							},
						},
					},
				},
			},
			want: map[string][]resource.Resourcer{
				"export": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-tunnel-80-10001",
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
												"unix:///dev/shm/svc1-tunnel-80-10001.socks",
												"ssh://export@10.0.0.3:8080?identity_file=/var/ferry/ssh/identity&target_hub=proxy",
											},
											Proxy: []string{
												"svc1.test.svc:80",
											},
										},
									},
								),
							},
						},
					},
				},
				"import": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-service",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "service",
								},
							},
							Data: map[string]string{
								"export_hub_name":          "export",
								"export_service_name":      "svc1",
								"export_service_namespace": "test",
								"import_service_name":      "svc1",
								"import_service_namespace": "test",
								"ports":                    `[{"name":"http","protocol":"TCP","port":80,"targetPort":10001}]`,
							},
						},
					},
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-tunnel-80-10001",
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
												":10001",
											},
											Proxy: []string{
												"unix:///dev/shm/svc1-tunnel-80-10001.socks",
												"ssh://import@10.0.0.3:8080?identity_file=/var/ferry/ssh/identity&target_hub=proxy",
											},
										},
									},
								),
							},
						},
					},
				},
				"proxy": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "export-authorized",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "authorized",
								},
							},
							Data: map[string]string{
								"authorized_keys": "export-authorized export@ferryproxy.io",
								"user":            "export",
							},
						},
					},
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "import-authorized",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "authorized",
								},
							},
							Data: map[string]string{
								"authorized_keys": "import-authorized import@ferryproxy.io",
								"user":            "import",
							},
						},
					},
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1-allows-80-10001",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"tunnel.ferryproxy.io/config": "allows",
								},
							},
							Data: map[string]string{
								consts.TunnelRulesAllowKey: toJson(
									map[string]AllowList{
										"export": {
											StreamlocalForward: permissions.Permission{
												Allows: []string{
													"/dev/shm/svc1-tunnel-80-10001.socks",
												},
											},
										},
										"import": {
											DirectStreamlocal: permissions.Permission{
												Allows: []string{
													"/dev/shm/svc1-tunnel-80-10001.socks",
												},
											},
										},
									},
								),
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.BuildResource()
			if err != nil {
				t.Errorf("BuildResource() error = %v", err)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("BuildResource(): got - want + \n%s", diff)
			}

		})
	}
}

func toJson(c interface{}) string {
	data, _ := json.Marshal(c)
	return string(data)
}

type fakeRouter struct {
	Services []*corev1.Service
	Hubs     []*v1alpha2.Hub
	Routes   []*v1alpha2.Route
}

func (f *fakeRouter) BuildResource() (out map[string][]resource.Resourcer, err error) {
	hubs := map[string]*v1alpha2.Hub{}
	for _, hub := range f.Hubs {
		hubs[hub.Name] = hub
	}

	fake := &fakeHubInterface{
		services:  f.Services,
		hubs:      hubs,
		port:      10000,
		portCache: map[string]int{},
	}

	exportHubName := "export"
	importHubName := "import"

	solution := Solution{
		getHubGateway: fake.GetHubGateway,
	}

	ways, err := solution.CalculateWays(exportHubName, importHubName)
	if err != nil {
		return nil, err
	}

	router := NewRouter(RouterConfig{
		Labels:        map[string]string{},
		ExportHubName: exportHubName,
		ImportHubName: importHubName,
		HubInterface:  fake,
	})

	return router.BuildResource(f.Routes, ways)
}

type fakeHubInterface struct {
	services  []*corev1.Service
	hubs      map[string]*v1alpha2.Hub
	portCache map[string]int
	port      int
}

func (f *fakeHubInterface) ListServices(name string) []*corev1.Service {
	return f.services
}

func (f *fakeHubInterface) GetHub(name string) *v1alpha2.Hub {
	return f.hubs[name]
}

func (f *fakeHubInterface) GetHubGateway(hubName string, forHub string) v1alpha2.HubSpecGateway {
	hub := f.hubs[hubName]
	if hub != nil {
		if hub.Spec.Override != nil {
			h, ok := hub.Spec.Override[forHub]
			if ok {
				return h
			}
		}
		return hub.Spec.Gateway
	}
	return v1alpha2.HubSpecGateway{}
}

func (f fakeHubInterface) GetAuthorized(name string) string {
	return fmt.Sprintf("%s-%s", name, "authorized")
}

func (f *fakeHubInterface) GetPortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error) {
	key := fmt.Sprintf("%s-%s-%s-%s-%d", importHubName, cluster, namespace, name, port)
	v, ok := f.portCache[key]
	if ok {
		return int32(v), nil
	}
	f.port++
	f.portCache[key] = f.port
	return int32(f.port), nil
}
