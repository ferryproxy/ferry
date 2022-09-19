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
	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ManualConfig struct {
	RouteName        string
	ExportHubName    string
	ExportName       string
	ExportNamespace  string
	ExportAuthorized string
	ExportGateway    v1alpha2.HubSpecGateway
	ImportHubName    string
	ImportName       string
	ImportNamespace  string
	ImportAuthorized string
	ImportGateway    v1alpha2.HubSpecGateway
	Port             int32
	BindPort         int32
}

type Manual struct {
	dateSource dateSource
}

func NewManual(conf ManualConfig) *Manual {
	return &Manual{
		dateSource: dateSource{
			routeName:        conf.RouteName,
			exportHubName:    conf.ExportHubName,
			exportName:       conf.ExportName,
			exportNamespace:  conf.ExportNamespace,
			exportGateway:    conf.ExportGateway,
			exportAuthorized: conf.ExportAuthorized,
			importHubName:    conf.ImportHubName,
			importName:       conf.ImportName,
			importNamespace:  conf.ImportNamespace,
			importGateway:    conf.ImportGateway,
			importAuthorized: conf.ImportAuthorized,
			port:             conf.Port,
			bindPort:         conf.BindPort,
		},
	}
}

type dateSource struct {
	routeName        string
	exportHubName    string
	exportName       string
	exportNamespace  string
	exportAuthorized string
	exportGateway    v1alpha2.HubSpecGateway
	importHubName    string
	importName       string
	importNamespace  string
	importAuthorized string
	importGateway    v1alpha2.HubSpecGateway
	port             int32
	bindPort         int32
}

func (f *dateSource) GetPortPeer(importHubName string, cluster, namespace, name string, port int32) (int32, error) {
	return f.bindPort, nil
}

func (f *dateSource) ListServices(name string) []*corev1.Service {
	if name != f.exportHubName {
		return nil
	}
	svc := &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{
			Name:      f.exportName,
			Namespace: f.exportNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     f.port,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
	return []*corev1.Service{
		svc,
	}
}
func (f *dateSource) GetHubGateway(hubName string, forHub string) v1alpha2.HubSpecGateway {
	if hubName == f.importHubName {
		return f.importGateway
	} else if hubName == f.exportHubName {
		return f.exportGateway
	}
	return v1alpha2.HubSpecGateway{}
}

func (f *dateSource) GetAuthorized(name string) string {
	if name == f.importHubName {
		return f.importAuthorized
	} else if name == f.exportHubName {
		return f.exportAuthorized
	}
	return ""
}

func (f *Manual) BuildResource() (out map[string][]resource.Resourcer, err error) {
	solution := NewSolution(SolutionConfig{
		GetHubGateway: f.dateSource.GetHubGateway,
	})

	ways, err := solution.CalculateWays(f.dateSource.exportHubName, f.dateSource.importHubName)
	if err != nil {
		return nil, err
	}

	router := NewRouter(RouterConfig{
		Labels:        map[string]string{},
		ExportHubName: f.dateSource.exportHubName,
		ImportHubName: f.dateSource.importHubName,
		HubInterface:  &f.dateSource,
	})

	routes := []*v1alpha2.Route{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: f.dateSource.routeName,
			},
			Spec: v1alpha2.RouteSpec{
				Import: v1alpha2.RouteSpecRule{
					HubName: f.dateSource.importHubName,
					Service: v1alpha2.RouteSpecRuleService{
						Name:      f.dateSource.importName,
						Namespace: f.dateSource.importNamespace,
					},
				},
				Export: v1alpha2.RouteSpecRule{
					HubName: f.dateSource.exportHubName,
					Service: v1alpha2.RouteSpecRuleService{
						Name:      f.dateSource.exportName,
						Namespace: f.dateSource.exportNamespace,
					},
				},
			},
		},
	}

	return router.BuildResource(routes, ways)
}
