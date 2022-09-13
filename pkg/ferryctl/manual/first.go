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

package manual

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ferryproxy/ferry/pkg/consts"
)

type FirstConfig struct {
	RouteName         string
	ImportHub         string
	ExportHub         string
	Next              string
	Reachable         bool
	BindPort          string
	TunnelAddress     string
	TunnelAuthorized  string
	ExportPort        string
	ExportService     string
	ImportService     string
	PeerTunnelAddress string
}

func First(conf FirstConfig) (next string, err error) {
	tunnelAddress := conf.TunnelAddress
	tunnelAuthorized := conf.TunnelAuthorized
	exportService := conf.ExportService
	exportPort := conf.ExportPort
	importService := conf.ImportService
	bindPort := conf.BindPort
	peerTunnelAddress := conf.PeerTunnelAddress

	args := []string{
		"--first=false",
		"--reachable=" + strconv.FormatBool(!conf.Reachable),
	}

	args = append(args, "--route-name="+conf.RouteName)

	if conf.ExportHub != "" {
		args = append(args, "--export-hub="+conf.ExportHub)
	}
	if conf.ImportHub != "" {
		args = append(args, "--import-hub="+conf.ImportHub)
	}
	if !conf.Reachable {
		args = append(args, "--peer-authorized-data="+tunnelAuthorized)
		tunnelAddress = ""
	}
	args = append(args, "--peer-tunnel-address="+tunnelAddress)

	args = append(args, "--export-service="+exportService)

	args = append(args, "--port="+exportPort)

	if importService == "" {
		importService = fmt.Sprintf("%s-%s.%s", strings.ReplaceAll(exportService, ".", "-"), exportPort, consts.FerryTunnelNamespace)
	}
	args = append(args, "--bind-port="+bindPort)
	args = append(args, "--import-service="+importService)
	args = append(args, "--tunnel-address="+peerTunnelAddress)

	return fmt.Sprintf("ferryctl local manual %s %s\n",
		conf.Next,
		strings.Join(args, " "),
	), nil
}
