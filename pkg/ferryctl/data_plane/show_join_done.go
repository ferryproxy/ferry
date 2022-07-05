package data_plane

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/setup_steps/third"
)

type ShowJoinDoneConfig struct {
	ControlPlaneName               string
	DataPlaneName                  string
	DataPlaneReachable             bool
	DataPlaneApiserverAddress      string
	DataPlaneTunnelAddress         string
	DataPlaneNavigationClusterName string
	DataPlaneReceptionClusterName  string
}

func ShowJoinDone(ctx context.Context, conf ShowJoinDoneConfig) error {
	kubeconfig, err := GetKubeconfig(ctx, conf.DataPlaneApiserverAddress)
	if err != nil {
		return err
	}
	ci, err := third.BuildClusterInformation(third.BuildClusterInformationConfig{
		DataPlaneName:                  conf.DataPlaneName,
		DataPlaneReachable:             conf.DataPlaneReachable,
		DataPlaneTunnelAddress:         conf.DataPlaneTunnelAddress,
		DataPlaneNavigationClusterName: conf.DataPlaneNavigationClusterName,
		DataPlaneReceptionClusterName:  conf.DataPlaneReceptionClusterName,
		DataPlaneKubeconfig:            kubeconfig,
	})
	if err != nil {
		return err
	}

	baseCmd := base64.StdEncoding.EncodeToString([]byte(ci))

	fmt.Printf("# ++++ Seccussfully generated control kubeconfig for %s\n", conf.DataPlaneName)
	fmt.Printf("# ++++ Please run the following command to join the %s cluster:\n", conf.ControlPlaneName)
	fmt.Printf("# Apiserver: %s\n", conf.DataPlaneApiserverAddress)
	if conf.DataPlaneTunnelAddress != "" {
		fmt.Printf("# Tunnel: %s\n", conf.DataPlaneTunnelAddress)
	}
	if conf.DataPlaneNavigationClusterName != "" {
		fmt.Printf("# Proxy: %s\n", conf.DataPlaneNavigationClusterName)
	}
	fmt.Printf("# =============================================\n")
	fmt.Printf("echo %s | base64 --decode | kubectl apply -f -\n", baseCmd)
	fmt.Printf("# =============================================\n")
	return nil
}
