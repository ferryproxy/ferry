package diffobjs

import (
	"fmt"
	"reflect"

	"github.com/ferryproxy/ferry/pkg/utils/objref"
)

func uniqName[T objref.KMetadata](m T) string {
	typ := reflect.TypeOf(m).Name()
	name := m.GetName()
	ns := m.GetNamespace()
	return fmt.Sprintf("%s/%s/%s", typ, ns, name)
}

// ShouldDeleted calculates the resources that need to be deleted in the
// given older and newer resources.
func ShouldDeleted[T objref.KMetadata](older, newer []T) (deleted []T) {
	if len(older) == 0 {
		return nil
	}
	if len(newer) == 0 {
		return older
	}

	exist := map[string]T{}

	for _, r := range older {
		name := uniqName(r)
		exist[name] = r
	}
	for _, r := range newer {
		name := uniqName(r)
		delete(exist, name)
	}
	for _, r := range exist {
		deleted = append(deleted, r)
	}
	return deleted
}
