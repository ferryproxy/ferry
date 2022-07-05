package data_plane

import (
	"context"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
)

func GetKubeconfig(ctx context.Context, apiserverAddress string) ([]byte, error) {
	kctl := kubectl.NewKubectl()
	err := kctl.ApplyWithReader(ctx, strings.NewReader(joinRBACYaml))
	if err != nil {
		return nil, err
	}

	kubeconfig, err := kctl.GetKubeconfig(ctx, apiserverAddress)
	if err != nil {
		return nil, err
	}
	return []byte(kubeconfig), nil
}
