package flag

import "strings"

func ParseFlags(setFlags []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, item := range setFlags {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		val := parts[1]

		insertToMap(result, key, val)
	}

	return result
}

func insertToMap(m map[string]interface{}, key string, value interface{}) {
	keys := strings.Split(key, ".")
	last := len(keys) - 1

	current := m
	for i, k := range keys {
		if i == last {
			current[k] = value
			return
		}

		if _, ok := current[k]; !ok {
			current[k] = make(map[string]interface{})
		} else {
			if _, ok := current[k].(map[string]interface{}); !ok {
				current[k] = make(map[string]interface{})
			}
		}

		current = current[k].(map[string]interface{})
	}
}

func ParsePrefix(m map[string]interface{}, prefix string) map[string]interface{} {
	if val, ok := m[prefix]; ok {
		if subMap, ok2 := val.(map[string]interface{}); ok2 {
			return subMap
		}
	}
	return make(map[string]interface{})
}
