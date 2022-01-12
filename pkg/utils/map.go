package utils

func MergeMap(m1, m2 map[string]string) map[string]string {
	mergedMap := map[string]string{}

	for k, v := range m1 {
		mergedMap[k] = v
	}
	for k, v := range m2 {
		mergedMap[k] = v
	}
	return mergedMap
}
