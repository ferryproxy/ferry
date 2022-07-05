package control_plane

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/setup_steps/second"
)

type ShowJoinWithDirectConfig struct {
	DataPlaneName             string
	DataPlaneApiserverAddress string
	DataPlaneTunnelAddress    string
	DataPlaneIdentity         string
	DataPlaneAuthorized       string
	DataPlaneHostkey          string
}

func ShowJoinWithDirect(ctx context.Context, conf ShowJoinWithDirectConfig) error {
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
