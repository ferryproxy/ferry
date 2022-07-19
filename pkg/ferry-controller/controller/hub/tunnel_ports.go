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
	"strconv"
	"strings"

	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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

func (d *tunnelPorts) LoadPortPeer(list *corev1.ServiceList) {
	for _, item := range list.Items {
		d.loadPortPeerForService(&item)
	}
}

func (d *tunnelPorts) loadPortPeerForService(svc *corev1.Service) {
	if svc.Labels == nil ||
		svc.Labels[consts.LabelFerryExportedFromKey] == "" ||
		svc.Labels[consts.LabelFerryExportedFromNamespaceKey] == "" ||
		svc.Labels[consts.LabelFerryExportedFromNameKey] == "" ||
		svc.Labels[consts.LabelFerryExportedFromPortsKey] == "" {
		return
	}
	cluster := svc.Labels[consts.LabelFerryExportedFromKey]
	namespace := svc.Labels[consts.LabelFerryExportedFromNamespaceKey]
	name := svc.Labels[consts.LabelFerryExportedFromNameKey]
	ports := strings.Split(svc.Labels[consts.LabelFerryExportedFromPortsKey], "-")
	logger := d.logger.WithValues(
		"cluster", cluster,
		"namespace", namespace,
		"name", name,
	)
	for _, portStr := range ports {
		portRaw, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil {
			logger.Error(err, "Failed to parse port")
			continue
		}

		var serverPort int32
		for _, svcPort := range svc.Spec.Ports {
			if svcPort.TargetPort.String() == portStr {
				serverPort = svcPort.Port
				break
			}
		}

		if serverPort == 0 {
			logger.Info("no match service port")
			continue
		}

		port := int32(portRaw)
		peer := portPeer{
			Cluster:   cluster,
			Namespace: namespace,
			Name:      name,
			Port:      serverPort,
		}

		if v, ok := d.portToPeer[port]; ok {
			if v != peer {
				logger.Info("duplicate port", "port", port, "peer", peer, "duplicate", v)
				continue
			}
		} else {
			d.portToPeer[port] = peer
		}

		if v, ok := d.peerToPort[peer]; ok {
			if v != port {
				logger.Info("duplicate peer", "port", port, "peer", peer, "duplicate", v)
				continue
			}
		} else {
			d.peerToPort[peer] = port
		}
	}
}
