package data_plane

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/ferryctl/utils"
)

type ClusterInitConfig struct {
	FerryTunnelImage string
}

func ClusterInit(ctx context.Context, conf ClusterInitConfig) error {
	kctl := kubectl.NewKubectl()
	fmt.Fprintf(os.Stderr, "ferry tunnel image: %s\n", conf.FerryTunnelImage)
	tunnel, err := BuildInitTunnel(BuildInitTunnelConfig{
		Image: conf.FerryTunnelImage,
	})
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, strings.NewReader(tunnel))
	if err != nil {
		return err
	}

	ident, err := kctl.GetSecretIdentity(ctx)
	if err != nil || ident == "" {
		identity, authorized, err := utils.GetKey()
		if err != nil {
			return err
		}
		key, err := BuildInitKey(BuildInitKeyConfig{
			Identity:   identity,
			Authorized: authorized,
			Hostkey:    identity,
		})
		if err != nil {
			return err
		}
		err = kctl.ApplyWithReader(ctx, strings.NewReader(key))
		if err != nil {
			return err
		}
	}
	return nil
}
