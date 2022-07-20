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
	"fmt"
	"testing"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/google/go-cmp/cmp"
)

func TestHubsChain_Build(t *testing.T) {
	origin := objref.ObjectRef{Name: "oname", Namespace: "ons"}
	destination := objref.ObjectRef{Name: "dname", Namespace: "dns"}
	const (
		originPort = 80
		peerPort   = 10000
	)

	tests := []struct {
		name           string
		hubs           map[string]*v1alpha2.Hub
		ways           []string
		wantHubsChains map[string][]*Chain
		wantErr        bool
	}{

		// the 0x0 is can reachable to export, the 0x1 is not

		// 2 hubs (export, import)
		// 0b0
		{
			name: "2 hubs 0b0",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "export:8080",
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
						},
					},
				},
			},
		},
		// 0b1
		{
			name: "2 hubs 0b1",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "import:8080",
						},
					},
				},
			},
			ways: []string{
				"export",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import": nil,
			},
		},

		// 3 hubs (export, repeater, import)
		// 0b10
		{
			name: "3 hubs 0b10",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{},
				},
				"repeater": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "repeater:8080",
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater:8080?identity_data=repeater-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
						},
						Proxy: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater:8080?identity_data=repeater-identity",
						},
					},
				},
				"repeater": nil,
			},
		},
		// 0b01
		{
			name: "3 hubs 0b01",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "export:8080",
						},
					},
				},
				"repeater": {
					Spec: v1alpha2.HubSpec{},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "import:8080",
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": nil,
				"repeater": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
						},
					},
				},
			},
		},
		// 0b00
		{
			name: "3 hubs 0b00",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
							"ssh://repeater:8080?identity_data=repeater-identity",
						},
					},
				},
				"repeater": nil,
			},
		},
		// 0b11
		{
			name: "3 hubs 0b11",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{},
				},
				"repeater": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater": {
								Reachable: true,
								Address:   "import:8080",
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
							"ssh://repeater:8080?identity_data=repeater-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import":   nil,
				"repeater": nil,
			},
		},

		// 4 hubs (export, repeater-export, repeater-import, import)
		// 0b000
		{
			name: "4 hubs 0b000",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater-export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
							"ssh://repeater-export:8080?identity_data=repeater-export-identity",
							"ssh://repeater-import:8080?identity_data=repeater-import-identity",
						},
					},
				},
				"repeater-import": nil,
				"repeater-export": nil,
			},
		},
		// 0b111
		{
			name: "4 hubs 0b111",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{},
				},
				"repeater-export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "import:8080",
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
							"ssh://repeater-import:8080?identity_data=repeater-import-identity",
							"ssh://repeater-export:8080?identity_data=repeater-export-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import":          nil,
				"repeater-import": nil,
				"repeater-export": nil,
			},
		},
		// 0b100
		{
			name: "4 hubs 0b100",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{},
				},
				"repeater-export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
							"repeater-import": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater-export:8080?identity_data=repeater-export-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import": {
					{
						Bind: []string{"0.0.0.0:10000"},
						Proxy: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater-export:8080?identity_data=repeater-export-identity",
							"ssh://repeater-import:8080?identity_data=repeater-import-identity",
						},
					},
				},
				"repeater-import": nil,
				"repeater-export": nil,
			},
		},
		// 0b011
		{
			name: "4 hubs 0b011",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater-export": {
					Spec: v1alpha2.HubSpec{},
				},
				"repeater-import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "import:8080",
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export":          nil,
				"import":          nil,
				"repeater-import": nil,
				"repeater-export": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
							"ssh://repeater-import:8080?identity_data=repeater-import-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
						},
					},
				},
			},
		},
		// 0b110
		{
			name: "4 hubs 0b110",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{},
				},
				"repeater-export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
							"repeater-export": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater-import:8080?identity_data=repeater-import-identity",
							"ssh://repeater-export:8080?identity_data=repeater-export-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import": {
					{
						Bind: []string{"0.0.0.0:10000"},
						Proxy: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater-import:8080?identity_data=repeater-import-identity",
						},
					},
				},
				"repeater-import": nil,
				"repeater-export": nil,
			},
		},
		// 0b001
		{
			name: "4 hubs 0b001",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater-export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "import:8080",
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": nil,
				"repeater-import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
							"ssh://repeater-export:8080?identity_data=repeater-export-identity",
						},
					},
				},
				"repeater-export": nil,
			},
		},
		// 0b101
		{
			name: "4 hubs 0b101",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{},
				},
				"repeater-export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
							"repeater-import": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "import:8080",
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater-export:8080?identity_data=repeater-export-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import": nil,
				"repeater-import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
						},
						Proxy: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater-export:8080?identity_data=repeater-export-identity",
						},
					},
				},
				"repeater-export": nil,
			},
		},
		// 0b010
		{
			name: "4 hubs 0b010",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater-export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{},
					},
				},
				"repeater-import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
							"import": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{},
					},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
						},
						Proxy: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater-import:8080?identity_data=repeater-import-identity",
						},
					},
				},
				"repeater-import": nil,
				"repeater-export": {
					{
						Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater-import:8080?identity_data=repeater-import-identity",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
						},
					},
				},
			},
		},

		// 2 hubs with proxy (export, import)
		// 0b0
		{
			name: "2 hubs 0b0 with proxy",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "export:8080",
							ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://export-reception-1:8080",
								},
								{
									Proxy: "socks5://export-reception-2:8080",
								},
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://import-navigation-1:8080",
								},
								{
									Proxy: "socks5://import-navigation-2:8080",
								},
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
							"socks5://export-reception-1:8080",
							"socks5://export-reception-2:8080",
							"socks5://import-navigation-1:8080",
							"socks5://import-navigation-2:8080",
						},
					},
				},
			},
		},
		// 0b1
		{
			name: "2 hubs 0b1 with proxy",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://export-navigation-1:8080",
								},
								{
									Proxy: "socks5://export-navigation-2:8080",
								},
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "import:8080",
							ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://import-reception-1:8080",
								},
								{
									Proxy: "socks5://import-reception-2:8080",
								},
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
							"socks5://import-reception-1:8080",
							"socks5://import-reception-2:8080",
							"socks5://export-navigation-1:8080",
							"socks5://export-navigation-2:8080",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import": nil,
			},
		},

		// 3 hubs with proxy (export, repeater, import)
		// 0b10
		{
			name: "3 hubs 0b10 with proxy",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://export-navigation-1:8080",
								},
								{
									Proxy: "socks5://export-navigation-2:8080",
								},
							},
						},
					},
				},
				"repeater": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "repeater:8080",
							ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://repeater-reception-1:8080",
								},
								{
									Proxy: "socks5://repeater-reception-2:8080",
								},
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://import-navigation-1:8080",
								},
								{
									Proxy: "socks5://import-navigation-2:8080",
								},
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater:8080?identity_data=repeater-identity",
							"socks5://repeater-reception-1:8080",
							"socks5://repeater-reception-2:8080",
							"socks5://export-navigation-1:8080",
							"socks5://export-navigation-2:8080",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
						},
						Proxy: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://repeater:8080?identity_data=repeater-identity",
							"socks5://repeater-reception-1:8080",
							"socks5://repeater-reception-2:8080",
							"socks5://import-navigation-1:8080",
							"socks5://import-navigation-2:8080",
						},
					},
				},
				"repeater": nil,
			},
		},
		// 0b01
		{
			name: "3 hubs 0b01 with proxy",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "export:8080",
							ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://export-reception-1:8080",
								},
								{
									Proxy: "socks5://export-reception-2:8080",
								},
							},
						},
					},
				},
				"repeater": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://repeater-navigation-1:8080",
								},
								{
									Proxy: "socks5://repeater-navigation-2:8080",
								},
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "import:8080",
							ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://import-reception-1:8080",
								},
								{
									Proxy: "socks5://import-reception-2:8080",
								},
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": nil,
				"repeater": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
							"socks5://import-reception-1:8080",
							"socks5://import-reception-2:8080",
							"socks5://repeater-navigation-1:8080",
							"socks5://repeater-navigation-2:8080",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
							"socks5://export-reception-1:8080",
							"socks5://export-reception-2:8080",
							"socks5://repeater-navigation-1:8080",
							"socks5://repeater-navigation-2:8080",
						},
					},
				},
			},
		},
		// 0b00
		{
			name: "3 hubs 0b00 with proxy",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater": {
								Reachable: true,
								Address:   "export:8080",
								ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://export-reception-1:8080",
									},
									{
										Proxy: "socks5://export-reception-2:8080",
									},
								},
							},
						},
					},
				},
				"repeater": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://repeater-navigation-1:8080",
								},
								{
									Proxy: "socks5://repeater-navigation-2:8080",
								},
							},
						},
						Override: map[string]v1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater:8080",
								ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://repeater-reception-1:8080",
									},
									{
										Proxy: "socks5://repeater-reception-2:8080",
									},
								},
								NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://repeater-navigation-1:8080",
									},
									{
										Proxy: "socks5://repeater-navigation-2:8080",
									},
								},
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://import-navigation-1:8080",
								},
								{
									Proxy: "socks5://import-navigation-2:8080",
								},
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": nil,
				"import": {
					{
						Bind: []string{
							"0.0.0.0:10000",
						},
						Proxy: []string{
							"oname.ons.svc:80",
							"ssh://export:8080?identity_data=export-identity",
							"socks5://export-reception-1:8080",
							"socks5://export-reception-2:8080",
							"socks5://repeater-navigation-1:8080",
							"socks5://repeater-navigation-2:8080",
							"ssh://repeater:8080?identity_data=repeater-identity",
							"socks5://repeater-reception-1:8080",
							"socks5://repeater-reception-2:8080",
							"socks5://import-navigation-1:8080",
							"socks5://import-navigation-2:8080",
						},
					},
				},
				"repeater": nil,
			},
		},
		// 0b11
		{
			name: "3 hubs 0b11 with proxy",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://export-navigation-1:8080",
								},
								{
									Proxy: "socks5://export-navigation-2:8080",
								},
							},
						},
					},
				},
				"repeater": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://repeater-navigation-1:8080",
								},
								{
									Proxy: "socks5://repeater-navigation-2:8080",
								},
							},
						},
						Override: map[string]v1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater:8080",
								NavigationProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://repeater-navigation-1:8080",
									},
									{
										Proxy: "socks5://repeater-navigation-2:8080",
									},
								},
								ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://repeater-reception-1:8080",
									},
									{
										Proxy: "socks5://repeater-reception-2:8080",
									},
								},
							},
						},
					},
				},
				"import": {
					Spec: v1alpha2.HubSpec{
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater": {
								Reachable: true,
								Address:   "import:8080",
								ReceptionProxy: []v1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://import-reception-1:8080",
									},
									{
										Proxy: "socks5://import-reception-2:8080",
									},
								},
							},
						},
					},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantHubsChains: map[string][]*Chain{
				"export": {
					{
						Bind: []string{
							"0.0.0.0:10000",
							"ssh://import:8080?identity_data=import-identity",
							"socks5://import-reception-1:8080",
							"socks5://import-reception-2:8080",
							"socks5://repeater-navigation-1:8080",
							"socks5://repeater-navigation-2:8080",
							"ssh://repeater:8080?identity_data=repeater-identity",
							"socks5://repeater-reception-1:8080",
							"socks5://repeater-reception-2:8080",
							"socks5://export-navigation-1:8080",
							"socks5://export-navigation-2:8080",
						},
						Proxy: []string{
							"oname.ons.svc:80",
						},
					},
				},
				"import":   nil,
				"repeater": nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := fakeDataSource{
				hubs: tt.hubs,
			}
			h := &HubsChain{
				getIdentity:   d.GetIdentity,
				getHubGateway: d.GetHubGateway,
			}
			name := fmt.Sprintf("%s-%s-%d-%s-%s-%d-tunnel", destination.Namespace, destination.Name, originPort, origin.Namespace, origin.Name, peerPort)
			gotHubsChains, err := h.Build(name, origin, destination, originPort, peerPort, tt.ways)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildServiceDiscovery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(gotHubsChains, tt.wantHubsChains); diff != "" {
				t.Errorf("BuildServiceDiscovery(): got want + \n%s", diff)
			}
		})
	}
}

type fakeDataSource struct {
	hubs map[string]*v1alpha2.Hub
}

func (f *fakeDataSource) GetIdentity(hubName string) string {
	return fmt.Sprintf("%s-%s", hubName, "identity")
}

func (f *fakeDataSource) GetHubGateway(hubName string, forHub string) v1alpha2.HubSpecGateway {
	hub := f.hubs[hubName]
	if hub.Spec.Override == nil {
		return hub.Spec.Gateway
	}
	if o, ok := hub.Spec.Override[forHub]; ok {
		return o
	}
	return hub.Spec.Gateway
}
