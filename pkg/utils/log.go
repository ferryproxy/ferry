package utils

import (
	"fmt"
	"reflect"
)

// ObjectRef references a kubernetes object
type ObjectRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

func (ref ObjectRef) String() string {
	if ref.Namespace != "" {
		return fmt.Sprintf("%s/%s", ref.Namespace, ref.Name)
	}
	return ref.Name
}

type KMetadata interface {
	GetName() string
	GetNamespace() string
}

func KObj(obj KMetadata) ObjectRef {
	if obj == nil {
		return ObjectRef{}
	}
	if val := reflect.ValueOf(obj); val.Kind() == reflect.Ptr && val.IsNil() {
		return ObjectRef{}
	}

	return ObjectRef{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

// KRef returns ObjectRef from name and namespace
func KRef(namespace, name string) ObjectRef {
	return ObjectRef{
		Name:      name,
		Namespace: namespace,
	}
}

// KObjs returns slice of ObjectRef from an slice of ObjectMeta
func KObjs(arg interface{}) []ObjectRef {
	s := reflect.ValueOf(arg)
	if s.Kind() != reflect.Slice {
		return nil
	}
	objectRefs := make([]ObjectRef, 0, s.Len())
	for i := 0; i < s.Len(); i++ {
		if v, ok := s.Index(i).Interface().(KMetadata); ok {
			objectRefs = append(objectRefs, KObj(v))
		} else {
			return nil
		}
	}
	return objectRefs
}
