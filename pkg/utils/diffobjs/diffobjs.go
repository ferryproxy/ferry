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

package diffobjs

import (
	"fmt"
	"reflect"
	"sort"

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

// Unique remove duplicate data
func Unique[T objref.KMetadata](older []T) (newer []T) {
	if len(older) <= 1 {
		return older
	}

	exist := map[string]T{}

	for _, r := range older {
		name := uniqName(r)
		exist[name] = r
	}

	newer = make([]T, 0, len(exist))
	for _, r := range exist {
		newer = append(newer, r)
	}

	sort.Slice(newer, func(i, j int) bool {
		return uniqName(newer[i]) < uniqName(newer[j])
	})
	return newer
}
