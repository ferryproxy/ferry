package control_plane

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
)

type ClusterInitConfig struct {
	ControlPlaneName          string
	ControlPlaneReachable     bool
	ControlPlaneTunnelAddress string
	FerryControllerImage      string
}

func ClusterInit(ctx context.Context, conf ClusterInitConfig) error {
	kctl := kubectl.NewKubectl()
	err := kctl.ApplyWithReader(ctx, strings.NewReader(crdYaml))
	if err != nil {
		return err
	}

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

	apiserver, err := kctl.GetApiserverAddress(ctx)
	if err != nil {
		return err
	}

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
