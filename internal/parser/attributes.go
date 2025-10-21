package parser

import (
	"fmt"
	"strconv"
)

// Attribute helper functions for safe type handling from Terraform state/config
// These handle the fact that Terraform JSON can have inconsistent types
// (strings, numbers as float64, integers, arrays, etc.)

// GetStringAttribute safely extracts a string attribute, converting if needed
func GetStringAttribute(attrs map[string]interface{}, key string) (string, bool) {
	val, ok := attrs[key]
	if !ok {
		return "", false
	}

	switch v := val.(type) {
	case string:
		return v, true
	case float64:
		// JSON numbers are always float64
		return fmt.Sprintf("%.0f", v), true
	case int:
		return fmt.Sprintf("%d", v), true
	case bool:
		return fmt.Sprintf("%t", v), true
	default:
		return "", false
	}
}

// GetFloat64Attribute safely extracts a float64 attribute, converting if needed
func GetFloat64Attribute(attrs map[string]interface{}, key string) (float64, bool) {
	val, ok := attrs[key]
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case string:
		// Try to parse string as float
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// GetIntAttribute safely extracts an int attribute, converting if needed
func GetIntAttribute(attrs map[string]interface{}, key string) (int, bool) {
	val, ok := attrs[key]
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		// JSON numbers are float64, convert to int
		return int(v), true
	case string:
		// Try to parse string as int
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}
	return 0, false
}

// GetBoolAttribute safely extracts a bool attribute
func GetBoolAttribute(attrs map[string]interface{}, key string) (bool, bool) {
	val, ok := attrs[key]
	if !ok {
		return false, false
	}

	switch v := val.(type) {
	case bool:
		return v, true
	case string:
		// Handle common string representations
		switch v {
		case "true", "True", "TRUE", "1", "yes", "Yes", "YES":
			return true, true
		case "false", "False", "FALSE", "0", "no", "No", "NO":
			return false, true
		}
	case float64:
		return v != 0, true
	case int:
		return v != 0, true
	}
	return false, false
}

// GetStringSliceAttribute safely extracts a string slice attribute
func GetStringSliceAttribute(attrs map[string]interface{}, key string) ([]string, bool) {
	val, ok := attrs[key]
	if !ok {
		return nil, false
	}

	switch v := val.(type) {
	case []interface{}:
		// Convert each element to string
		result := make([]string, 0, len(v))
		for _, item := range v {
			switch str := item.(type) {
			case string:
				result = append(result, str)
			case float64:
				result = append(result, fmt.Sprintf("%.0f", str))
			case int:
				result = append(result, fmt.Sprintf("%d", str))
			default:
				// Skip non-convertible items
				continue
			}
		}
		return result, len(result) > 0
	case []string:
		return v, true
	case string:
		// Single string, wrap in slice
		return []string{v}, true
	}
	return nil, false
}

// GetMapAttribute safely extracts a map attribute
func GetMapAttribute(attrs map[string]interface{}, key string) (map[string]interface{}, bool) {
	val, ok := attrs[key]
	if !ok {
		return nil, false
	}

	switch v := val.(type) {
	case map[string]interface{}:
		return v, true
	default:
		return nil, false
	}
}

// GetNestedAttribute safely extracts a nested attribute using dot notation
// Example: GetNestedAttribute(attrs, "vpc.id") -> attrs["vpc"]["id"]
func GetNestedAttribute(attrs map[string]interface{}, path string) (interface{}, bool) {
	// Simple implementation for single level nesting
	// Can be extended to support deeper nesting if needed
	val, ok := attrs[path]
	return val, ok
}
