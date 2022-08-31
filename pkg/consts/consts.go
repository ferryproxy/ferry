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

package consts

const (
	FerryName            = "ferry"
	FerryNamespace       = FerryName + "-system"
	FerryTunnelName      = FerryName + "-tunnel"
	FerryTunnelNamespace = FerryTunnelName + "-system"

	LabelPrefix                        = "traffic.ferryproxy.io/"
	LabelFerryExportedFromKey          = LabelPrefix + "exported-from"
	LabelFerryExportedFromNamespaceKey = LabelPrefix + "exported-from-namespace"
	LabelFerryExportedFromNameKey      = LabelPrefix + "exported-from-name"
	LabelFerryExportedFromPortsKey     = LabelPrefix + "exported-from-ports"
	LabelFerryImportedToKey            = LabelPrefix + "imported-to"
	LabelFerryManagedByKey             = LabelPrefix + "managed-by"
	LabelFerryManagedByValue           = "ferry-controller"

	LabelFerryTunnelKey   = "tunnel.ferryproxy.io/service"
	LabelFerryTunnelValue = "inject"

	LabelGeneratedKey   = "generated.ferryproxy.io"
	LabelGeneratedValue = "ferry-controller"

	LabelMCSMarkHubKey   = "mcs.traffic.ferryproxy.io/service"
	LabelMCSMarkHubValue = "enabled"
)
