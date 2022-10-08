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

	trafficv1alpha2 "github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/utils/objref"
	"github.com/google/go-cmp/cmp"
	"github.com/wzshiming/sshproxy/permissions"
)

func TestHubsChain_Build(t *testing.T) {
	origin := objref.ObjectRef{Name: "oname", Namespace: "ons"}
	destination := objref.ObjectRef{Name: "dname", Namespace: "dns"}
	const (
		originPort = 80
		peerPort   = 10000
	)

	tests := []struct {
		name      string
		hubs      map[string]*trafficv1alpha2.Hub
		ways      []string
		wantBound map[string]*Bound
		wantErr   bool
	}{

		// the 0x0 is can reachable to export, the 0x1 is not

		// 2 hubs (export, import)
		// 0b0
		{
			name: "2 hubs 0b0",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "export:8080",
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"import",
			},
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80",
								},
							},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
							},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://import@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
							},
						},
					},
				},
			},
		},
		// 0b1
		{
			name: "2 hubs 0b1",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
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
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
								"ssh://export@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
							},
							Proxy: []string{
								"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"export": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
			},
		},

		// 3 hubs (export, repeater, import)
		// 0b10
		{
			name: "3 hubs 0b10",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"repeater": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "repeater:8080",
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{
							Bind: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://export@repeater:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater",
							},
							Proxy: []string{
								"oname.ons.svc:80",
							},
						},
					},
					Inbound: nil,
				},
				"import": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
							},
							Proxy: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://import@repeater:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater",
							},
						},
					},
					Inbound: nil,
				},
				"repeater": {
					Inbound: map[string]*AllowList{
						"export": {
							StreamlocalForward: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								},
							},
						},
						"import": {
							DirectStreamlocal: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								},
							},
						},
					},
				},
			},
		},
		// 0b01
		{
			name: "3 hubs 0b01",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "export:8080",
						},
					},
				},
				"repeater": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
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
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"repeater": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80",
								},
							},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"repeater": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
				"repeater": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
								"ssh://repeater@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
							},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://repeater@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
							},
						},
					},
				},
			},
		},
		// 0b00
		{
			name: "3 hubs 0b00",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater:8080",
							},
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater",
				"import",
			},
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80",
								},
							},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
							},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://import@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
								"ssh://import@repeater:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater",
							},
						},
					},
				},
				"repeater": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"export:8080"},
							},
						},
					},
				},
			},
		},
		// 0b11
		{
			name: "3 hubs 0b11",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"repeater": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater:8080",
							},
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
							"ssh://export@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
							"ssh://export@repeater:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater",
						}, Proxy: []string{
							"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"export": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
				"repeater": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"export": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"import:8080"},
							},
						},
					},
				},
			},
		},

		// 4 hubs (export, repeater-export, repeater-import, import)
		// 0b000
		{
			name: "4 hubs 0b000",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater-export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80"},
							},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
							},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://import@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
								"ssh://import@repeater-export:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-export",
								"ssh://import@repeater-import:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-import",
							},
						},
					},
				},
				"repeater-export": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"export:8080"},
							},
						},
					},
				},
				"repeater-import": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"repeater-export:8080"},
							},
						},
					},
				},
			},
		},
		// 0b111
		{
			name: "4 hubs 0b111",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"repeater-export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
							"ssh://export@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
							"ssh://export@repeater-import:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-import",
							"ssh://export@repeater-export:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-export",
						},
							Proxy: []string{
								"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"export": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
				"repeater-export": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"export": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"repeater-import:8080"},
							},
						},
					},
				},
				"repeater-import": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"export": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"import:8080"},
							},
						},
					},
				},
			},
		},
		// 0b100
		{
			name: "4 hubs 0b100",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"repeater-export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://export@repeater-export:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-export",
						},
							Proxy: []string{
								"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
						},
							Proxy: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://import@repeater-export:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-export",
								"ssh://import@repeater-import:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-import",
							},
						},
					},
				},
				"repeater-export": {
					Inbound: map[string]*AllowList{
						"export": {
							StreamlocalForward: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
						"import": {
							DirectStreamlocal: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
					},
				},
				"repeater-import": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"repeater-export:8080"},
							},
						},
					},
				},
			},
		},
		// 0b011
		{
			name: "4 hubs 0b011",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater-export": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"repeater-import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"repeater-export": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80"},
							},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"repeater-export": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
				"repeater-export": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
							"ssh://repeater-export@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
							"ssh://repeater-export@repeater-import:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-import",
						},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://repeater-export@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
							},
						},
					},
				},
				"repeater-import": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"repeater-export": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"import:8080"},
							},
						},
					},
				},
			},
		},
		// 0b110
		{
			name: "4 hubs 0b110",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"repeater-export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
					Spec: trafficv1alpha2.HubSpec{},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{
							Bind: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://export@repeater-import:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-import",
								"ssh://export@repeater-export:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-export",
							},
							Proxy: []string{
								"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
						},
							Proxy: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://import@repeater-import:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-import",
							},
						},
					},
				},
				"repeater-export": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"export": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"repeater-import:8080"},
							},
						},
					},
				},
				"repeater-import": {
					Inbound: map[string]*AllowList{
						"export": {
							StreamlocalForward: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
						"import": {
							DirectStreamlocal: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
					},
				},
			},
		},
		// 0b001
		{
			name: "4 hubs 0b001",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater-export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "repeater-export:8080",
							},
						},
					},
				},
				"repeater-import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "repeater-import:8080",
							},
						},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"repeater-import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80"},
							},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"repeater-import": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
				"repeater-export": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"repeater-import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"export:8080"},
							},
						},
					},
				},
				"repeater-import": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
							"ssh://repeater-import@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
						},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://repeater-import@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
								"ssh://repeater-import@repeater-export:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-export",
							},
						},
					},
				},
			},
		},
		// 0b101
		{
			name: "4 hubs 0b101",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{},
				},
				"repeater-export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{},
					},
				},
				"import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://export@repeater-export:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-export",
						},
							Proxy: []string{
								"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"repeater-import": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
				"repeater-export": {
					Inbound: map[string]*AllowList{
						"export": {
							StreamlocalForward: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
						"repeater-import": {
							DirectStreamlocal: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
					},
				},
				"repeater-import": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
								"ssh://repeater-import@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
							},
							Proxy: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://repeater-import@repeater-export:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-export",
							},
						},
					},
				},
			},
		},
		// 0b010
		{
			name: "4 hubs 0b010",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater-export": {
								Reachable: true,
								Address:   "export:8080",
							},
						},
					},
				},
				"repeater-export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{},
					},
				},
				"repeater-import": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
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
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{},
					},
				},
			},
			ways: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"repeater-export": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80"},
							},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
							},
							Proxy: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://import@repeater-import:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-import",
							},
						},
					},
				},
				"repeater-export": {
					Outbound: []*Chain{
						{
							Bind: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://repeater-export@repeater-import:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater-import",
							},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://repeater-export@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
							},
						},
					},
				},
				"repeater-import": {
					Inbound: map[string]*AllowList{
						"import": {
							DirectStreamlocal: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
						"repeater-export": {
							StreamlocalForward: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
					},
				},
			},
		},

		// 2 hubs with proxy (export, import)
		// 0b0
		{
			name: "2 hubs 0b0 with proxy",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "export:8080",
							ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80"},
							},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
							},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://import@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
								"socks5://export-reception-1:8080",
								"socks5://export-reception-2:8080",
								"socks5://import-navigation-1:8080",
								"socks5://import-navigation-2:8080",
							},
						},
					},
				},
			},
		},
		// 0b1
		{
			name: "2 hubs 0b1 with proxy",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "import:8080",
							ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
							"ssh://export@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
							"socks5://import-reception-1:8080", "socks5://import-reception-2:8080",
							"socks5://export-navigation-1:8080", "socks5://export-navigation-2:8080",
						},
							Proxy: []string{
								"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"export": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
			},
		},

		// 3 hubs with proxy (export, repeater, import)
		// 0b10
		{
			name: "3 hubs 0b10 with proxy",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "repeater:8080",
							ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{Bind: []string{
							"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
							"ssh://export@repeater:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater",
							"socks5://repeater-reception-1:8080", "socks5://repeater-reception-2:8080",
							"socks5://export-navigation-1:8080", "socks5://export-navigation-2:8080",
						},
							Proxy: []string{
								"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
						},
							Proxy: []string{
								"unix:///dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks",
								"ssh://import@repeater:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater",
								"socks5://repeater-reception-1:8080", "socks5://repeater-reception-2:8080",
								"socks5://import-navigation-1:8080", "socks5://import-navigation-2:8080",
							},
						},
					},
				},
				"repeater": {
					Inbound: map[string]*AllowList{
						"export": {
							StreamlocalForward: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
						"import": {
							DirectStreamlocal: permissions.Permission{
								Allows: []string{
									"/dev/shm/dns-dname-80-ons-oname-10000-tunnel.socks"},
							},
						},
					},
				},
			},
		},
		// 0b01
		{
			name: "3 hubs 0b01 with proxy",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "export:8080",
							ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							Reachable: true,
							Address:   "import:8080",
							ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"repeater": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80"},
							},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"repeater": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
				"repeater": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
								"ssh://repeater@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
								"socks5://import-reception-1:8080",
								"socks5://import-reception-2:8080",
								"socks5://repeater-navigation-1:8080",
								"socks5://repeater-navigation-2:8080",
							},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://repeater@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
								"socks5://export-reception-1:8080",
								"socks5://export-reception-2:8080",
								"socks5://repeater-navigation-1:8080",
								"socks5://repeater-navigation-2:8080",
							},
						},
					},
				},
			},
		},
		// 0b00
		{
			name: "3 hubs 0b00 with proxy",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater": {
								Reachable: true,
								Address:   "export:8080",
								ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://repeater-navigation-1:8080",
								},
								{
									Proxy: "socks5://repeater-navigation-2:8080",
								},
							},
						},
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"import": {
								Reachable: true,
								Address:   "repeater:8080",
								ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://repeater-reception-1:8080",
									},
									{
										Proxy: "socks5://repeater-reception-2:8080",
									},
								},
								NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
			wantBound: map[string]*Bound{
				"export": {
					Inbound: map[string]*AllowList{
						"import": {
							DirectTcpip: permissions.Permission{
								Allows: []string{
									"oname.ons.svc:80"},
							},
						},
					},
				},
				"import": {
					Outbound: []*Chain{
						{
							Bind: []string{
								":10000",
							},
							Proxy: []string{
								"oname.ons.svc:80",
								"ssh://import@export:8080?identity_file=/var/ferry/ssh/identity&target_hub=export",
								"socks5://export-reception-1:8080",
								"socks5://export-reception-2:8080",
								"socks5://repeater-navigation-1:8080",
								"socks5://repeater-navigation-2:8080",
								"ssh://import@repeater:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater",
								"socks5://repeater-reception-1:8080",
								"socks5://repeater-reception-2:8080",
								"socks5://import-navigation-1:8080",
								"socks5://import-navigation-2:8080",
							},
						},
					},
				},
				"repeater": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"import": {},
					},
				},
			},
		},
		// 0b11
		{
			name: "3 hubs 0b11 with proxy",
			hubs: map[string]*trafficv1alpha2.Hub{
				"export": {
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Gateway: trafficv1alpha2.HubSpecGateway{
							NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
								{
									Proxy: "socks5://repeater-navigation-1:8080",
								},
								{
									Proxy: "socks5://repeater-navigation-2:8080",
								},
							},
						},
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"export": {
								Reachable: true,
								Address:   "repeater:8080",
								NavigationProxy: []trafficv1alpha2.HubSpecGatewayProxy{
									{
										Proxy: "socks5://repeater-navigation-1:8080",
									},
									{
										Proxy: "socks5://repeater-navigation-2:8080",
									},
								},
								ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
					Spec: trafficv1alpha2.HubSpec{
						Override: map[string]trafficv1alpha2.HubSpecGateway{
							"repeater": {
								Reachable: true,
								Address:   "import:8080",
								ReceptionProxy: []trafficv1alpha2.HubSpecGatewayProxy{
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
			wantBound: map[string]*Bound{
				"export": {
					Outbound: []*Chain{
						{Bind: []string{
							":10000",
							"ssh://export@import:8080?identity_file=/var/ferry/ssh/identity&target_hub=import",
							"socks5://import-reception-1:8080",
							"socks5://import-reception-2:8080",
							"socks5://repeater-navigation-1:8080",
							"socks5://repeater-navigation-2:8080",
							"ssh://export@repeater:8080?identity_file=/var/ferry/ssh/identity&target_hub=repeater",
							"socks5://repeater-reception-1:8080",
							"socks5://repeater-reception-2:8080",
							"socks5://export-navigation-1:8080",
							"socks5://export-navigation-2:8080",
						},
							Proxy: []string{
								"oname.ons.svc:80"},
						},
					},
				},
				"import": {
					Inbound: map[string]*AllowList{
						"export": {
							TcpipForward: permissions.Permission{
								Allows: []string{
									":10000",
								},
							},
						},
					},
				},
				"repeater": {
					Outbound: []*Chain{},
					Inbound: map[string]*AllowList{
						"export": {},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := fakeDataSource{
				hubs: tt.hubs,
			}
			h := &HubsChain{
				getHubGateway: d.GetHubGateway,
			}
			name := fmt.Sprintf("%s-%s-%d-%s-%s-%d-tunnel", destination.Namespace, destination.Name, originPort, origin.Namespace, origin.Name, peerPort)
			gotBound, err := h.Build(name, origin, destination, originPort, peerPort, tt.ways)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildServiceDiscovery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(gotBound, tt.wantBound); diff != "" {
				t.Errorf("BuildServiceDiscovery(): got want + \n%s", diff)
				fmt.Printf("====\n%#3v\n====\n", gotBound)
			}
		})
	}
}

type fakeDataSource struct {
	hubs map[string]*trafficv1alpha2.Hub
}

func (f *fakeDataSource) GetAuthorized(hubName string) string {
	return fmt.Sprintf("%s-%s", hubName, "authorized")
}

func (f *fakeDataSource) GetHubGateway(hubName string, forHub string) trafficv1alpha2.HubSpecGateway {
	hub := f.hubs[hubName]
	if hub.Spec.Override == nil {
		return hub.Spec.Gateway
	}
	if o, ok := hub.Spec.Override[forHub]; ok {
		return o
	}
	return hub.Spec.Gateway
}
