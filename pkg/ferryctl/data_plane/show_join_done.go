package data_plane

import (
	"context"
	"encoding/base64"
	"fmt"
)

type ShowJoinDoneConfig struct {
	ControlPlaneName          string
	DataPlaneName             string
	DataPlaneReachable        bool
	DataPlaneApiserverAddress string
	DataPlaneTunnelAddress    string
	DataPlaneNavigationWay    []string
	DataPlaneReceptionWay     []string
	DataPlaneNavigationProxy  []string
	DataPlaneReceptionProxy   []string
}

func ShowJoinDone(ctx context.Context, conf ShowJoinDoneConfig) (next string, err error) {
	kubeconfig, err := GetKubeconfig(ctx, conf.DataPlaneApiserverAddress)
	if err != nil {
		return "", err
	}
	ci, err := BuildHub(BuildHubConfig{
		DataPlaneName:          conf.DataPlaneName,
		DataPlaneReachable:     conf.DataPlaneReachable,
		DataPlaneTunnelAddress: conf.DataPlaneTunnelAddress,
		DataPlaneNavigationWay: conf.DataPlaneNavigationWay,
		DataPlaneReceptionWay:  conf.DataPlaneReceptionWay,
		DataPlaneNavigationProxy:conf.DataPlaneNavigationProxy,
		DataPlaneReceptionProxy:conf.DataPlaneReceptionProxy,
		DataPlaneKubeconfig:    base64.StdEncoding.EncodeToString(kubeconfig),
	})
	if err != nil {
		return "", err
	}

	baseCmd := base64.StdEncoding.EncodeToString([]byte(ci))

	return fmt.Sprintf("echo %s | base64 --decode | kubectl apply -f -", baseCmd), nil
}
