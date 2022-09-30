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
	"strings"
	"sync"
	"time"

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
	conds := c.cache[name]
	newConds := []metav1.Condition{}
	for _, c := range conds {
		newConds = append(newConds, c)
	}
	meta.SetStatusCondition(&newConds, newCondition)
	c.cache[name] = newConds
}

func (c *ConditionsManager) SetWithDuration(name string, newCondition metav1.Condition, dur time.Duration) bool {
	c.mut.Lock()
	defer c.mut.Unlock()
	conds := c.cache[name]
	cond := meta.FindStatusCondition(conds, newCondition.Type)
	if cond != nil && cond.Status == newCondition.Status && time.Since(cond.LastTransitionTime.Time) < dur {
		return false
	}

	newCond := []metav1.Condition{}
	for _, c := range conds {
		newCond = append(newCond, c)
	}
	meta.SetStatusCondition(&newCond, newCondition)
	c.cache[name] = newCond
	return true
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

func (c *ConditionsManager) Ready(name string, conditionTypes ...string) (bool, string) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	conds := c.cache[name]
	if len(conds) == 0 {
		return false, "<unknown>"
	}

	notReadyReasons := []string{}

	for _, conditionType := range conditionTypes {
		cond := meta.FindStatusCondition(conds, conditionType)
		if cond == nil {
			notReadyReasons = append(notReadyReasons, conditionType+"NotSet")
			continue
		}
		if cond.Status == metav1.ConditionTrue {
			continue
		}
		if cond.Status == metav1.ConditionFalse {
			notReadyReasons = append(notReadyReasons, cond.Reason)
			continue
		}

		notReadyReasons = append(notReadyReasons, cond.Reason+string(cond.Status))
	}

	if len(notReadyReasons) == 0 {
		return true, ""
	}
	return false, strings.Join(notReadyReasons, ",")
}
