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

package data_plane

import (
	"context"
	"encoding/base64"
	"fmt"
)

type ShowJoinDoneConfig struct {
	ControlPlaneName          string
	DataPlaneName             string
	DataPlaneReachable        bool
	DataPlaneApiserverAddress string
	DataPlaneTunnelAddress    string
	DataPlaneNavigationWay    []string
	DataPlaneReceptionWay     []string
	DataPlaneNavigationProxy  []string
	DataPlaneReceptionProxy   []string
}

func ShowJoinDone(ctx context.Context, conf ShowJoinDoneConfig) (next string, err error) {
	kubeconfig, err := GetKubeconfig(ctx, conf.DataPlaneApiserverAddress)
	if err != nil {
		return "", err
	}
	ci, err := BuildHub(BuildHubConfig{
		DataPlaneName:            conf.DataPlaneName,
		DataPlaneReachable:       conf.DataPlaneReachable,
		DataPlaneTunnelAddress:   conf.DataPlaneTunnelAddress,
		DataPlaneNavigationWay:   conf.DataPlaneNavigationWay,
		DataPlaneReceptionWay:    conf.DataPlaneReceptionWay,
		DataPlaneNavigationProxy: conf.DataPlaneNavigationProxy,
		DataPlaneReceptionProxy:  conf.DataPlaneReceptionProxy,
		DataPlaneKubeconfig:      base64.StdEncoding.EncodeToString(kubeconfig),
	})
	if err != nil {
		return "", err
	}

	baseCmd := base64.StdEncoding.EncodeToString([]byte(ci))

	return fmt.Sprintf("echo %s | base64 --decode | kubectl apply -f -", baseCmd), nil
}
