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
}

func ShowJoin(ctx context.Context, conf ShowJoinConfig) (next string, err error) {

	args := []string{}
	args = append(args, "--data-plane-tunnel-address="+conf.DataPlaneTunnelAddress)
	args = append(args, "--data-plane-apiserver-address="+conf.DataPlaneApiserverAddress)
	args = append(args, "--control-plane-hub-name="+conf.ControlPlaneName)

	if !conf.ControlPlaneReachable {
		args = append(args, "--data-plane-navigation-hub-name="+conf.DataPlaneName)
		args = append(args, "--data-plane-reception-hub-name="+conf.DataPlaneName)
	} else if !conf.DataPlaneReachable {
		args = append(args, "--data-plane-navigation-hub-name="+conf.ControlPlaneName)
		args = append(args, "--data-plane-reception-hub-name="+conf.ControlPlaneName)
		args = append(args, "--data-plane-reachable=false")
	}

	return fmt.Sprintf("ferryctl data-plane join %s %s\n",
		conf.DataPlaneName,
		strings.Join(args, " "),
	), nil
}
