package controller

import (
	"strconv"
	"strings"

	"github.com/ferry-proxy/ferry/pkg/consts"
	corev1 "k8s.io/api/core/v1"
)

type portPeer struct {
	Cluster   string
	Namespace string
	Name      string
	Port      int32
}

type tunnelPorts struct {
	portToPeer map[int32]portPeer
	peerToPort map[portPeer]int32
	portOffset int32
}

func newTunnelPorts() *tunnelPorts {
	return &tunnelPorts{
		portOffset: 40000,
	}
}

func (d *tunnelPorts) getPort(cluster, namespace, name string, port int32) int32 {
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
			p = d.portOffset
			d.portOffset++
			break
		}
		d.portOffset++
	}

	d.portToPeer[p] = pp
	d.peerToPort[pp] = p
	return p
}

func (d *tunnelPorts) loadPortPeer(list *corev1.ServiceList) error {
	d.portToPeer = make(map[int32]portPeer)
	d.peerToPort = make(map[portPeer]int32)

	for _, item := range list.Items {
		err := d.loadPortPeerForService(&item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *tunnelPorts) loadPortPeerForService(svc *corev1.Service) error {
	if svc.Labels == nil ||
		svc.Labels[consts.LabelFerryExportedFromKey] == "" ||
		svc.Labels[consts.LabelFerryExportedFromNamespaceKey] == "" ||
		svc.Labels[consts.LabelFerryExportedFromNameKey] == "" ||
		svc.Labels[consts.LabelFerryExportedFromPortsKey] == "" {
		return nil
	}
	cluster := svc.Labels[consts.LabelFerryExportedFromKey]
	namespace := svc.Labels[consts.LabelFerryExportedFromNamespaceKey]
	name := svc.Labels[consts.LabelFerryExportedFromNameKey]
	ports := strings.Split(svc.Labels[consts.LabelFerryExportedFromPortsKey], "-")
	for _, portStr := range ports {
		port, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil {
			return err
		}
		var serverPort int32
		for _, svcPort := range svc.Spec.Ports {
			if strings.HasSuffix(svcPort.Name, "-"+portStr) {
				serverPort = svcPort.Port
				break
			}
		}
		peer := portPeer{
			Cluster:   cluster,
			Namespace: namespace,
			Name:      name,
			Port:      serverPort,
		}

		d.portToPeer[int32(port)] = peer
		d.peerToPort[peer] = int32(port)
	}
	return nil
}
