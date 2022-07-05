package control_plane

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/setup_steps/first"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/setup_steps/second"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
)

type ShowJoinWithTunnelConfig struct {
	ControlPlaneName          string
	DataPlaneName             string
	ControlPlaneTunnelAddress string
	DataPlaneIdentity         string
	DataPlaneAuthorized       string
	DataPlaneHostkey          string
}

func ShowJoinWithTunnel(ctx context.Context, conf ShowJoinWithTunnelConfig) error {
	kctl := kubectl.NewKubectl()

	port, err := kctl.GetUnusedPort(ctx)
	if err != nil {
		return err
	}

	joinService, err := first.BuildPreJoinTunnel(first.BuildPreJoinTunnelConfig{
		DataPlaneName:          conf.DataPlaneName,
		DataPlaneApiserverPort: port,
	})
	if err != nil {
		return err
	}

	err = kctl.ApplyWithReader(ctx, strings.NewReader(joinService))
	if err != nil {
		return err
	}

	controlPlaneIdentity, err := kctl.GetSecretIdentity(ctx, vars.FerryNamespace, conf.ControlPlaneName)
	if err != nil {
		return err
	}

	data, err := second.BuildJoin(second.BuildJoinConfig{
		ControlPlaneTunnelAddress: conf.ControlPlaneTunnelAddress,
		DataPlaneApiserverPort:    port,
		ControlPlaneIdentity:      controlPlaneIdentity,
		DataPlaneIdentity:         conf.DataPlaneIdentity,
		DataPlaneAuthorized:       conf.DataPlaneAuthorized,
		DataPlaneHostkey:          conf.DataPlaneHostkey,
	})
	if err != nil {
		return err
	}

	fmt.Printf("# ++++ Please run the following command to join the %s data cluster:\n", conf.DataPlaneName)
	fmt.Printf("# =============================================\n")
	fmt.Printf("ferryctl data-plane init\n")
	fmt.Printf("echo %s | base64 --decode | kubectl apply -f -\n", base64.StdEncoding.EncodeToString([]byte(data)))
	fmt.Printf("ferryctl data-plane join tunnel %s\n",
		conf.DataPlaneName,
	)
	fmt.Printf("# =============================================\n")
	return nil
}
