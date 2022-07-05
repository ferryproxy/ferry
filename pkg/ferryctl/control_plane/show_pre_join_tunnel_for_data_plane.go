package control_plane

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/setup_steps/first"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/setup_steps/second"
)

type ShowJoinWithTunnelForDataPlaneConfig struct {
	DataPlaneName             string
	DataPlaneTunnelAddress    string
	DataPlaneApiserverAddress string
	DataPlaneIdentity         string
	DataPlaneAuthorized       string
	DataPlaneHostkey          string
}

func ShowJoinWithTunnelForDataPlane(ctx context.Context, conf ShowJoinWithTunnelForDataPlaneConfig) error {
	kctl := kubectl.NewKubectl()

	port, err := kctl.GetUnusedPort(ctx)
	if err != nil {
		return err
	}

	joinService, err := first.BuildPreJoinTunnel(first.BuildPreJoinTunnelConfig{
		DataPlaneName:          conf.DataPlaneName,
		DataPlaneApiserverPort: port,
		DataPlaneTunnelAddress: conf.DataPlaneTunnelAddress,
		DataPlaneIdentity:      conf.DataPlaneIdentity,
	})
	if err != nil {
		return err
	}

	err = kctl.ApplyWithReader(ctx, strings.NewReader(joinService))
	if err != nil {
		return err
	}

	data, err := second.BuildJoin(second.BuildJoinConfig{
		DataPlaneIdentity:   conf.DataPlaneIdentity,
		DataPlaneAuthorized: conf.DataPlaneAuthorized,
		DataPlaneHostkey:    conf.DataPlaneHostkey,
	})
	if err != nil {
		return err
	}

	fmt.Printf("# ++++ Please run the following command to join the %s data cluster:\n", conf.DataPlaneName)
	fmt.Printf("# =============================================\n")
	fmt.Printf("ferryctl data-plane init\n")
	fmt.Printf("echo %s | base64 --decode | kubectl apply -f -\n", base64.StdEncoding.EncodeToString([]byte(data)))
	args := []string{}
	args = append(args, "--data-plane-tunnel-address="+conf.DataPlaneTunnelAddress)
	args = append(args, "--data-plane-apiserver-address="+conf.DataPlaneApiserverAddress)

	fmt.Printf("ferryctl data-plane join direct %s %s\n",
		conf.DataPlaneName,
		strings.Join(args, " "),
	)
	fmt.Printf("# =============================================\n")
	return nil
}
