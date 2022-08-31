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

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
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
		policies []*v1alpha2.RoutePolicy
		want     []*v1alpha2.Route
	}{
		{
			name: "only hub name",
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{},
		},

		{
			name: "export name",
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Name: "app-1",
								},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{},
		},

		{
			name: "import name",
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Name: "app-1",
								},
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{},
		},

		{
			name: "export namespace",
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Namespace: "default",
								},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-2-import-1-default-app-2",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-2",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
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
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Namespace: "default",
									Name:      "app-1",
								},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
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
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Namespace: "default",
								},
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-2-import-1-default-app-2",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-2",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
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
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Namespace: "default",
									Name:      "app-1",
								},
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
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
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
								},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-test-app-1-import-1-test-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "test",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "test",
							},
						},
					},
				},
			},
		},
		{
			name: "export labels and namespace",
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
									Namespace: "default",
								},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
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
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
								},
							},
						},
					},
				},
			},
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-test-app-1-import-1-test-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "test",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "test",
							},
						},
					},
				},
			},
		},
		{
			name: "import labels and namespace",
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
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
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
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
			policies: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "export-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
									Namespace: "default",
								},
							},
							{
								HubName: "export-2",
								Service: v1alpha2.RoutePolicySpecRuleService{
									Labels: map[string]string{
										"app": "app-1",
									},
									Namespace: "default",
								},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "import-1",
								Service: v1alpha2.RoutePolicySpecRuleService{
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
			want: []*v1alpha2.Route{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-1-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-export-2-default-app-1-import-1-default-app-1",
						OwnerReferences: ownerReferences,
						Labels:          labelsForRoute,
					},
					Spec: v1alpha2.RouteSpec{
						Import: v1alpha2.RouteSpecRule{
							HubName: "import-1",
							Service: v1alpha2.RouteSpecRuleService{
								Name:      "app-1",
								Namespace: "default",
							},
						},
						Export: v1alpha2.RouteSpecRule{
							HubName: "export-2",
							Service: v1alpha2.RouteSpecRuleService{
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
