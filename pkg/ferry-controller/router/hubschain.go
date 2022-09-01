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
	"strings"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferry-controller/router/resource"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HubsChainConfig struct {
	GetIdentity func(hubName string) string

	GetHubGateway func(hubName string, forHub string) v1alpha2.HubSpecGateway
}

func NewHubsChain(conf HubsChainConfig) *HubsChain {
	return &HubsChain{
		getHubGateway: conf.GetHubGateway,
		getIdentity:   conf.GetIdentity,
	}
}

type HubsChain struct {

	// Get identity for hub
	getIdentity func(hubName string) string

	// Get export's gateway for import
	getHubGateway func(hubName string, forHub string) v1alpha2.HubSpecGateway
}

func (h *HubsChain) Build(name string, origin, destination objref.ObjectRef, originPort, peerPort int32, ways []string) (hubsChains map[string][]*Chain, err error) {
	hubsChains, err = h.buildRaw(name, origin, destination, originPort, peerPort, ways)
	if err != nil {
		return nil, err
	}

	mergeUnreachableRepeater(hubsChains)

	mergeContinuousReachableExport(hubsChains, ways)

	mergeContinuousReachableImport(hubsChains, ways)

	for _, way := range ways {
		if len(hubsChains[way]) == 0 {
			hubsChains[way] = nil
		}
	}
	return hubsChains, nil
}

func mergeUnreachableRepeater(m map[string][]*Chain) {
	for key, chains := range m {
		if len(chains) == 2 {
			exportBind := chains[0].Bind
			importProxy := chains[1].Proxy
			if exportBind[0] == importProxy[0] {
				m[key] = []*Chain{
					{
						Proxy: chains[0].Proxy,
						Bind:  mergeStrings(chains[1].Bind, importProxy[1:]),
					},
				}
			}
		}
	}
	return
}

func mergeContinuousReachableExport(m map[string][]*Chain, hubNames []string) {
	for i := 0; i < len(hubNames)-1; i++ {
		exportHubName := hubNames[i]
		importHubName := hubNames[i+1]
		exportHub := m[exportHubName]
		importHub := m[importHubName]
		if len(exportHub) == 1 && len(importHub) == 1 {
			exportBind := exportHub[0].Bind
			importProxy := importHub[0].Proxy
			if len(exportBind) == 1 &&
				strings.HasPrefix(exportBind[0], "unix://") &&
				exportBind[0] == importProxy[0] {
				importHub[0].Proxy = mergeStrings(exportHub[0].Proxy, importHub[0].Proxy[1:])
				m[exportHubName] = m[exportHubName][1:]
			}
		}
	}
	return
}

func mergeContinuousReachableImport(m map[string][]*Chain, hubNames []string) {
	for i := len(hubNames) - 1; i > 0; i-- {
		exportHubName := hubNames[i-1]
		importHubName := hubNames[i]
		exportHub := m[exportHubName]
		importHub := m[importHubName]
		if len(exportHub) == 1 && len(importHub) == 1 {
			exportBind := exportHub[0].Bind
			importProxy := importHub[0].Proxy
			if len(importProxy) == 1 &&
				strings.HasPrefix(importProxy[0], "unix://") &&
				exportBind[0] == importProxy[0] {
				exportHub[0].Bind = mergeStrings(importHub[0].Bind, exportBind[1:])
				m[importHubName] = m[importHubName][1:]
			}
		}
	}
	return
}

