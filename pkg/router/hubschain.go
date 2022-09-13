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
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/resource"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/wzshiming/sshproxy/permissions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HubsChainConfig struct {
	GetHubGateway func(hubName string, forHub string) v1alpha2.HubSpecGateway
}

func NewHubsChain(conf HubsChainConfig) *HubsChain {
	return &HubsChain{
		getHubGateway: conf.GetHubGateway,
	}
}

type HubsChain struct {

	// Get export's gateway for import
	getHubGateway func(hubName string, forHub string) v1alpha2.HubSpecGateway
}

func (h *HubsChain) Build(name string, origin, destination objref.ObjectRef, originPort, peerPort int32, ways []string) (map[string]*Bound, error) {
	hubsChains, err := h.buildRaw(name, origin, destination, originPort, peerPort, ways)
	if err != nil {
		return nil, err
	}

	mergeUnreachableRepeater(hubsChains)

	mergeContinuousReachableExport(hubsChains, ways)

	mergeContinuousReachableImport(hubsChains, ways)

	bound := h.modifyAuth(hubsChains)
	return bound, nil
}

type Bound struct {
	Outbound []*Chain
	Inbound  map[string]*AllowList
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

var identityFile = path.Join(consts.TunnelSshDir, consts.TunnelIdentityKeyName)

func (h *HubsChain) modifyAuth(m map[string][]*Chain) map[string]*Bound {
	bound := map[string]*Bound{}
	const (
		sshPrefix  = "ssh://"
		unixPrefix = "unix://"
	)
	for name, chains := range m {
		if bound[name] == nil {
			bound[name] = &Bound{}
		}
		bound[name].Outbound = chains

		for _, chain := range chains {
			for i, p := range chain.Proxy[1:] {
				if strings.HasPrefix(p, sshPrefix) {
					uri, _ := url.Parse(p)
					query := uri.Query()

					query.Set("identity_file", identityFile)
					uri.RawQuery = query.Encode()
					uri.User = url.User(name)
					chain.Proxy[i+1] = strings.Replace(uri.String(), "%2F", "/", -1)

					targetHub := query.Get("target_hub")
					if bound[targetHub] == nil {
						bound[targetHub] = &Bound{}
					}

					allowList := &AllowList{}
					if i == 0 {
						if !strings.Contains(chain.Proxy[i], "/") {
							allowList.DirectTcpip.Allows = []string{chain.Proxy[i]}
						} else if strings.HasPrefix(chain.Proxy[i], unixPrefix) {
							next, _ := url.Parse(chain.Proxy[i])
							allowList.DirectStreamlocal.Allows = []string{next.Path}
						}
					} else if strings.HasPrefix(chain.Proxy[i], sshPrefix) {
						next, _ := url.Parse(chain.Proxy[i])
						allowList.DirectTcpip.Allows = []string{next.Host}
					}
					if bound[targetHub].Inbound == nil {
						bound[targetHub].Inbound = map[string]*AllowList{}
					}
					bound[targetHub].Inbound[name] = bound[targetHub].Inbound[name].Merge(allowList)
				}
			}
			for i, b := range chain.Bind[1:] {
				if strings.HasPrefix(b, sshPrefix) {
					uri, _ := url.Parse(b)
					query := uri.Query()
					query.Set("identity_file", identityFile)
					uri.RawQuery = query.Encode()
					uri.User = url.User(name)
					chain.Bind[i+1] = strings.Replace(uri.String(), "%2F", "/", -1)

					targetHub := query.Get("target_hub")
					if bound[targetHub] == nil {
						bound[targetHub] = &Bound{}
					}

					allowList := &AllowList{}
					if i == 0 {
						if !strings.Contains(chain.Bind[i], "/") {
							allowList.TcpipForward.Allows = []string{chain.Bind[i]}
						} else if strings.HasPrefix(chain.Bind[i], unixPrefix) {
							next, _ := url.Parse(chain.Bind[i])
							allowList.StreamlocalForward.Allows = []string{next.Path}
						}
					} else if strings.HasPrefix(chain.Bind[i], sshPrefix) {
						next, _ := url.Parse(chain.Bind[i])
						allowList.DirectTcpip.Allows = []string{next.Host}
					}
					if bound[targetHub].Inbound == nil {
						bound[targetHub].Inbound = map[string]*AllowList{}
					}
					bound[targetHub].Inbound[name] = bound[targetHub].Inbound[name].Merge(allowList)
				}
			}
		}
	}
	return bound
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
		destinationAddress := fmt.Sprintf(":%d", peerPort)
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
			hubURI := sshURI(gw.Address, r.HubName)
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

func sshURI(address string, target string) string {
	return fmt.Sprintf("ssh://%s?target_hub=%s", address, target)
}

func ConvertInboundToResourcers(name, namespace string, labels map[string]string, cs map[string]*Bound) (map[string][]resource.Resourcer, error) {
	out := map[string][]resource.Resourcer{}

	for inboundHub, bound := range cs {
		if len(bound.Inbound) == 0 {
			continue
		}
		r, err := convertInboundToResourcer(name, namespace, labels, bound)
		if err != nil {
			return nil, err
		}

		out[inboundHub] = r
	}
	return out, nil
}

func convertInboundToResourcer(name, namespace string, labels map[string]string, b *Bound) ([]resource.Resourcer, error) {
	inbound, err := json.Marshal(b.Inbound)
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
			consts.TunnelRulesAllowKey: string(inbound),
		},
	}

	return []resource.Resourcer{resource.ConfigMap{configMap}}, nil
}

