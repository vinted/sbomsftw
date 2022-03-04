package utils

// Keys returns all keys from the map in unpredictable order
func Keys(target map[string]bool) []string {
	keys := make([]string, len(target))
	i := 0
	for k := range target {
		keys[i] = k
		i++
	}
	return keys
}