func mergeStrings(a, b []string) []string {
	out := make([]string, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

func (h *HubsChain) buildRaw(name string, origin, destination objref.ObjectRef, originPort, peerPort int32, ways []string) (hubsChains map[string][]*Chain, err error) {

	hubsChains = map[string][]*Chain{}

	for i := 0; i < len(ways)-1; i++ {
		exportHubName := ways[i]
		importHubName := ways[i+1]

		exportRepeater := i != 0
		importRepeater := i != len(ways)-2

		exportGateway := h.getHubGateway(exportHubName, importHubName)
		importGateway := h.getHubGateway(importHubName, exportHubName)

		exportHubChain, importHubChain, err := h.buildPeer(
			name, origin, destination, originPort, peerPort,
			exportHubName, exportRepeater, exportGateway,
			importHubName, importRepeater, importGateway,
		)
		if err != nil {
			return nil, err
		}

		if exportHubChain != nil {
			hubsChains[exportHubName] = append(hubsChains[exportHubName], exportHubChain)
		}

		if importHubChain != nil {
			hubsChains[importHubName] = append(hubsChains[importHubName], importHubChain)
		}
	}

	return hubsChains, nil
}

func (h *HubsChain) buildPeer(
	name string, origin, destination objref.ObjectRef, originPort, peerPort int32,
	exportHubName string, exportRepeater bool, exportGateway v1alpha2.HubSpecGateway,
	importHubName string, importRepeater bool, importGateway v1alpha2.HubSpecGateway,
) (exportHubChain *Chain, importHubChain *Chain, err error) {

	chain := &Chain{
		Bind:  []string{},
		Proxy: []string{},
	}

	if importRepeater {
		unixSocks := unixSocksPath(name)
		chain.Bind = append(chain.Bind, unixSocks)
	} else {
		destinationAddress := fmt.Sprintf("0.0.0.0:%d", peerPort)
		chain.Bind = append(chain.Bind, destinationAddress)
	}

	if exportRepeater {
		unixSocks := unixSocksPath(name)
		chain.Proxy = append(chain.Proxy, unixSocks)
	} else {
		originSvc := fmt.Sprintf("%s.%s.svc:%d", origin.Name, origin.Namespace, originPort)
		chain.Proxy = append(chain.Proxy, originSvc)
	}

	if exportGateway.Reachable {
		proxies := []v1alpha2.HubSpecGatewayProxy{}
		proxies = append(proxies, v1alpha2.HubSpecGatewayProxy{
			HubName: exportHubName,
		})
		proxies = append(proxies, exportGateway.ReceptionProxy...)
		proxies = append(proxies, importGateway.NavigationProxy...)
		chain.Proxy = h.proxies(chain.Proxy, importHubName, proxies)
		return nil, chain, nil
	} else if importGateway.Reachable {
		binds := []v1alpha2.HubSpecGatewayProxy{}
		binds = append(binds, v1alpha2.HubSpecGatewayProxy{
			HubName: importHubName,
		})
		binds = append(binds, importGateway.ReceptionProxy...)
		binds = append(binds, exportGateway.NavigationProxy...)
		chain.Bind = h.proxies(chain.Bind, exportHubName, binds)
		return chain, nil, nil
	}

	return nil, nil, fmt.Errorf("both export %q and import %q hubs are unreachable", exportHubName, importHubName)
}

func (h *HubsChain) proxies(a []string, prev string, proxies []v1alpha2.HubSpecGatewayProxy) []string {
	for _, r := range proxies {
		if r.HubName != "" {
			gw := h.getHubGateway(r.HubName, prev)
			hubURI := sshURI(gw.Address, h.getIdentity(r.HubName))
			a = append(a, hubURI)
			prev = r.HubName
		} else if r.Proxy != "" {
			a = append(a, r.Proxy)
		}
	}
	return a
}

func unixSocksPath(name string) string {
	return fmt.Sprintf("unix:///dev/shm/%s.socks", name)
}

func sshURI(address string, identity string) string {
	return fmt.Sprintf("ssh://%s?identity_data=%s", address, identity)
}

func ConvertChainsToResourcers(name, namespace string, labels map[string]string, cs map[string][]*Chain) (map[string][]resource.Resourcer, error) {
	out := map[string][]resource.Resourcer{}

	for k, chains := range cs {
		if len(chains) == 0 {
			continue
		}
		r, err := convertChainToResourcer(name, namespace, labels, chains)
		if err != nil {
			return nil, err
		}
		out[k] = r
	}
	return out, nil
}

func convertChainToResourcer(name, namespace string, labels map[string]string, cs []*Chain) ([]resource.Resourcer, error) {
	data, err := json.MarshalIndent(cs, "", "  ")
	if err != nil {
		return nil, err
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			consts.TunnelRulesKey: string(data),
		},
	}

	return []resource.Resourcer{resource.ConfigMap{configMap}}, nil
}

type Chain struct {
	Bind  []string `json:"bind"`
	Proxy []string `json:"proxy"`
}
