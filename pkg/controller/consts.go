package controller

const (
	LabelPrefix              = "ferry.zsm.io/"
	LabelFerryExportedFrom   = LabelPrefix + "exported-from"
	LabelFerryImportedTo     = LabelPrefix + "imported-to"
	LabelFerryManagedByKey   = LabelPrefix + "managed-by"
	LabelFerryManagedByValue = "ferry-controller"
)
