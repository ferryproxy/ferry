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
)
