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
	"github.com/ferryproxy/ferry/pkg/services"
	"github.com/go-logr/logr"
)

type portPeer struct {
	Cluster   string
	Namespace string
	Name      string
	Port      int32
}

type tunnelPorts struct {
	logger     logr.Logger
	portToPeer map[int32]portPeer
	peerToPort map[portPeer]int32
	portOffset int32
}

type tunnelPortsConfig struct {
	Logger logr.Logger
}

func newTunnelPorts(conf tunnelPortsConfig) *tunnelPorts {
	return &tunnelPorts{
		logger:     conf.Logger,
		portOffset: 40000,
		portToPeer: map[int32]portPeer{},
		peerToPort: map[portPeer]int32{},
	}
}

func (d *tunnelPorts) GetPort(cluster, namespace, name string, port int32) int32 {
	pp := portPeer{
		Cluster:   cluster,
		Namespace: namespace,
		Name:      name,
		Port:      port,
	}

	p := d.peerToPort[pp]
	if p != 0 {
		return p
	}

	for {
		_, ok := d.portToPeer[d.portOffset]
		if !ok {
			break
		}
		d.portOffset++
	}

	p = d.portOffset
	d.portOffset++

	d.portToPeer[p] = pp
	d.peerToPort[pp] = p
	return p
}

func (d *tunnelPorts) LoadPortPeer(cluster, namespace, name string, ports []services.MappingPort) {
	for _, port := range ports {
		peer := portPeer{
			Cluster:   cluster,
			Namespace: namespace,
			Name:      name,
			Port:      port.TargetPort,
		}

		if v, ok := d.portToPeer[port.Port]; ok {
			if v != peer {
				d.logger.Info("duplicate port", "port", port.Port, "peer", peer, "duplicate", v)
				continue
			}
		} else {
			d.portToPeer[port.Port] = peer
		}

		if v, ok := d.peerToPort[peer]; ok {
			if v != port.Port {
				d.logger.Info("duplicate peer", "port", port.Port, "peer", peer, "duplicate", v)
				continue
			}
		} else {
			d.peerToPort[peer] = port.Port
		}
	}
}
