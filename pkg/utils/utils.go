package utils

import (
	"fmt"
	"reflect"

	"github.com/ferry-proxy/utils/objref"
)

type KMetadata = objref.KMetadata

// CalculatePatchResources calculates the resources that need to be updated and deleted in the
// given older and newer resources.
func CalculatePatchResources[T KMetadata](older, newer []T) (updated, deleted []T) {
	if len(older) == 0 {
		return newer, nil
	}
	if len(newer) == 0 {
		return nil, older
	}

	exist := map[string]T{}

	nameFunc := func(m T) string {
		return fmt.Sprintf("%s/%s/%s", reflect.TypeOf(m).Name(), m.GetNamespace(), m.GetName())
	}
	for _, r := range older {
		name := nameFunc(r)
		exist[name] = r
	}
	for _, r := range newer {
		name := nameFunc(r)
		delete(exist, name)
	}
	for _, r := range exist {
		deleted = append(deleted, r)
	}
	return newer, deleted
}
