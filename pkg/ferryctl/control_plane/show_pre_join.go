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

package control_plane

import (
	"context"
	"fmt"
	"strings"
)

type ShowJoinConfig struct {
	DataPlaneName             string
	DataPlaneTunnelAddress    string
	DataPlaneApiserverAddress string
	DataPlaneReachable        bool

	ControlPlaneName          string
	ControlPlaneTunnelAddress string
	ControlPlaneReachable     bool

	DataPlaneNavigationWay   []string
	DataPlaneReceptionWay    []string
	DataPlaneNavigationProxy []string
	DataPlaneReceptionProxy  []string
}

func ShowJoin(ctx context.Context, conf ShowJoinConfig) (next string, err error) {

	args := []string{}
	args = append(args, "--data-plane-tunnel-address="+conf.DataPlaneTunnelAddress)
	args = append(args, "--data-plane-apiserver-address="+conf.DataPlaneApiserverAddress)
	args = append(args, "--control-plane-hub-name="+conf.ControlPlaneName)

	if !conf.DataPlaneReachable {
		args = append(args, "--data-plane-reachable=false")
	}

	if len(conf.DataPlaneNavigationWay) > 0 {
		args = append(args, "--data-plane-navigation-way="+strings.Join(conf.DataPlaneNavigationWay, ","))
	}
	if len(conf.DataPlaneReceptionWay) > 0 {
		args = append(args, "--data-plane-reception-way="+strings.Join(conf.DataPlaneReceptionWay, ","))
	}
	if len(conf.DataPlaneNavigationProxy) > 0 {
		args = append(args, "--data-plane-navigation-proxy="+strings.Join(conf.DataPlaneNavigationProxy, ","))
	}
	if len(conf.DataPlaneReceptionProxy) > 0 {
		args = append(args, "--data-plane-reception-proxy="+strings.Join(conf.DataPlaneReceptionProxy, ","))
	}

	return fmt.Sprintf("ferryctl data-plane join %s %s\n",
		conf.DataPlaneName,
		strings.Join(args, " "),
	), nil
}
