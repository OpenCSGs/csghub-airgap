package utils

import (
	"reflect"
)

// MergeMap recursively merges src into dst.
// If both dst and src have a key with map[string]interface{} values, they are merged recursively.
// Other types are overwritten.
func MergeMap(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}
	if src == nil {
		return dst
	}

	for k, v := range src {
		if existing, ok := dst[k]; ok {
			existingMap, ok1 := existing.(map[string]interface{})
			vMap, ok2 := v.(map[string]interface{})
			if ok1 && ok2 {
				dst[k] = MergeMap(existingMap, vMap)
			} else {
				dst[k] = v
			}
		} else {
			dst[k] = v
		}
	}

	return dst
}

// IsZeroValue helper: check if a reflection.Value is zero
func IsZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// GetNestedString retrieves a nested string value from a map[string]interface{}
// using a sequence of keys. It safely traverses the map layer by layer.
//
// Parameters:
// - m: the root map to search
// - keys: a variadic list of keys representing the nested path
//
// Returns:
// - string: the value found at the specified path if it exists and is a string
// - bool: true if the value exists and is a string, false otherwise
func GetNestedString(m map[string]interface{}, keys ...string) (string, bool) {
	var current interface{} = m
	for _, k := range keys {
		nextMap, ok := current.(map[string]interface{})
		if !ok {
			return "", false
		}
		next, ok := nextMap[k]
		if !ok {
			return "", false
		}
		current = next
	}
	val, ok := current.(string)
	return val, ok
}

// SetNestedString sets a string value at a nested path in a map[string]interface{}.
// It creates intermediate maps if they do not exist.
func SetNestedString(m map[string]interface{}, value string, keys ...string) {
	if len(keys) == 0 {
		return
	}

	current := m
	for _, k := range keys[:len(keys)-1] {
		if next, ok := current[k]; ok {
			if nextMap, ok2 := next.(map[string]interface{}); ok2 {
				current = nextMap
			} else {
				// overwrite non-map with a new map
				newMap := make(map[string]interface{})
				current[k] = newMap
				current = newMap
			}
		} else {
			newMap := make(map[string]interface{})
			current[k] = newMap
			current = newMap
		}
	}

	// set the final key to the value
	current[keys[len(keys)-1]] = value
}
