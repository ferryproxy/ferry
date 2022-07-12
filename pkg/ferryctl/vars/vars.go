package vars

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/ferryproxy/ferry/pkg/utils/env"
)

var (
	ControlPlaneName = "control-plane"
	home, _          = os.UserHomeDir()
	KubeconfigPath   = env.GetEnv("KUBECONFIG", filepath.Join(home, ".kube/config"))

	FerryImagePrefix = env.GetEnv("FERRY_IMAGE_PREFIX", "ghcr.io/ferryproxy/ferry")

	FerryVersion = env.GetEnv("FERRY_VERSION", "v0.3.0")

	FerryControllerImage = env.GetEnv("FERRY_CONTROLLER_IMAGE", FerryImagePrefix+"/ferry-controller:"+FerryVersion)

	FerryTunnelImage = env.GetEnv("FERRY_TUNNEL_IMAGE", FerryImagePrefix+"/ferry-tunnel:"+FerryVersion)

	AutoPlaceholders = "AUTO"
)
