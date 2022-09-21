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
	Kubeconfig string
}

func NewKubectlInCluster() *Kubectl {
	return &Kubectl{
		BinPath: "kubectl",
	}
}

func NewKubectl() *Kubectl {
	return &Kubectl{
		BinPath:    "kubectl",
		Kubeconfig: vars.KubeconfigPath,
	}
}

func (c *Kubectl) ApplyWithReader(ctx context.Context, r io.Reader) error {
	tmp := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "apply", "-f", "-")
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
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "version", "-o", "json")
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

func (c *Kubectl) GetToken(ctx context.Context) (string, error) {
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
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "create", "token", "-n", consts.FerryTunnelNamespace, "ferry-control")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Kubectl) getSecretName(ctx context.Context) (string, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "get", "sa", "-n", consts.FerryTunnelNamespace, "ferry-control", "-o", "jsonpath={$.secrets[0].name}")
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
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "get", "secret", "-n", consts.FerryTunnelNamespace, secretName, "-o", "jsonpath={$.data.token}")
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
	token, err := c.GetToken(ctx)
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
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "get", "secret", "-n", consts.FerryTunnelNamespace, consts.FerryTunnelName, "-o", "jsonpath={$.data.identity}")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Kubectl) GetSecretAuthorized(ctx context.Context) (string, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "get", "secret", "-n", consts.FerryTunnelNamespace, consts.FerryTunnelName, "-o", "jsonpath={$.data.authorized_keys}")
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

	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "get", "cm", "-n", "kube-public", "cluster-info", "-o", "jsonpath={$.data.kubeconfig}")
	if err != nil {
		data, err := os.ReadFile(c.Kubeconfig)
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
	take := corev1.Service{}

	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "get", "svc", "-n", consts.FerryTunnelNamespace, "gateway-ferry-tunnel", "-o", "yaml")
	if err != nil {
		return "", err
	}
	err = yaml.Unmarshal(out, &take)
	if err != nil {
		return "", err
	}

	address := ""
	port := ""

	if take.Spec.Type == corev1.ServiceTypeLoadBalancer {
		for len(take.Status.LoadBalancer.Ingress) == 0 {
			log.Printf("svc %s.%s ingress is empty, waiting to be created", "gateway-ferry-tunnel", consts.FerryTunnelNamespace)
			time.Sleep(1 * time.Second)
			out, err = commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "get", "svc", "-n", consts.FerryTunnelNamespace, "gateway-ferry-tunnel", "-o", "yaml")
			if err != nil {
				return "", err
			}
			err = yaml.Unmarshal(out, &take)
			if err != nil {
				return "", err
			}
		}

		ingress := take.Status.LoadBalancer.Ingress
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
	_, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "wait", "-n", "ferry-tunnel-system", "deploy/ferry-tunnel", "--for=condition=Available")
	if err != nil {
		return "", err
	}
	data, err := commandRun(ctx, "kubectl", "--kubeconfig="+c.Kubeconfig, "exec", "-n", "ferry-tunnel-system", "deploy/ferry-tunnel", "--", "wget", "-O-", "http://127.0.0.1:8080/ports/unused")
	if err != nil {
		return "", err
	}
	return string(data), nil
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
