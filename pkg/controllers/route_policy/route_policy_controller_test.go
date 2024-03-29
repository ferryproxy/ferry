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

package route_policy

import (
	"testing"

	trafficv1alpha2 "github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_policiesToRoutes(t *testing.T) {
	controller := true
	ownerReferences := []metav1.OwnerReference{
		{
			APIVersion: "traffic.ferryproxy.io/v1alpha2",
			Kind:       "RoutePolicy",
			Name:       "test",
			Controller: &controller,
		},
	}
	src := &fakeDataSource{
		svcs: map[string][]*corev1.Service{
			"export-1": {
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-1",
						Namespace: "default",
						Labels: map[string]string{
							"app": "app-1",
							"hub": "export-1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-2",
						Namespace: "default",
						Labels: map[string]string{
							"app": "app-2",
							"hub": "export-1",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-1",
						Namespace: "test",
						Labels: map[string]string{
							"app": "app-1",
							"hub": "export-1",
						},
					},
				},
			},
			"export-2": {
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-1",
						Namespace: "default",
						Labels: map[string]string{
							"app": "app-1",
							"hub": "export-2",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-1",
						Namespace: "test",
						Labels: map[string]string{
							"app": "app-1",
							"hub": "export-2",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		policies []*trafficv1alpha2.RoutePolicy
		want     []*trafficv1alpha2.Route
	}{
		{
			name: "only hub name",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{},
		},

		{
			name: "export name",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Name: "app-1",
								},
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{},
		},

		{
			name: "import name",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Name: "app-1",
								},
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{},
		},

		{
			name: "export namespace",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Namespace: "default",
								},
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-efb4858c4a53",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-2",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-2",
								Namespace: "default",
							},
						},
					},
				},
			},
		},
		{
			name: "export namespace and name",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Namespace: "default",
									Name:      "app-1",
								},
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
			},
		},

		{
			name: "import namespace",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Namespace: "default",
								},
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-efb4858c4a53",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-2",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-2",
								Namespace: "default",
							},
						},
					},
				},
			},
		},
		{
			name: "import namespace and name",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Namespace: "default",
									Name:      "app-1",
								},
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
			},
		},

		{
			name: "export labels",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
								},
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-786e6a1077a3",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "test",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "test",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
			},
		},
		{
			name: "export labels and namespace",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
									Namespace: "default",
								},
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
			},
		},

		{
			name: "import labels",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
								},
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-786e6a1077a3",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "test",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "test",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
			},
		},
		{
			name: "import labels and namespace",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
									Namespace: "default",
								},
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
			},
		},

		{
			name: "label 2 export to import 1",
			policies: []*trafficv1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: trafficv1alpha2.RoutePolicySpec{
						Exports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
									Namespace: "default",
								},
							},
							{
								HubName: "export-2",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
									Namespace: "default",
								},
							},
						},
						Imports: []trafficv1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: trafficv1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
									Namespace: "default",
								},
							},
						},
					},
				},
			},
			want: []*trafficv1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-5d5069c28854",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-2",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-cd2f4ea8a90c",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: trafficv1alpha2.RouteSpec{
						Import: trafficv1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: trafficv1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: trafficv1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policiesToRoutes(src, tt.policies)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("policiesToRoutes(): got - want + \n%s", diff)
			}
		})
	}
}

type fakeDataSource struct {
	svcs map[string][]*corev1.Service
}

func (f *fakeDataSource) ListServices(name string) []*corev1.Service {
	return f.svcs[name]
}

func (f *fakeDataSource) ListHubs() []*trafficv1alpha2.Hub {
	return nil
}
