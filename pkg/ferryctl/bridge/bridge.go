package bridge

import (
	"context"
	"log"

	_ "github.com/wzshiming/bridge/protocols/command"
	_ "github.com/wzshiming/bridge/protocols/connect"
	_ "github.com/wzshiming/bridge/protocols/netcat"
	_ "github.com/wzshiming/bridge/protocols/socks4"
	_ "github.com/wzshiming/bridge/protocols/socks5"
	_ "github.com/wzshiming/bridge/protocols/ssh"
	_ "github.com/wzshiming/bridge/protocols/tls"

	_ "github.com/wzshiming/anyproxy/proxies/httpproxy"
	_ "github.com/wzshiming/anyproxy/proxies/shadowsocks"
	_ "github.com/wzshiming/anyproxy/proxies/socks4"
	_ "github.com/wzshiming/anyproxy/proxies/socks5"
	_ "github.com/wzshiming/anyproxy/proxies/sshproxy"

	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/go-logr/logr/funcr"
	"github.com/wzshiming/bridge/chain"
)

var Std = funcr.NewJSON(func(obj string) {
	log.Println(obj)
}, funcr.Options{})

type Bridge struct {
}

func NewBridge() *Bridge {
	return &Bridge{}
}

// Perform the following steps to forward:
// local -> kubectl -> apiserver -> kubelet -> container runtime -> nc -> sshd -> target host
// This is a very long data stream, and the estimated performance will be very poor,
// but this is only used for local development tests.

func (c *Bridge) ForwardDial(ctx context.Context, address string, target string) error {
	return chain.Bridge(ctx, Std,
		[]string{
			address,
		},
		[]string{
			target,
			"ssh://127.0.0.1:31088",
			"cmd: kubectl --kubeconfig " + vars.KubeconfigPath + " exec service/ferry-tunnel -i -n ferry-tunnel-system -- nc %h %p",
		},
		false,
	)
}

func (c *Bridge) ForwardListen(ctx context.Context, address string, target string) error {
	return chain.Bridge(ctx, Std,
		[]string{
			address,
			"ssh://127.0.0.1:31088",
			"cmd: kubectl --kubeconfig " + vars.KubeconfigPath + " exec service/ferry-tunnel -i -n ferry-tunnel-system -- nc %h %p",
		},
		[]string{
			target,
		},
		false,
	)
}

func (c *Bridge) ForwardProxy(ctx context.Context, address string) error {
	return chain.Bridge(ctx, Std,
		[]string{
			address,
		},
		[]string{
			"-",
			"ssh://127.0.0.1:31088",
			"cmd: kubectl --kubeconfig " + vars.KubeconfigPath + " exec service/ferry-tunnel -i -n ferry-tunnel-system -- nc %h %p",
		},
		false,
	)
}
