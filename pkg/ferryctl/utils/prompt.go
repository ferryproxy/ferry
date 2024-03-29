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

package utils

import (
	"fmt"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
	"os"
	"os/exec"
)

func Prompt(want string, lines ...string) {
	if vars.PeerKubeconfigPath != "" {
		fmt.Printf("# Run command to %s:\n", want)
		for _, line := range lines {
			cmd := exec.Command("sh", "-c", line)
			cmd.Env = append(os.Environ(),
				"KUBECONFIG="+vars.PeerKubeconfigPath,
				"FERRY_PEER_KUBECONFIG="+vars.KubeconfigPath,
			)
			fmt.Printf("> %s\n", line)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				fmt.Println(cmd)
			}
		}
	} else {
		fmt.Printf("# ++++ Please run the following command to %s:\n", want)
		fmt.Printf("# =============================================\n")
		defer fmt.Printf("# =============================================\n")
		for _, line := range lines {
			fmt.Printf("%s\n", line)
		}
	}
}
