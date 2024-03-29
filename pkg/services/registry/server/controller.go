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

package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"sync"

	trafficv1alpha2 "github.com/ferryproxy/api/apis/traffic/v1alpha2"
	"github.com/ferryproxy/ferry/pkg/client"
	"github.com/ferryproxy/ferry/pkg/consts"
	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/router"
	"github.com/ferryproxy/ferry/pkg/services/registry/models"
	"github.com/ferryproxy/ferry/pkg/utils/encoding"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Controller struct {
	mut           sync.Mutex
	GetBindPort   func(ctx context.Context) (int32, error)
	TunnelAddress string
	Clientset     client.Interface
	Logger        logr.Logger
}

func (c *Controller) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	switch r.Method {
	case http.MethodHead:
		c.Head(rw, r)
	case http.MethodPost:
		c.Create(rw, r)
	default:
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

// Head HEAD /hubs/{hub_name}
func (c *Controller) Head(rw http.ResponseWriter, r *http.Request) {
	c.mut.Lock()
	defer c.mut.Unlock()

	ok, err := c.isExistHub(r.Context(), c.getHubName(r))
	if err != nil {
		c.Logger.Error(err, "get hub")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		rw.WriteHeader(http.StatusCreated)
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

// Create POST /hubs/{hub_name}
func (c *Controller) Create(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1*1024*1024))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	joinHub := models.JoinHub{}
	err = json.Unmarshal(body, &joinHub)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if joinHub.HubName == "" || joinHub.HubName != c.getHubName(r) {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if joinHub.AuthorizedKey == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if joinHub.Token == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	c.mut.Lock()
	defer c.mut.Unlock()

	importHubName := consts.ControlPlaneName
	exportHubName := joinHub.HubName

	bindPort, err := c.GetBindPort(r.Context())
	if err != nil {
		c.Logger.Error(err, "build resource")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	mc := router.ManualConfig{
		Labels: map[string]string{
			consts.TunnelRouteKey: joinHub.HubName,
		},
		RouteName:       exportHubName + "-apiserver",
		ImportHubName:   importHubName,
		ImportName:      exportHubName + "-apiserver",
		ImportNamespace: consts.FerryTunnelNamespace,
		ImportGateway: trafficv1alpha2.HubSpecGateway{
			Reachable: true,
			Address:   c.TunnelAddress,
		},
		ImportAuthorized: joinHub.AuthorizedKey,
		BindPort:         bindPort,
		Port:             443,
		ExportHubName:    exportHubName,
		ExportName:       "kubernetes",
		ExportNamespace:  "default",
		ExportGateway: trafficv1alpha2.HubSpecGateway{
			Reachable: false,
			Address:   "",
		},
		ExportAuthorized: joinHub.AuthorizedKey,
	}
	m := router.NewManual(mc)

	out, err := m.BuildResource()
	if err != nil {
		c.Logger.Error(err, "build resource")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	importHubResource := out[importHubName]
	exportHubResource := out[exportHubName]

	repo, err := encoding.MarshalJSON(exportHubResource...)
	if err != nil {
		c.Logger.Error(err, "Marshal JSON")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	kubeconfig, err := kubectl.BuildKubeconfig(kubectl.BuildKubeconfigConfig{
		Name:             "ferry-control",
		ApiserverAddress: "https://" + exportHubName + "-apiserver.ferry-tunnel-system:443",
		Token:            joinHub.Token,
	})
	if err != nil {
		c.Logger.Error(err, "Build Kubeconfig")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      joinHub.HubName,
			Namespace: consts.FerryNamespace,
		},
		Type: "traffic.ferryproxy.io/kubeconfig-key",
		Data: map[string][]byte{
			"kubeconfig": []byte(kubeconfig),
		},
	}

	importHubResource = append(importHubResource, &secret)

	hub := &trafficv1alpha2.Hub{
		ObjectMeta: metav1.ObjectMeta{
			Name:      joinHub.HubName,
			Namespace: consts.FerryNamespace,
		},
		Spec: trafficv1alpha2.HubSpec{
			Gateway: trafficv1alpha2.HubSpecGateway{
				NavigationWay: []trafficv1alpha2.HubSpecGatewayWay{
					{
						HubName: importHubName,
					},
				},
				ReceptionWay: []trafficv1alpha2.HubSpecGatewayWay{
					{
						HubName: importHubName,
					},
				},
			},
		},
	}

	ok, err := c.isExistHub(r.Context(), joinHub.HubName)
	if err != nil {
		c.Logger.Error(err, "get hub")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		c.Logger.Error(err, "hub existing", "hub", joinHub.HubName)
		http.Error(rw, http.StatusText(http.StatusConflict), http.StatusConflict)
	}

	defer func() {
		if err != nil {
			ctx := context.Background()
			for _, src := range importHubResource {
				client.Delete(ctx, c.Logger, c.Clientset, src)
			}
		}
	}()
	for _, src := range importHubResource {
		err = client.Apply(r.Context(), c.Logger, c.Clientset, src)
		if err != nil {
			c.Logger.Error(err, "Apply")
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	defer func() {
		if err != nil {
			client.Delete(r.Context(), c.Logger, c.Clientset, hub)
		}
	}()

	err = client.Apply(r.Context(), c.Logger, c.Clientset, hub)
	if err != nil {
		c.Logger.Error(err, "Apply")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(repo)
}

func (c *Controller) getHubName(r *http.Request) string {
	return path.Base(r.URL.Path)
}

func (c *Controller) isExistHub(ctx context.Context, hubName string) (bool, error) {
	_, err := c.Clientset.
		Ferry().
		TrafficV1alpha2().
		Hubs(consts.FerryNamespace).
		Get(ctx, hubName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
