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
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/ferryproxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
)

type ClusterInitConfig struct {
	ControlPlaneName          string
	ControlPlaneReachable     bool
	ControlPlaneTunnelAddress string
	FerryControllerImage      string
}

func ClusterInit(ctx context.Context, conf ClusterInitConfig) error {
	kctl := kubectl.NewKubectl()

	fmt.Fprintf(os.Stderr, "ferry controller image: %s\n", conf.FerryControllerImage)
	ferry, err := BuildInitFerry(BuildInitFerryConfig{
		Image: conf.FerryControllerImage,
	})
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, strings.NewReader(ferry))
	if err != nil {
		return err
	}

	apiserver := "kubernetes.default.svc:443"

	kubeconfig, err := data_plane.GetKubeconfig(ctx, apiserver)
	if err != nil {
		return err
	}
	hub, err := data_plane.BuildHub(data_plane.BuildHubConfig{
		DataPlaneName:          conf.ControlPlaneName,
		DataPlaneReachable:     conf.ControlPlaneReachable,
		DataPlaneTunnelAddress: conf.ControlPlaneTunnelAddress,
		DataPlaneKubeconfig:    base64.StdEncoding.EncodeToString(kubeconfig),
	})
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, strings.NewReader(hub))
	if err != nil {
		return err
	}
	return nil
}
