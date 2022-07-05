package control_plane

import (
	_ "embed"
)

//go:generate kubectl kustomize -o init_crd.yaml https://github.com/ferry-proxy/api/config/crd

//go:embed init_crd.yaml
var crdYaml string
