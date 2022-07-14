package router

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

			want: map[string][]resource.Resourcer{
				"import": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "import-test-svc1-80-export-test-svc1-10001-tunnel",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"traffic.ferryproxy.io/exported-from-name":      "svc1",
									"traffic.ferryproxy.io/exported-from-namespace": "test",
									"traffic.ferryproxy.io/exported-from-ports":     "10001",
									"tunnel.ferryproxy.io/service":                  "inject",
								},
							},
							Data: map[string]string{
								"tunnel": toJson(
									[]Chain{
										{
											Bind: []string{
												"0.0.0.0:10001",
											},
											Proxy: []string{
												"svc1.test.svc:80",
												"ssh://10.0.0.1:8080?identity_data=export-identity",
												"socks5://reception2",
												"socks5://reception1",
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
									"traffic.ferryproxy.io/exported-from-name":      "svc1",
									"traffic.ferryproxy.io/exported-from-namespace": "test",
									"traffic.ferryproxy.io/exported-from-ports":     "10001",
									"tunnel.ferryproxy.io/service":                  "inject",
								},
							},
							Spec: corev1.ServiceSpec{
								Ports: []corev1.ServicePort{
									{
										Name:       "http",
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
			want: map[string][]resource.Resourcer{
				"import": {
					resource.Service{
						Service: &corev1.Service{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc1",
								Namespace: "test",
								Labels: map[string]string{
									"traffic.ferryproxy.io/exported-from-name":      "svc1",
									"traffic.ferryproxy.io/exported-from-namespace": "test",
									"traffic.ferryproxy.io/exported-from-ports":     "10001",
									"tunnel.ferryproxy.io/service":                  "inject",
								},
							},
							Spec: corev1.ServiceSpec{
								Ports: []corev1.ServicePort{
									{
										Name:       "http",
										Protocol:   "TCP",
										Port:       80,
										TargetPort: intstr.IntOrString{IntVal: 10001},
									},
								},
							},
						},
					},
				},
				"export": {
					resource.ConfigMap{
						ConfigMap: &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "import-test-svc1-80-export-test-svc1-10001-tunnel",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"traffic.ferryproxy.io/exported-from-name":      "svc1",
									"traffic.ferryproxy.io/exported-from-namespace": "test",
									"traffic.ferryproxy.io/exported-from-ports":     "10001",
									"tunnel.ferryproxy.io/service":                  "inject",
								},
							},
							Data: map[string]string{
								"tunnel": toJson(
									[]Chain{
										{
											Bind: []string{
												"0.0.0.0:10001",
												"ssh://10.0.0.2:8080?identity_data=import-identity",
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
								Navigation: v1alpha2.HubSpecGatewayWays{
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
								Name:      "import-test-svc1-80-export-test-svc1-10001-tunnel",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"traffic.ferryproxy.io/exported-from-name":      "svc1",
									"traffic.ferryproxy.io/exported-from-namespace": "test",
									"traffic.ferryproxy.io/exported-from-ports":     "10001",
									"tunnel.ferryproxy.io/service":                  "inject",
								},
							},
							Data: map[string]string{
								"tunnel": toJson(
									[]Chain{
										{
											Bind: []string{
												"unix:///dev/shm/import-test-svc1-80-export-test-svc1-10001-tunnel.socks",
												"ssh://10.0.0.3:8080?identity_data=proxy-identity",
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
								Name:      "import-test-svc1-80-export-test-svc1-10001-tunnel",
								Namespace: "ferry-tunnel-system",
								Labels: map[string]string{
									"traffic.ferryproxy.io/exported-from-name":      "svc1",
									"traffic.ferryproxy.io/exported-from-namespace": "test",
									"traffic.ferryproxy.io/exported-from-ports":     "10001",
									"tunnel.ferryproxy.io/service":                  "inject",
								},
							},
							Data: map[string]string{
								"tunnel": toJson(
									[]Chain{
										{
											Bind: []string{
												"0.0.0.0:10001",
											},
											Proxy: []string{
												"unix:///dev/shm/import-test-svc1-80-export-test-svc1-10001-tunnel.socks",
												"ssh://10.0.0.3:8080?identity_data=proxy-identity",
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
									"traffic.ferryproxy.io/exported-from-name":      "svc1",
									"traffic.ferryproxy.io/exported-from-namespace": "test",
									"traffic.ferryproxy.io/exported-from-ports":     "10001",
									"tunnel.ferryproxy.io/service":                  "inject",
								},
							},
							Spec: corev1.ServiceSpec{
								Ports: []corev1.ServicePort{
									{
										Name:       "http",
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

func toJson(c []Chain) string {
	data, _ := json.MarshalIndent(c, "", "  ")
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

	fakeCache := &fakeClusterCache{
		services:  f.Services,
		hubs:      hubs,
		port:      10000,
		portCache: map[string]int{},
	}

	exportHubName := "export"
	importHubName := "import"

	solution := Solution{
		getHubGateway: fakeCache.GetHubGateway,
	}

	ways, err := solution.CalculateWays(exportHubName, importHubName)
	if err != nil {
		return nil, err
	}

	router := NewRouter(RouterConfig{
		Namespace:     consts.FerryTunnelNamespace,
		Labels:        map[string]string{},
		ExportHubName: exportHubName,
		ImportHubName: importHubName,
		GetIdentity:   fakeCache.GetIdentity,
		ListServices:  fakeCache.ListServices,
		GetHubGateway: fakeCache.GetHubGateway,
		GetPortPeer:   fakeCache.GetPortPeer,
	})

	router.SetRoutes(f.Routes)

	return router.BuildResource(ways)
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

func (f *fakeClusterCache) GetHubGateway(hubName string, forHub string) v1alpha2.HubSpecGateway {
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
