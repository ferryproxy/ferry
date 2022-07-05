package control_plane

import (
	"context"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
)

type ClusterPreJoinConfig struct {
	DataPlaneName       string
	DataPlaneIdentity   string
	DataPlaneAuthorized string
	DataPlaneHostkey    string
}

func ClusterPreJoin(ctx context.Context, conf ClusterPreJoinConfig) error {
	kctl := kubectl.NewKubectl()

	initKey, err := BuildInitKey(BuildInitKeyConfig{
		ClusterName: conf.DataPlaneName,
		Identity:    conf.DataPlaneIdentity,
		Authorized:  conf.DataPlaneAuthorized,
		Hostkey:     conf.DataPlaneHostkey,
	})
	if err != nil {
		return err
	}

	err = kctl.ApplyWithReader(ctx, strings.NewReader(initKey))
	if err != nil {
		return err
	}
	return nil
}
