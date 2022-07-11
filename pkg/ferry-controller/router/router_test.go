package router

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ferry-proxy/api/apis/traffic/v1alpha2"
	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferry-proxy/ferry/pkg/ferry-controller/router/tunnel"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name                string
		args                fakeRouter
		wantIngressResource []resource.Resourcer
		wantEgressResource  []resource.Resourcer
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
								Reception: []v1alpha2.HubSpecGatewayWay{
									{
										Proxy: "socks5://reception2",
									},
									{
										Proxy: "socks5://reception1",
									},
								},
								Navigation: []v1alpha2.HubSpecGatewayWay{
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

			wantIngressResource: []resource.Resourcer{
				resource.ConfigMap{
					ConfigMap: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "import-test-svc1-export-test-svc1-tunnel-server",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},
					},
				},
			},
			wantEgressResource: []resource.Resourcer{
				resource.ConfigMap{
					ConfigMap: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "import-test-svc1-export-test-svc1-tunnel-client",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},
						Data: map[string]string{
							"80": toJson(
								[]tunnel.Chain{
									{
										Bind: []string{
											"0.0.0.0:10001",
										},
										Proxy: []string{
											"svc1.test.svc:80",
											"ssh://10.0.0.1:8080?identity_data=export-identity",
											"socks5://reception2",
											"socks5://reception1",
											"socks5://navigation2",
											"socks5://navigation1",
										},
									},
								},
							),
						},
					},
				},
				resource.Service{
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"traffic.ferry.zsm.io/exported-from-ports":     "10001",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "svc1-test-80-10001",
									Protocol:   "TCP",
									Port:       80,
									TargetPort: intstr.IntOrString{IntVal: 10001},
								},
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
								Reception: []v1alpha2.HubSpecGatewayWay{
									{
										Proxy: "socks5://reception2",
									},
									{
										Proxy: "socks5://reception1",
									},
								},
								Navigation: []v1alpha2.HubSpecGatewayWay{
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

			wantIngressResource: []resource.Resourcer{
				resource.ConfigMap{
					ConfigMap: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "import-test-svc1-export-test-svc1-tunnel-client",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},
						Data: map[string]string{
							"80": toJson(
								[]tunnel.Chain{
									{
										Bind: []string{
											"0.0.0.0:10001",
											"ssh://10.0.0.2:8080?identity_data=import-identity",
											"socks5://reception2",
											"socks5://reception1",
											"socks5://navigation2",
											"socks5://navigation1",
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
			wantEgressResource: []resource.Resourcer{
				resource.ConfigMap{
					ConfigMap: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "import-test-svc1-export-test-svc1-tunnel-server",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},

						Data: nil,
					},
				},
				resource.Service{
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"traffic.ferry.zsm.io/exported-from-ports":     "10001",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "svc1-test-80-10001",
									Protocol:   "TCP",
									Port:       80,
									TargetPort: intstr.IntOrString{IntVal: 10001},
								},
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
								Reception: v1alpha2.HubSpecGatewayWays{
									{
										HubName: "proxy",
									},
									{
										Proxy: "socks5://export-reception2",
									},
									{
										Proxy: "socks5://export-reception1",
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
								Navigation: []v1alpha2.HubSpecGatewayWay{
									{
										HubName: "proxy",
									},
									{
										Proxy: "socks5://import-navigation2",
									},
									{
										Proxy: "socks5://import-navigation1",
									},
								},
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

			wantIngressResource: []resource.Resourcer{
				resource.ConfigMap{
					ConfigMap: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "import-test-svc1-export-test-svc1-tunnel-server",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},
						Data: map[string]string{
							"80": toJson(
								[]tunnel.Chain{
									{
										Bind: []string{
											"unix:///dev/shm/import-test-svc1-export-test-svc1-80-10001-tunnel.socks",
											"ssh://10.0.0.3:8080?identity_data=proxy-identity",
											"socks5://import-navigation2",
											"socks5://import-navigation1",
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
			wantEgressResource: []resource.Resourcer{
				resource.ConfigMap{
					ConfigMap: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "import-test-svc1-export-test-svc1-tunnel-client",
							Namespace: "ferry-tunnel-system",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},
						Data: map[string]string{
							"80": toJson(
								[]tunnel.Chain{
									{
										Bind: []string{
											"0.0.0.0:10001",
										},
										Proxy: []string{
											"unix:///dev/shm/import-test-svc1-export-test-svc1-80-10001-tunnel.socks",
											"ssh://10.0.0.3:8080?identity_data=proxy-identity",
											"socks5://export-reception2",
											"socks5://export-reception1",
										},
									},
								},
							),
						},
					},
				},
				resource.Service{
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "svc1",
							Namespace: "test",
							Labels: map[string]string{
								"traffic.ferry.zsm.io/exported-from-name":      "svc1",
								"traffic.ferry.zsm.io/exported-from-namespace": "test",
								"traffic.ferry.zsm.io/exported-from-ports":     "10001",
								"tunnel.ferry.zsm.io/service":                  "inject",
							},
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "svc1-test-80-10001",
									Protocol:   "TCP",
									Port:       80,
									TargetPort: intstr.IntOrString{IntVal: 10001},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIngressResource, gotEgressResource, err := tt.args.BuildResource()
			if err != nil {
				t.Errorf("BuildResource() error = %v", err)
				return
			}
			if diff := cmp.Diff(gotIngressResource, tt.wantIngressResource); diff != "" {
				t.Errorf("BuildResource() IngressResource: got - want + \n%s", diff)
			}

			if diff := cmp.Diff(gotEgressResource, tt.wantEgressResource); diff != "" {
				t.Errorf("BuildResource() EgressResource: got - want + \n%s", diff)
			}
		})
	}
}

func toJson(c []tunnel.Chain) string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

type fakeRouter struct {
	Services []*corev1.Service
	Hubs     []*v1alpha2.Hub
	Routes   []*v1alpha2.Route
}

func (f *fakeRouter) BuildResource() (ir, er []resource.Resourcer, err error) {
	hubs := map[string]*v1alpha2.Hub{}
	for _, hub := range f.Hubs {
		hubs[hub.Name] = hub
	}
	router := NewRouter(RouterConfig{
		Namespace:     consts.FerryTunnelNamespace,
		Labels:        map[string]string{},
		ExportHubName: "export",
		ImportHubName: "import",
		ClusterCache: &fakeClusterCache{
			services:  f.Services,
			hubs:      hubs,
			port:      10000,
			portCache: map[string]int{},
		},
	})

	router.SetRoutes(f.Routes)

	return router.BuildResource()
}

type fakeClusterCache struct {
	services  []*corev1.Service
	hubs      map[string]*v1alpha2.Hub
	portCache map[string]int
	port      int
}

func (f *fakeClusterCache) ListServices(name string) []*corev1.Service {
	return f.services
}

func (f *fakeClusterCache) GetHub(name string) *v1alpha2.Hub {
	return f.hubs[name]
}

func (f fakeClusterCache) GetIdentity(name string) string {
	return fmt.Sprintf("%s-%s", name, "identity")
}

func (f *fakeClusterCache) GetPortPeer(importHubName string, cluster, namespace, name string, port int32) int32 {
	key := fmt.Sprintf("%s-%s-%s-%s-%d", importHubName, cluster, namespace, name, port)
	v, ok := f.portCache[key]
	if ok {
		return int32(v)
	}
	f.port++
	f.portCache[key] = f.port
	return int32(f.port)
}
