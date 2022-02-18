package consts

const (
	LabelPrefix               = "ferry.zsm.io/"
	LabelFerryExportedFromKey = LabelPrefix + "exported-from"
	LabelFerryImportedToKey   = LabelPrefix + "imported-to"
	LabelFerryManagedByKey    = LabelPrefix + "managed-by"
	LabelFerryManagedByValue  = "ferry-controller"

	LabelFerryTunnelKey   = "ferry-tunnel"
	LabelFerryTunnelValue = "ferry"
)
