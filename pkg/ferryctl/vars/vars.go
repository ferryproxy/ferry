/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vars

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/utils/env"
)

var (
	ControlPlaneName   = consts.ControlPlaneName
	home, _            = os.UserHomeDir()
	KubeconfigPath     = env.GetEnv("KUBECONFIG", filepath.Join(home, ".kube/config"))
	PeerKubeconfigPath = env.GetEnv("FERRY_PEER_KUBECONFIG", "")

	FerryImagePrefix = env.GetEnv("FERRY_IMAGE_PREFIX", consts.ImagePrefix)

	FerryVersion = env.GetEnv("FERRY_VERSION", consts.Version)

	FerryControllerImage = env.GetEnv("FERRY_CONTROLLER_IMAGE", FerryImagePrefix+"/ferry-controller:"+FerryVersion)

	FerryTunnelImage = env.GetEnv("FERRY_TUNNEL_IMAGE", FerryImagePrefix+"/ferry-tunnel:"+FerryVersion)

	FerryRegisterImage = env.GetEnv("FERRY_REGISTER_IMAGE", FerryImagePrefix+"/ferry-register:"+FerryVersion)

	FerryJoinerImage = env.GetEnv("FERRY_JOINER_IMAGE", FerryImagePrefix+"/ferry-joiner:"+FerryVersion)

	AutoPlaceholders = "AUTO"
)
