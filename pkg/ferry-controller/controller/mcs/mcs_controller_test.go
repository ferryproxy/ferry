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

package mcs

import (
	"testing"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

func Test_mcsToRoutePolicies(t *testing.T) {
	type args struct {
		importMap map[string][]*v1alpha1.ServiceImport
		exportMap map[string][]*v1alpha1.ServiceExport
	}
	tests := []struct {
		name string
		args args
		want []*v1alpha2.RoutePolicy
	}{
		{
			name: "1 to 1",
			args: args{
				exportMap: map[string][]*v1alpha1.ServiceExport{
					"cluster-0": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
				importMap: map[string][]*v1alpha1.ServiceImport{
					"cluster-1": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
			},
			want: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mcs-default-svc-1",
						Namespace: "ferry-system",
						Labels:    labelsForRoutePolicy,
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "cluster-0",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "cluster-1",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
						},
					},
				},
			},
		},
		{
			name: "unmatch",
			args: args{
				exportMap: map[string][]*v1alpha1.ServiceExport{
					"cluster-0": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
				importMap: map[string][]*v1alpha1.ServiceImport{
					"cluster-1": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-2",
								Namespace: "default",
							},
						},
					},
				},
			},
			want: []*v1alpha2.RoutePolicy{},
		},
		{
			name: "1 to 2",
			args: args{
				exportMap: map[string][]*v1alpha1.ServiceExport{
					"cluster-0": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
				importMap: map[string][]*v1alpha1.ServiceImport{
					"cluster-1": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
					"cluster-2": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
			},
			want: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mcs-default-svc-1",
						Namespace: "ferry-system",
						Labels:    labelsForRoutePolicy,
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "cluster-0",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "cluster-1",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
							{
								HubName: "cluster-2",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
						},
					},
				},
			},
		},
		{
			name: "2 to 1",
			args: args{
				exportMap: map[string][]*v1alpha1.ServiceExport{
					"cluster-0": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
					"cluster-2": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
				importMap: map[string][]*v1alpha1.ServiceImport{
					"cluster-1": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
			},
			want: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mcs-default-svc-1",
						Namespace: "ferry-system",
						Labels:    labelsForRoutePolicy,
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "cluster-0",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
							{
								HubName: "cluster-2",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "cluster-1",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
						},
					},
				},
			},
		},
		{
			name: "2 to 2",
			args: args{
				exportMap: map[string][]*v1alpha1.ServiceExport{
					"cluster-0": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
					"cluster-2": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
				importMap: map[string][]*v1alpha1.ServiceImport{
					"cluster-1": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
					"cluster-3": {
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "svc-1",
								Namespace: "default",
							},
						},
					},
				},
			},
			want: []*v1alpha2.RoutePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mcs-default-svc-1",
						Namespace: "ferry-system",
						Labels:    labelsForRoutePolicy,
					},
					Spec: v1alpha2.RoutePolicySpec{
						Exports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "cluster-0",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
							{
								HubName: "cluster-2",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
						},
						Imports: []v1alpha2.RoutePolicySpecRule{
							{
								HubName: "cluster-1",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
							{
								HubName: "cluster-3",
								Service: v1alpha2.RoutePolicySpecRuleService{Namespace: "default", Name: "svc-1"},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mcsToRoutePolicies(tt.args.importMap, tt.args.exportMap)

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("mcsToRoutePolicies(): got - want + \n%s", diff)
			}
		})
	}
}
