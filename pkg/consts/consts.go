package consts

const (
	LabelPrefix                        = "traffic.ferry.zsm.io/"
	LabelFerryExportedFromKey          = LabelPrefix + "exported-from"
	LabelFerryExportedFromNamespaceKey = LabelPrefix + "exported-from-namespace"
	LabelFerryExportedFromNameKey      = LabelPrefix + "exported-from-name"
	LabelFerryExportedFromPortsKey     = LabelPrefix + "exported-from-ports"
	LabelFerryImportedToKey            = LabelPrefix + "imported-to"
	LabelFerryManagedByKey             = LabelPrefix + "managed-by"
	LabelFerryManagedByValue           = "ferry-controller"

	LabelFerryTunnelKey   = "ferry-tunnel"
	LabelFerryTunnelValue = "true"
)
