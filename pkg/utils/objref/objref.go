package objref

import (
	"fmt"
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

func KObj[T KMetadata](obj T) ObjectRef {
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
func KObjs[T KMetadata](objs []T) []ObjectRef {
	objectRefs := make([]ObjectRef, 0, len(objs))
	for _, obj := range objs {
		objectRefs = append(objectRefs, KObj(obj))
	}
	return objectRefs
}
