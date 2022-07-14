package router

import (
	"fmt"

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
			if len(importGateway.Navigation) != 0 {
				ok, err := isHubs(importGateway.Navigation)
				if err != nil {
					return nil, fmt.Errorf("%w for export gateway reachable with import navigation: export %q, import %q ", err, exportWay, importWay)
				}
				if ok {
					insertFunc(i+1, importGateway.Navigation...)
				}
			}
			if len(exportGateway.Reception) != 0 {
				ok, err := isHubs(exportGateway.Reception)
				if err != nil {
					return nil, fmt.Errorf("%w for export gateway reachable with export reception: export %q, import %q ", err, exportWay, importWay)
				}
				if ok {
					insertFunc(i+1, reverse(exportGateway.Reception)...)
				}
			}
		} else if importGateway.Reachable {
			if len(importGateway.Reception) != 0 {
				ok, err := isHubs(importGateway.Reception)
				if err != nil {
					return nil, fmt.Errorf("%w for import gateway reachable with import reception: export %q, import %q ", err, exportWay, importWay)
				}
				if ok {
					insertFunc(i+1, importGateway.Reception...)
				}
			}
			if len(exportGateway.Navigation) != 0 {
				ok, err := isHubs(exportGateway.Navigation)
				if err != nil {
					return nil, fmt.Errorf("%w for import gateway reachable with export navigation: export %q, import %q ", err, exportWay, importWay)
				}
				if ok {
					insertFunc(i+1, reverse(exportGateway.Navigation)...)
				}
			}
		} else {
			if len(importGateway.Navigation) == 0 && len(exportGateway.Navigation) == 0 {
				break
			}

			if len(importGateway.Navigation) != 0 {
				ok, err := isHubs(importGateway.Navigation)
				if err != nil {
					return nil, fmt.Errorf("%w for gateway not reachable with import navigation: export %q, import %q ", err, exportWay, importWay)
				}
				if ok {
					insertFunc(i+1, importGateway.Navigation...)
				}
			}
			if len(exportGateway.Navigation) != 0 {
				ok, err := isHubs(exportGateway.Navigation)
				if err != nil {
					return nil, fmt.Errorf("%w for gateway not reachable with export navigation: export %q, import %q ", err, exportWay, importWay)
				}
				if ok {
					insertFunc(i+1, reverse(exportGateway.Navigation)...)
				}
			}
		}
	}

	return ways, nil
}

var ErrBothHubAndProxy = fmt.Errorf("both proxy and hub exist")

func isHubs(ways v1alpha2.HubSpecGatewayWays) (isHub bool, err error) {
	hasHub := false
	hasProxy := false
	for _, way := range ways {
		if way.HubName != "" {
			if hasProxy {
				return false, ErrBothHubAndProxy
			}
			hasHub = true
		}
		if way.Proxy != "" {
			if hasHub {
				return false, ErrBothHubAndProxy
			}
			hasProxy = true
		}
	}
	return hasHub, nil
}

func reverse[T any](a []T) []T {
	for i := 0; i != len(a)/2; i++ {
		a[i], a[len(a)-i-1] = a[len(a)-i-1], a[i]
	}
	return a
}
