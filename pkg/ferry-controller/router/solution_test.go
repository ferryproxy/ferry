package router

import (
	"testing"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/google/go-cmp/cmp"
)

func TestSolution_Solution(t *testing.T) {

	tests := []struct {
		name    string
		hubs    map[string]*v1alpha2.Hub
		want    []string
		wantErr bool
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
			want: []string{
				"export",
				"import",
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
			want: []string{
				"export",
				"import",
			},
		},

		// 4 hubs (export, repeater-export, repeater-import, import)
		// 0b000
		{
			name: "4 hubs 0b000 export reception",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Reception: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-import",
								},
								{
									HubName: "repeater-export",
								},
							},
						},
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
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},
		{
			name: "4 hubs 0b000 import navigation",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
						},
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
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Navigation: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-export",
								},
								{
									HubName: "repeater-import",
								},
							},
						},
					},
				},
			},
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},
		{
			name: "4 hubs 0b000 import and export",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Reception: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-export",
								},
							},
						},
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
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Navigation: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-import",
								},
							},
						},
					},
				},
			},
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},

		// 0b111
		{
			name: "4 hubs 0b111 import reception",
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
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Reception: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-export",
								},
								{
									HubName: "repeater-import",
								},
							},
						},
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "import:8080",
							},
						},
					},
				},
			},
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},
		{
			name: "4 hubs 0b111 export navigation",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Navigation: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-import",
								},
								{
									HubName: "repeater-export",
								},
							},
						},
					},
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
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
						},
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "import:8080",
							},
						},
					},
				},
			},
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},
		{
			name: "4 hubs 0b111 import and export",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Navigation: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-export",
								},
							},
						},
					},
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
						Gateway: v1alpha2.HubSpecGateway{
							Reachable: true,
							Reception: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-import",
								},
							},
						},
						Override: map[string]v1alpha2.HubSpecGateway{
							"repeater-import": {
								Reachable: true,
								Address:   "import:8080",
							},
						},
					},
				},
			},
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},

		// 0b100
		{
			name: "4 hubs 0b100 export navigation",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Navigation: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-import",
								},
								{
									HubName: "repeater-export",
								},
							},
						},
					},
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
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},
		{
			name: "4 hubs 0b100 import navigation",
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
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Navigation: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-export",
								},
								{
									HubName: "repeater-import",
								},
							},
						},
					},
				},
			},
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},
		{
			name: "4 hubs 0b100 import and export",
			hubs: map[string]*v1alpha2.Hub{
				"export": {
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Navigation: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-export",
								},
							},
						},
					},
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
					Spec: v1alpha2.HubSpec{
						Gateway: v1alpha2.HubSpecGateway{
							Navigation: []v1alpha2.HubSpecGatewayWay{
								{
									HubName: "repeater-import",
								},
							},
						},
					},
				},
			},
			want: []string{
				"export",
				"repeater-export",
				"repeater-import",
				"import",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := fakeDataSource{
				hubs: tt.hubs,
			}
			s := &Solution{
				getHubGateway: d.GetHubGateway,
			}
			got, err := s.CalculateWays("export", "import")
			if (err != nil) != tt.wantErr {
				t.Errorf("CalculateWays() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("CalculateWays(): got want + \n%s", diff)
			}
		})
	}
}
