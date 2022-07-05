package vars

import (
	_ "embed"
	"github.com/ferry-proxy/ferry/pkg/utils/env"
	"os"
	"path/filepath"
)

var (
	FerryNamespace   = "ferry-system"
	ControlPlaneName = "control-plane"
	home, _          = os.UserHomeDir()
	KubeconfigPath   = env.GetEnv("KUBECONFIG", filepath.Join(home, ".kube/config"))

	//go:embed ferry_controller_image.txt
	ferryControllerImage string
	FerryControllerImage = env.GetEnv("FERRY_CONTROLLER_IMAGE", ferryControllerImage)
	//go:embed ferry_tunnel_image.txt
	ferryTunnelImage string
	FerryTunnelImage = env.GetEnv("FERRY_TUNNEL_IMAGE", ferryTunnelImage)

	AutoPlaceholders = "AUTO"
)
