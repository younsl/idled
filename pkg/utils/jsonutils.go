package utils

import (
	"encoding/json"
	"fmt"
)

// GetNestedString extracts a string from a nested map
func GetNestedString(data map[string]interface{}, keys ...string) (string, error) {
	var current interface{} = data

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key, should be a string
			if str, ok := current.(map[string]interface{})[key].(string); ok {
				return str, nil
			}
			return "", fmt.Errorf("key %s is not a string", key)
		}

		// Not the last key, should be a map
		if nestedMap, ok := current.(map[string]interface{})[key].(map[string]interface{}); ok {
			current = nestedMap
		} else {
			return "", fmt.Errorf("key %s is not a map", key)
		}
	}

	return "", fmt.Errorf("invalid keys")
}

// GetNestedFloat extracts a float64 from a nested map
func GetNestedFloat(data map[string]interface{}, keys ...string) (float64, error) {
	var current interface{} = data

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key, should be a number
			if num, ok := current.(map[string]interface{})[key].(float64); ok {
				return num, nil
			}

			// Maybe it's a string that can be parsed as a number
			if str, ok := current.(map[string]interface{})[key].(string); ok {
				if num, err := parseNumber(str); err == nil {
					return num, nil
				}
			}

			return 0, fmt.Errorf("key %s is not a number", key)
		}

		// Not the last key, should be a map
		if nestedMap, ok := current.(map[string]interface{})[key].(map[string]interface{}); ok {
			current = nestedMap
		} else {
			return 0, fmt.Errorf("key %s is not a map", key)
		}
	}

	return 0, fmt.Errorf("invalid keys")
}

// GetFirstMapValue returns the first value in a map
func GetFirstMapValue(m map[string]interface{}) (interface{}, error) {
	for _, v := range m {
		return v, nil
	}
	return nil, fmt.Errorf("map is empty")
}

// ParseJSON parses a JSON string into a map
func ParseJSON(jsonStr string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	return result, nil
}

// FormatJSON formats a map as JSON with indentation
func FormatJSON(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error formatting JSON: %w", err)
	}
	return string(bytes), nil
}

// Try to parse a string as a number
func parseNumber(str string) (float64, error) {
	var f float64
	err := json.Unmarshal([]byte(str), &f)
	return f, err
}
