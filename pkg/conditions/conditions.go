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

package conditions

import (
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionsManager struct {
	mut   sync.RWMutex
	cache map[string][]metav1.Condition
}

func NewConditionsManager() *ConditionsManager {
	return &ConditionsManager{
		cache: map[string][]metav1.Condition{},
	}
}

func (c *ConditionsManager) Set(name string, newCondition metav1.Condition) {
	c.mut.Lock()
	defer c.mut.Unlock()
	cond := c.cache[name]
	newCond := []metav1.Condition{}
	for _, c := range cond {
		newCond = append(newCond, c)
	}
	meta.SetStatusCondition(&newCond, newCondition)
	c.cache[name] = newCond
}

func (c *ConditionsManager) Get(name string) []metav1.Condition {
	c.mut.RLock()
	defer c.mut.RUnlock()
	return c.cache[name]
}

func (c *ConditionsManager) Find(name string, conditionType string) *metav1.Condition {
	c.mut.RLock()
	defer c.mut.RUnlock()
	cond := c.cache[name]
	if len(cond) == 0 {
		return nil
	}
	return meta.FindStatusCondition(cond, conditionType)
}

func (c *ConditionsManager) IsTrue(name string, conditionType string) bool {
	c.mut.RLock()
	defer c.mut.RUnlock()
	cond := c.cache[name]
	if len(cond) == 0 {
		return false
	}
	return meta.IsStatusConditionTrue(cond, conditionType)
}

func (c *ConditionsManager) Delete(name string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.cache, name)
}
