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
	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildMirrorTunnelRoutes(hubs []*v1alpha2.Hub, importHubName string) []*v1alpha2.Route {
	routes := make([]*v1alpha2.Route, 0, len(hubs))
	for _, hub := range hubs {
		if hub.Name == importHubName {
			continue
		}
		route := buildMirrorTunnelRoute(hub, importHubName)
		routes = append(routes, route)
	}
	return routes
}

func buildMirrorTunnelRoute(hub *v1alpha2.Hub, importHubName string) *v1alpha2.Route {
	controller := true
	r := &v1alpha2.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hub.Name + "-ferry-tunnel",
			Namespace: hub.Namespace,
			Labels:    labelsForRoute,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1alpha2.GroupVersion.String(),
					Kind:       "Hub",
					Name:       hub.Name,
					UID:        hub.UID,
					Controller: &controller,
				},
			},
		},
		Spec: v1alpha2.RouteSpec{
			Export: v1alpha2.RouteSpecRule{
				HubName: hub.Name,
				Service: v1alpha2.RouteSpecRuleService{
					Namespace: "ferry-tunnel-system",
					Name:      "ferry-tunnel",
				},
				Ports: []v1alpha2.RouteSpecRulePort{
					{
						Name: "http",
						Port: 8080,
					},
				},
			},
			Import: v1alpha2.RouteSpecRule{
				HubName: importHubName,
				Service: v1alpha2.RouteSpecRuleService{
					Namespace: "ferry-tunnel-system",
					Name:      hub.Name + "-ferry-tunnel",
				},
				Ports: []v1alpha2.RouteSpecRulePort{
					{
						Name: "http",
						Port: 8080,
					},
				},
			},
		},
	}
	return r
}
