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

package kubectl

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferryctl/vars"
	corev1 "k8s.io/api/core/v1"
	apimachineryversion "k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/yaml"
)

//TODO: Refactor this to use native code instead of other binaries(kubectl)

type Kubectl struct {
	BinPath    string
	KubeConfig string
}

func NewKubectl() *Kubectl {
	return &Kubectl{
		BinPath: "kubectl",
	}
}

func (c *Kubectl) ApplyWithReader(ctx context.Context, r io.Reader) error {
	tmp := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "apply", "-f", "-")
	cmd.Stdin = io.TeeReader(r, tmp)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("kubectl apply -f - <<EOF\n%s\nEOF\n%w", tmp.String(), err)
	}
	return nil
}

// Version is a struct for version information
type Version struct {
	ClientVersion    *apimachineryversion.Info `json:"clientVersion,omitempty" yaml:"clientVersion,omitempty"`
	KustomizeVersion string                    `json:"kustomizeVersion,omitempty" yaml:"kustomizeVersion,omitempty"`
	ServerVersion    *apimachineryversion.Info `json:"serverVersion,omitempty" yaml:"serverVersion,omitempty"`
}

func (c *Kubectl) getVersion(ctx context.Context) (*Version, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "version", "-o", "json")
	if err != nil {
		return nil, err
	}
	version := &Version{}
	err = json.Unmarshal(out, &version)
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (c *Kubectl) getToken(ctx context.Context) (string, error) {
	v, err := c.getVersion(ctx)
	if err != nil {
		return "", err
	}
	if v.ServerVersion != nil && v.ServerVersion.Minor != "" {
		minor, _ := strconv.ParseUint(v.ServerVersion.Minor, 10, 64)
		if minor >= 24 {
			return c.getTokenFor124AndAfter(ctx)
		}
	}
	return c.getTokenForBefore124(ctx)
}

func (c *Kubectl) getTokenFor124AndAfter(ctx context.Context) (string, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "create", "token", "-n", consts.FerryTunnelNamespace, "ferry-control")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Kubectl) getSecretName(ctx context.Context) (string, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "sa", "-n", consts.FerryTunnelNamespace, "ferry-control", "-o", "jsonpath={$.secrets[0].name}")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Kubectl) getTokenForBefore124(ctx context.Context) (string, error) {
	secretName, err := c.getSecretName(ctx)
	if err != nil {
		return "", err
	}
	for secretName == "" {
		log.Println("secret name of service account is empty, waiting to be created")
		time.Sleep(1 * time.Second)
		secretName, err = c.getSecretName(ctx)
		if err != nil {
			return "", err
		}
	}
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "secret", "-n", consts.FerryTunnelNamespace, secretName, "-o", "jsonpath={$.data.token}")
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(string(out))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (c *Kubectl) GetKubeconfig(ctx context.Context, address string) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}
	return BuildKubeconfig(BuildKubeconfigConfig{
		Name:             "ferry-control",
		ApiserverAddress: "https://" + address,
		Token:            token,
	})
}

func (c *Kubectl) GetSecretIdentity(ctx context.Context) (string, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "secret", "-n", consts.FerryTunnelNamespace, consts.FerryTunnelName, "-o", "jsonpath={$.data.identity}")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Kubectl) GetSecretAuthorized(ctx context.Context) (string, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "secret", "-n", consts.FerryTunnelNamespace, consts.FerryTunnelName, "-o", "jsonpath={$.data.authorized}")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Kubectl) GetApiserverAddress(ctx context.Context) (string, error) {
	take := struct {
		Clusters []struct {
			Cluster struct {
				Server string `yaml:"server"`
			} `yaml:"cluster"`
		} `yaml:"clusters"`
	}{}

	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "cm", "-n", "kube-public", "cluster-info", "-o", "jsonpath={$.data.kubeconfig}")
	if err != nil {
		data, err := os.ReadFile(vars.KubeconfigPath)
		if err != nil {
			return "", err
		}
		out = data
	}

	err = yaml.Unmarshal(out, &take)
	if err != nil {
		return "", err
	}

	if len(take.Clusters) == 0 || take.Clusters[0].Cluster.Server == "" {
		return "", fmt.Errorf("not found server address %s", string(out))
	}
	server := take.Clusters[0].Cluster.Server
	uri, err := url.Parse(server)
	if err != nil {
		return "", err
	}
	_, _, err = net.SplitHostPort(uri.Host)
	if err != nil {
		if err.Error() == "missing port in address" {
			return uri.Host + ":443", nil
		}
		return "", err
	}
	return uri.Host, nil
}

func (c *Kubectl) GetTunnelAddress(ctx context.Context) (string, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "svc", "-n", consts.FerryTunnelNamespace, "gateway-ferry-tunnel", "-o", "yaml")
	if err != nil {
		return "", err
	}
	take := corev1.Service{}
	err = yaml.Unmarshal(out, &take)
	if err != nil {
		return "", err
	}

	address := ""
	port := ""

	if take.Spec.Type == corev1.ServiceTypeLoadBalancer {
		ingress := take.Status.LoadBalancer.Ingress
		if len(ingress) != 0 {
			if address == "" && ingress[0].IP != "" {
				address = ingress[0].IP
			}
			if address == "" && ingress[0].Hostname != "" {
				address = ingress[0].Hostname
			}
			if port == "" && len(ingress[0].Ports) != 0 {
				port = strconv.FormatInt(int64(ingress[0].Ports[0].Port), 10)
			}
		}
	}

	if port == "" && len(take.Spec.Ports) != 0 {
		port = strconv.FormatInt(int64(take.Spec.Ports[0].Port), 10)
	}

	if port == "" {
		port = "31087"
	}

	if address == "" {
		host, err := c.GetApiserverAddress(ctx)
		if err != nil {
			return "", err
		}
		address, _, err = net.SplitHostPort(host)
		if err != nil {
			log.Printf("Failed to parse host: %v", err)
			return "", err
		}
	}

	return address + ":" + port, nil
}

func (c *Kubectl) GetUnusedPort(ctx context.Context) (string, error) {
	data, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "-A", "svc", "-l", "traffic.ferryproxy.io/exported-from-ports", "-o", "jsonpath={$.items[*].spec.ports[*].targetPort}")
	if err != nil {
		return "", err
	}

	used := map[string]struct{}{}
	for _, i := range strings.Split(string(data), " ") {
		used[i] = struct{}{}
	}
	var port int64 = 20000
	for ; ; port++ {
		p := strconv.FormatInt(port, 10)
		if _, ok := used[p]; !ok {
			return p, nil
		}
	}
}

func (c *Kubectl) Wrap(ctx context.Context, args ...string) error {
	fmt.Fprintf(os.Stderr, "kubectl %s\n", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func commandRun(ctx context.Context, name string, args ...string) ([]byte, error) {
	out := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%s %s :%w", name, strings.Join(args, " "), err)
	}
	return bytes.TrimSpace(out.Bytes()), nil
}
