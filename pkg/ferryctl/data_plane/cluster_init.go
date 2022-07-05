package data_plane

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
)

func ClusterInit(ctx context.Context) error {
	kctl := kubectl.NewKubectl()
	fmt.Fprintf(os.Stderr, "ferry tunnel image: %s\n", vars.FerryTunnelImage)
	tunnel, err := BuildInitTunnel(BuildInitTunnelConfig{
		Image: vars.FerryTunnelImage,
	})
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, strings.NewReader(tunnel))
	if err != nil {
		return err
	}
	return nil
}
