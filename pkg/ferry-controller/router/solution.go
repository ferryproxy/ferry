package router

import (
	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
)

type SolutionConfig struct {
	GetHubGateway func(hubName string, forHub string) v1alpha2.HubSpecGateway
}

func NewSolution(conf SolutionConfig) *Solution {
	return &Solution{
		getHubGateway: conf.GetHubGateway,
	}
}

type Solution struct {
	getHubGateway func(hubName string, forHub string) v1alpha2.HubSpecGateway
}

// CalculateWays is calculated the ways based on the export hub and import hub
func (s *Solution) CalculateWays(exportHub, importHub string) ([]string, error) {
	ways := []string{exportHub, importHub}

	hubs := map[string]int{}
	insertFunc := func(i int, vs ...v1alpha2.HubSpecGatewayWay) {
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

	return ways, nil
}

func reverse[T any](a []T) []T {
	for i := 0; i != len(a)/2; i++ {
		a[i], a[len(a)-i-1] = a[len(a)-i-1], a[i]
	}
	return a
}
