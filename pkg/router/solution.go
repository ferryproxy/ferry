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
	trafficv1alpha2 "github.com/ferryproxy/api/apis/traffic/v1alpha2"
)

type SolutionConfig struct {
	GetHubGateway func(hubName string, forHub string) trafficv1alpha2.HubSpecGateway
}

func NewSolution(conf SolutionConfig) *Solution {
	return &Solution{
		getHubGateway: conf.GetHubGateway,
	}
}

type Solution struct {
	getHubGateway func(hubName string, forHub string) trafficv1alpha2.HubSpecGateway
}

// CalculateWays is calculated the ways based on the export hub and import hub
func (s *Solution) CalculateWays(exportHub, importHub string) ([]string, error) {
	ways := []string{exportHub, importHub}

	hubs := map[string]int{}
	insertFunc := func(i int, vs ...trafficv1alpha2.HubSpecGatewayWay) {
		w := make([]string, 0, len(ways)+len(vs))
		w = append(w, ways[:i]...)

		for _, v := range vs {
			_, ok := hubs[v.HubName]
			if ok {
				continue
			}
			w = append(w, v.HubName)
		}
		w = append(w, ways[i:]...)
		ways = w
	}
	for i := 0; i < len(ways)-1; i++ {
		exportWay := ways[i]
		importWay := ways[i+1]

		hubs[exportWay] = i
		hubs[importWay] = i + 1

		exportGateway := s.getHubGateway(exportWay, importWay)
		importGateway := s.getHubGateway(importWay, exportWay)
		if exportGateway.Reachable {
			if len(importGateway.NavigationWay) != 0 {
				insertFunc(i+1, importGateway.NavigationWay...)
			}
			if len(exportGateway.ReceptionWay) != 0 {
				insertFunc(i+1, reverse(exportGateway.ReceptionWay)...)
			}
		} else if importGateway.Reachable {
			if len(importGateway.ReceptionWay) != 0 {
				insertFunc(i+1, importGateway.ReceptionWay...)
			}
			if len(exportGateway.NavigationWay) != 0 {
				insertFunc(i+1, reverse(exportGateway.NavigationWay)...)
			}
		} else {
			if len(importGateway.NavigationWay) == 0 && len(exportGateway.NavigationWay) == 0 {
				break
			}

			if len(importGateway.NavigationWay) != 0 {
				insertFunc(i+1, importGateway.NavigationWay...)
			}
			if len(exportGateway.NavigationWay) != 0 {
				insertFunc(i+1, reverse(exportGateway.NavigationWay)...)
			}
		}
	}

	ways = removeInvalidWays(ways)
	return ways, nil
}

func removeInvalidWays(ways []string) []string {
	hit := map[string]int{}
	for i := 0; i < len(ways); i++ {
		way := ways[i]
		if prevIndex, ok := hit[way]; ok {
			copy(ways[prevIndex:], ways[i:])
			ways = ways[:len(ways)-(i-prevIndex)]
			i = prevIndex
		} else {
			hit[way] = i
		}
	}
	return ways
}

func reverse[T any](a []T) []T {
	for i := 0; i != len(a)/2; i++ {
		a[i], a[len(a)-i-1] = a[len(a)-i-1], a[i]
	}
	return a
}
