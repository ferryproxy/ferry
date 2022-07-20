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
	"context"
	"fmt"
	"strconv"
	"strings"
)

type FirstConfig struct {
	Next              string
	Reachable         bool
	BindPort          string
	TunnelAddress     string
	TunnelIdentity    string
	ExportPort        string
	ExportHost        string
	ImportServiceName string
	PeerTunnelAddress string
}

func First(ctx context.Context, conf FirstConfig) (next string, err error) {
	tunnelAddress := conf.TunnelAddress
	tunnelIdentity := conf.TunnelIdentity
	exportHost := conf.ExportHost
	exportPort := conf.ExportPort
	importServiceName := conf.ImportServiceName
	bindPort := conf.BindPort
	peerTunnelAddress := conf.PeerTunnelAddress

	args := []string{
		"--first=false",
		"--reachable=" + strconv.FormatBool(!conf.Reachable),
	}
	if conf.Reachable {
		args = append(args, "--peer-identity-data="+tunnelIdentity)
	} else {
		tunnelAddress = ""
	}
	args = append(args, "--peer-tunnel-address="+tunnelAddress)

	args = append(args, "--export-host-port="+exportHost+":"+exportPort)

	if importServiceName == "" {
		importServiceName = fmt.Sprintf("%s-%s", strings.ReplaceAll(exportHost, ".", "-"), exportPort)
	}
	args = append(args, "--bind-port="+bindPort)
	args = append(args, "--import-service-name="+importServiceName)
	args = append(args, "--tunnel-address="+peerTunnelAddress)

	return fmt.Sprintf("ferryctl local manual %s %s\n",
		conf.Next,
		strings.Join(args, " "),
	), nil
}
