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
