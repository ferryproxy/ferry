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

	DataPlaneNavigation []string
	DataPlaneReception  []string
}

func ShowJoin(ctx context.Context, conf ShowJoinConfig) (next string, err error) {

	args := []string{}
	args = append(args, "--data-plane-tunnel-address="+conf.DataPlaneTunnelAddress)
	args = append(args, "--data-plane-apiserver-address="+conf.DataPlaneApiserverAddress)
	args = append(args, "--control-plane-hub-name="+conf.ControlPlaneName)

	if !conf.DataPlaneReachable {
		args = append(args, "--data-plane-reachable=false")
	}

	if len(conf.DataPlaneNavigation) > 0 {
		args = append(args, "--data-plane-navigation="+strings.Join(conf.DataPlaneNavigation, ","))
	}
	if len(conf.DataPlaneReception) > 0 {
		args = append(args, "--data-plane-reception="+strings.Join(conf.DataPlaneReception, ","))
	}

	return fmt.Sprintf("ferryctl data-plane join %s %s\n",
		conf.DataPlaneName,
		strings.Join(args, " "),
	), nil
}
