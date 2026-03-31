package utils

// NVL An empty string fallback: Returns a fallback if the input is empty, otherwise returns the input.
func NVL(input, fallback string) string {
	if input == "" {
		return fallback
	}
	return input
}

// NVLMap An empty string fallback: Returns a fallback if the input is empty, otherwise returns the input.
func NVLMap(input, fallback map[string]interface{}) map[string]interface{} {
	if input == nil {
		return fallback
	}
	return input
}
