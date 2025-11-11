package parser

import (
	"testing"
)

func TestGetStringAttribute(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]any
		key      string
		expected string
		ok       bool
	}{
		{
			name:     "string value",
			attrs:    map[string]any{"name": "test"},
			key:      "name",
			expected: "test",
			ok:       true,
		},
		{
			name:     "float64 value",
			attrs:    map[string]any{"count": 42.0},
			key:      "count",
			expected: "42",
			ok:       true,
		},
		{
			name:     "int value",
			attrs:    map[string]any{"count": 42},
			key:      "count",
			expected: "42",
			ok:       true,
		},
		{
			name:     "bool value",
			attrs:    map[string]any{"enabled": true},
			key:      "enabled",
			expected: "true",
			ok:       true,
		},
		{
			name:     "missing key",
			attrs:    map[string]any{"name": "test"},
			key:      "other",
			expected: "",
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetStringAttribute(tt.attrs, tt.key)
			if ok != tt.ok {
				t.Errorf("GetStringAttribute() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.expected {
				t.Errorf("GetStringAttribute() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetFloat64Attribute(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]any
		key      string
		expected float64
		ok       bool
	}{
		{
			name:     "float64 value",
			attrs:    map[string]any{"price": 42.5},
			key:      "price",
			expected: 42.5,
			ok:       true,
		},
		{
			name:     "int value",
			attrs:    map[string]any{"count": 42},
			key:      "count",
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "string value",
			attrs:    map[string]any{"price": "42.5"},
			key:      "price",
			expected: 42.5,
			ok:       true,
		},
		{
			name:     "missing key",
			attrs:    map[string]any{"price": 42.5},
			key:      "other",
			expected: 0,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetFloat64Attribute(tt.attrs, tt.key)
			if ok != tt.ok {
				t.Errorf("GetFloat64Attribute() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.expected {
				t.Errorf("GetFloat64Attribute() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetIntAttribute(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]any
		key      string
		expected int
		ok       bool
	}{
		{
			name:     "int value",
			attrs:    map[string]any{"count": 42},
			key:      "count",
			expected: 42,
			ok:       true,
		},
		{
			name:     "float64 value",
			attrs:    map[string]any{"count": 42.0},
			key:      "count",
			expected: 42,
			ok:       true,
		},
		{
			name:     "string value",
			attrs:    map[string]any{"count": "42"},
			key:      "count",
			expected: 42,
			ok:       true,
		},
		{
			name:     "missing key",
			attrs:    map[string]any{"count": 42},
			key:      "other",
			expected: 0,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetIntAttribute(tt.attrs, tt.key)
			if ok != tt.ok {
				t.Errorf("GetIntAttribute() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.expected {
				t.Errorf("GetIntAttribute() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetBoolAttribute(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]any
		key      string
		expected bool
		ok       bool
	}{
		{
			name:     "bool true",
			attrs:    map[string]any{"enabled": true},
			key:      "enabled",
			expected: true,
			ok:       true,
		},
		{
			name:     "bool false",
			attrs:    map[string]any{"enabled": false},
			key:      "enabled",
			expected: false,
			ok:       true,
		},
		{
			name:     "string true",
			attrs:    map[string]any{"enabled": "true"},
			key:      "enabled",
			expected: true,
			ok:       true,
		},
		{
			name:     "string yes",
			attrs:    map[string]any{"enabled": "yes"},
			key:      "enabled",
			expected: true,
			ok:       true,
		},
		{
			name:     "float64 non-zero",
			attrs:    map[string]any{"enabled": 1.0},
			key:      "enabled",
			expected: true,
			ok:       true,
		},
		{
			name:     "int zero",
			attrs:    map[string]any{"enabled": 0},
			key:      "enabled",
			expected: false,
			ok:       true,
		},
		{
			name:     "missing key",
			attrs:    map[string]any{"enabled": true},
			key:      "other",
			expected: false,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetBoolAttribute(tt.attrs, tt.key)
			if ok != tt.ok {
				t.Errorf("GetBoolAttribute() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.expected {
				t.Errorf("GetBoolAttribute() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetStringSliceAttribute(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]any
		key      string
		expected []string
		ok       bool
	}{
		{
			name:     "string slice",
			attrs:    map[string]any{"tags": []string{"a", "b", "c"}},
			key:      "tags",
			expected: []string{"a", "b", "c"},
			ok:       true,
		},
		{
			name:     "interface slice",
			attrs:    map[string]any{"tags": []any{"a", "b", "c"}},
			key:      "tags",
			expected: []string{"a", "b", "c"},
			ok:       true,
		},
		{
			name:     "single string",
			attrs:    map[string]any{"tag": "single"},
			key:      "tag",
			expected: []string{"single"},
			ok:       true,
		},
		{
			name:     "mixed types",
			attrs:    map[string]any{"values": []any{"str", 42, 42.5}},
			key:      "values",
			expected: []string{"str", "42", "42"},
			ok:       true,
		},
		{
			name:     "missing key",
			attrs:    map[string]any{"tags": []string{"a"}},
			key:      "other",
			expected: nil,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetStringSliceAttribute(tt.attrs, tt.key)
			if ok != tt.ok {
				t.Errorf("GetStringSliceAttribute() ok = %v, want %v", ok, tt.ok)
			}
			if len(got) != len(tt.expected) {
				t.Errorf("GetStringSliceAttribute() length = %v, want %v", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("GetStringSliceAttribute()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGetMapAttribute(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]any
		key      string
		expected map[string]any
		ok       bool
	}{
		{
			name:     "valid map",
			attrs:    map[string]any{"config": map[string]any{"key": "value"}},
			key:      "config",
			expected: map[string]any{"key": "value"},
			ok:       true,
		},
		{
			name:     "missing key",
			attrs:    map[string]any{"config": map[string]any{"key": "value"}},
			key:      "other",
			expected: nil,
			ok:       false,
		},
		{
			name:     "non-map value",
			attrs:    map[string]any{"config": "string"},
			key:      "config",
			expected: nil,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetMapAttribute(tt.attrs, tt.key)
			if ok != tt.ok {
				t.Errorf("GetMapAttribute() ok = %v, want %v", ok, tt.ok)
			}
			if ok && len(got) != len(tt.expected) {
				t.Errorf("GetMapAttribute() length = %v, want %v", len(got), len(tt.expected))
			}
		})
	}
}

func TestGetNestedAttribute(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[string]any
		path     string
		expected any
		ok       bool
	}{
		{
			name:     "existing path",
			attrs:    map[string]any{"vpc.id": "vpc-123"},
			path:     "vpc.id",
			expected: "vpc-123",
			ok:       true,
		},
		{
			name:     "missing path",
			attrs:    map[string]any{"vpc.id": "vpc-123"},
			path:     "other.id",
			expected: nil,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetNestedAttribute(tt.attrs, tt.path)
			if ok != tt.ok {
				t.Errorf("GetNestedAttribute() ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.expected {
				t.Errorf("GetNestedAttribute() = %v, want %v", got, tt.expected)
			}
		})
	}
}
