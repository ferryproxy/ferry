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

package hub

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
)

type portPeer struct {
	Cluster   string
	Namespace string
	Name      string
	Port      int32
}

type tunnelPorts struct {
	logger        logr.Logger
	portToPeer    map[int32]portPeer
	peerToPort    map[portPeer]int32
	mut           sync.Mutex
	getUnusedPort func() (int32, error)
}

type tunnelPortsConfig struct {
	Logger        logr.Logger
	GetUnusedPort func() (int32, error)
}

func newTunnelPorts(conf tunnelPortsConfig) *tunnelPorts {
	return &tunnelPorts{
		logger:        conf.Logger,
		portToPeer:    map[int32]portPeer{},
		peerToPort:    map[portPeer]int32{},
		getUnusedPort: conf.GetUnusedPort,
	}
}

func (d *tunnelPorts) GetPortBind(cluster, namespace, name string, port int32) (int32, error) {
	d.mut.Lock()
	defer d.mut.Unlock()
	peer := portPeer{
		Cluster:   cluster,
		Namespace: namespace,
		Name:      name,
		Port:      port,
	}

	p := d.peerToPort[peer]
	if p != 0 {
		return p, nil
	}

	p, err := d.getUnusedPort()
	if err != nil {
		return 0, err
	}

	d.portToPeer[p] = peer
	d.peerToPort[peer] = p
	return p, nil
}

func (d *tunnelPorts) DeletePortBind(cluster, namespace, name string, port int32) (int32, error) {
	d.mut.Lock()
	defer d.mut.Unlock()
	peer := portPeer{
		Cluster:   cluster,
		Namespace: namespace,
		Name:      name,
		Port:      port,
	}

	p := d.peerToPort[peer]
	if p == 0 {
		return 0, fmt.Errorf("not found bind port for %s.%s:%d on %s", namespace, name, port, cluster)
	}
	delete(d.peerToPort, peer)
	delete(d.portToPeer, p)

	return p, nil
}

func (d *tunnelPorts) LoadPortBind(cluster, namespace, name string, port, bindPort int32) error {
	d.mut.Lock()
	defer d.mut.Unlock()
	peer := portPeer{
		Cluster:   cluster,
		Namespace: namespace,
		Name:      name,
		Port:      port,
	}

	if oldPeer, ok := d.portToPeer[bindPort]; ok && oldPeer != peer {
		return fmt.Errorf("duplicate peer, load peers %v and %v both trying to use the %d port", peer, oldPeer, bindPort)
	}
	if oldPort, ok := d.peerToPort[peer]; ok && oldPort != bindPort {
		return fmt.Errorf("duplicate peer port, load peer %v to use %d port, but it already uses %d port", peer, bindPort, oldPort)
	}

	d.portToPeer[bindPort] = peer
	d.peerToPort[peer] = bindPort
	return nil
}
