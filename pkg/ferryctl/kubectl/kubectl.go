package kubectl

import (
	"bytes"
	"context"
	"encoding/base64"
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

	"github.com/ferry-proxy/ferry/pkg/consts"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
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

func (c *Kubectl) getSecretName(ctx context.Context) (string, error) {
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "sa", "-n", consts.FerryTunnelNamespace, "ferry-control", "-o", "jsonpath={$.secrets[0].name}")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Kubectl) getToken(ctx context.Context) (string, error) {
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
	out, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "cm", "-n", "kube-public", "cluster-info", "-o", "jsonpath={$.data.kubeconfig}")
	if err != nil {
		return "", err
	}
	take := struct {
		Clusters []struct {
			Cluster struct {
				Server string `yaml:"server"`
			} `yaml:"cluster"`
		} `yaml:"clusters"`
	}{}
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
	address, err := c.GetApiserverAddress(ctx)
	if err != nil {
		return "", err
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		log.Printf("Failed to parse host: %v", err)
		return "", err
	}
	return host + ":31087", nil
}

func (c *Kubectl) GetUnusedPort(ctx context.Context) (string, error) {
	data, err := commandRun(ctx, "kubectl", "--kubeconfig="+vars.KubeconfigPath, "get", "-A", "svc", "-l", "traffic.ferry.zsm.io/exported-from-ports", "-o", "jsonpath={$.items[*].spec.ports[*].targetPort}")
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