func ConvertInboundAuthorizedToResourcers(suffix, namespace string, labels map[string]string, cs map[string]*Bound, getAuthorized func(name string) string) (map[string][]resource.Resourcer, error) {
	out := map[string][]resource.Resourcer{}
	for inboundHub, bound := range cs {
		if len(bound.Inbound) == 0 {
			continue
		}
		for outboundHub := range bound.Inbound {
			name := fmt.Sprintf("%s-%s", outboundHub, suffix)
			r, err := convertInboundAuthorizedToResourcer(outboundHub, name, namespace, labels, getAuthorized)
			if err != nil {
				return nil, err
			}
			out[inboundHub] = append(out[inboundHub], r...)
		}
	}
	return out, nil
}

func convertInboundAuthorizedToResourcer(hubName, name, namespace string, labels map[string]string, getAuthorized func(name string) string) ([]resource.Resourcer, error) {
	authorized := strings.TrimSpace(getAuthorized(hubName))
	if authorized == "" {
		return nil, fmt.Errorf("failed get authorized %q", hubName)
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			consts.TunnelUserKey:           hubName,
			consts.TunnelAuthorizedKeyName: fmt.Sprintf("%s %s@ferryproxy.io", authorized, hubName),
		},
	}
	return []resource.Resourcer{resource.ConfigMap{configMap}}, nil
}

func ConvertOutboundToResourcers(name, namespace string, labels map[string]string, cs map[string]*Bound) (map[string][]resource.Resourcer, error) {
	out := map[string][]resource.Resourcer{}

	for k, bound := range cs {
		if len(bound.Outbound) == 0 {
			continue
		}
		r, err := convertOutboundToResourcer(name, namespace, labels, bound)
		if err != nil {
			return nil, err
		}

		out[k] = r
	}
	return out, nil
}

func convertOutboundToResourcer(name, namespace string, labels map[string]string, b *Bound) ([]resource.Resourcer, error) {
	outbound, err := json.Marshal(b.Outbound)
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
			consts.TunnelRulesKey: string(outbound),
		},
	}

	return []resource.Resourcer{resource.ConfigMap{configMap}}, nil
}

type Chain struct {
	Bind  []string `json:"bind"`
	Proxy []string `json:"proxy"`
}

type AllowList struct {
	DirectStreamlocal  permissions.Permission `json:"direct-streamlocal"`
	DirectTcpip        permissions.Permission `json:"direct-tcpip"`
	StreamlocalForward permissions.Permission `json:"streamlocal-forward"`
	TcpipForward       permissions.Permission `json:"tcpip-forward"`
}

func (a *AllowList) Merge(a2 *AllowList) *AllowList {
	if a == nil {
		return a2
	}
	return &AllowList{
		DirectStreamlocal:  mergePermission(a.DirectStreamlocal, a2.DirectStreamlocal),
		DirectTcpip:        mergePermission(a.DirectTcpip, a2.DirectTcpip),
		StreamlocalForward: mergePermission(a.StreamlocalForward, a2.StreamlocalForward),
		TcpipForward:       mergePermission(a.TcpipForward, a2.TcpipForward),
	}
}

func mergePermission(p1, p2 permissions.Permission) permissions.Permission {
	p := permissions.Permission{}
	if p1.Default == p2.Default {
		p.Default = p1.Default
	}

	if len(p1.Allows) != 0 {
		if len(p2.Allows) != 0 {
			p.Allows = stringsSet(p1.Allows, p2.Allows)
		} else {
			p.Allows = p1.Allows
		}
	} else {
		if len(p2.Allows) != 0 {
			p.Allows = p2.Allows
		}
	}

	if len(p1.Blocks) != 0 {
		if len(p2.Blocks) != 0 {
			p.Blocks = stringsSet(p1.Blocks, p2.Blocks)
		} else {
			p.Blocks = p1.Blocks
		}
	} else {
		if len(p2.Blocks) != 0 {
			p.Blocks = p2.Blocks
		}
	}
	return p
}

func stringsSet(data ...[]string) []string {
	m := map[string]struct{}{}
	for _, list := range data {
		for _, item := range list {
			m[item] = struct{}{}
		}
	}

	out := make([]string, 0, len(m))
	for item := range m {
		out = append(out, item)
	}

	sort.Strings(out)
	return out
}
