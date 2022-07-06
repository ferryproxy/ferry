package control_plane

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/data_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/setup_steps/second"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/setup_steps/third"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
)

type ClusterInitConfig struct {
	ControlPlaneName          string
	ControlPlaneReachable     bool
	ControlPlaneTunnelAddress string
}

func ClusterInit(ctx context.Context, conf ClusterInitConfig) error {
	kctl := kubectl.NewKubectl()
	err := kctl.ApplyWithReader(ctx, strings.NewReader(crdYaml))
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "ferry controller image: %s\n", vars.FerryControllerImage)
	ferry, err := BuildInitFerry(BuildInitFerryConfig{
		Image: vars.FerryControllerImage,
	})
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, strings.NewReader(ferry))
	if err != nil {
		return err
	}

	err = data_plane.ClusterInit(ctx)
	if err != nil {
		return err
	}

	identity, authorized, err := utils.GetKey()
	if err != nil {
		return err
	}

	data, err := second.BuildJoin(second.BuildJoinConfig{
		DataPlaneIdentity:   identity,
		DataPlaneAuthorized: authorized,
		DataPlaneHostkey:    identity,
	})
	if err != nil {
		return err
	}

	err = kctl.ApplyWithReader(ctx, strings.NewReader(data))
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
	ci, err := third.BuildHub(third.BuildHubConfig{
		DataPlaneName:          conf.ControlPlaneName,
		DataPlaneReachable:     conf.ControlPlaneReachable,
		DataPlaneTunnelAddress: conf.ControlPlaneTunnelAddress,
		DataPlaneKubeconfig:    kubeconfig,
	})
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, strings.NewReader(ci))
	if err != nil {
		return err
	}
	return nil
}
