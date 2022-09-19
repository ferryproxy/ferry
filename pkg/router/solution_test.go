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
	"reflect"
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
							ReceptionWay: []v1alpha2.HubSpecGatewayWay{
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
							NavigationWay: []v1alpha2.HubSpecGatewayWay{
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
							ReceptionWay: []v1alpha2.HubSpecGatewayWay{
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
							NavigationWay: []v1alpha2.HubSpecGatewayWay{
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
							ReceptionWay: []v1alpha2.HubSpecGatewayWay{
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
							NavigationWay: []v1alpha2.HubSpecGatewayWay{
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
							NavigationWay: []v1alpha2.HubSpecGatewayWay{
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
							ReceptionWay: []v1alpha2.HubSpecGatewayWay{
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
							NavigationWay: []v1alpha2.HubSpecGatewayWay{
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
							NavigationWay: []v1alpha2.HubSpecGatewayWay{
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
							NavigationWay: []v1alpha2.HubSpecGatewayWay{
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
							NavigationWay: []v1alpha2.HubSpecGatewayWay{
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

func Test_removeInvalidWays(t *testing.T) {
	type args struct {
		ways []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "0 ways",
			args: args{
				ways: []string{},
			},
			want: []string{},
		},
		{
			name: "1 ways",
			args: args{
				ways: []string{"a"},
			},
			want: []string{"a"},
		},
		{
			name: "2 ways",
			args: args{
				ways: []string{"a", "b"},
			},
			want: []string{"a", "b"},
		},
		{
			name: "3 ways",
			args: args{
				ways: []string{"a", "b", "c"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "4 ways, 1 hit in middle",
			args: args{
				ways: []string{"a", "b", "b", "c"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "4 ways, 1 hit in end",
			args: args{
				ways: []string{"a", "b", "c", "c"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "4 ways, 1 hit in begin",
			args: args{
				ways: []string{"a", "a", "b", "c"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "5 ways, 2 hit in middle",
			args: args{
				ways: []string{"a", "b", "b", "b", "c"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "6 ways, 2 hit in middle",
			args: args{
				ways: []string{"a", "b", "c", "c", "b", "d"},
			},
			want: []string{"a", "b", "d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeInvalidWays(tt.args.ways); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("removeInvalidWays() = %v, want %v", got, tt.want)
			}
		})
	}
}
