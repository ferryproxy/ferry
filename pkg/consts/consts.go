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

var (
	Version     = "unknown"
	ImagePrefix = "ghcr.io/ferryproxy/ferry"
)

const (
	ControlPlaneName = "control-plane"

	FerryName            = "ferry"
	FerryNamespace       = FerryName + "-system"
	FerryTunnelName      = FerryName + "-tunnel"
	FerryTunnelNamespace = FerryTunnelName + "-system"

	LabelPrefix               = "traffic.ferryproxy.io/"
	LabelFerryExportedFromKey = LabelPrefix + "exported-from"
	LabelFerryImportedToKey   = LabelPrefix + "imported-to"
	LabelFerryManagedByValue  = "ferry"

	LabelGeneratedKey         = "generated.ferryproxy.io"
	LabelGeneratedValue       = "ferry-controller"
	LabelGeneratedTunnelValue = "ferry-tunnel"

	LabelMCSMarkHubKey   = "mcs.traffic.ferryproxy.io/service"
	LabelMCSMarkHubValue = "enabled"

	TunnelRulesKey      = "tunnel"
	TunnelRulesAllowKey = "allows"
	TunnelUserKey       = "user"

	TunnelConfigKey             = "tunnel.ferryproxy.io/config"
	TunnelConfigRulesValue      = "rules"
	TunnelConfigDiscoverValue   = "service"
	TunnelConfigAuthorizedValue = "authorized"
	TunnelConfigAllowValue      = "allows"

	TunnelRouteKey = "tunnel.ferryproxy.io/route"

	TunnelRulesConfigPath   = "/var/ferry/bridge.conf"
	TunnelSshDir            = "/var/ferry/ssh/"
	TunnelSshHomeDir        = "/var/ferry/home/"
	TunnelPermissionsName   = "permissions.json"
	TunnelAuthorizedKeyName = "authorized_keys"
	TunnelIdentityKeyName   = "identity"
)
