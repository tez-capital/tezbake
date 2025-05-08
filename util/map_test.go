package util

import (
	"reflect"
	"testing"
)

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name      string
		dst       map[string]any
		src       map[string]any
		overwrite bool
		expected  map[string]any
	}{
		{
			name:      "Basic merge with overwrite",
			dst:       map[string]any{"a": 1, "b": 2},
			src:       map[string]any{"b": 3, "c": 4},
			overwrite: true,
			expected:  map[string]any{"a": 1, "b": 3, "c": 4},
		},
		{
			name:      "Basic merge without overwrite",
			dst:       map[string]any{"a": 1, "b": 2},
			src:       map[string]any{"b": 3, "c": 4},
			overwrite: false,
			expected:  map[string]any{"a": 1, "b": 2, "c": 4},
		},
		{
			name:      "Nil dst map",
			dst:       nil,
			src:       map[string]any{"a": 1},
			overwrite: true,
			expected:  map[string]any{"a": 1},
		},
		{
			name:      "Nil src map",
			dst:       map[string]any{"a": 1},
			src:       nil,
			overwrite: true,
			expected:  map[string]any{"a": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMaps(tt.dst, tt.src, tt.overwrite)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMergeMapsDeep(t *testing.T) {
	tests := []struct {
		name      string
		dst       map[string]any
		src       map[string]any
		overwrite bool
		expected  map[string]any
	}{
		{
			name: "Deep merge with overwrite",
			dst: map[string]any{
				"a": 1,
				"b": map[string]any{
					"x": 1,
					"y": 2,
				},
			},
			src: map[string]any{
				"b": map[string]any{
					"y": 3,
					"z": 4,
				},
				"c": 5,
			},
			overwrite: true,
			expected: map[string]any{
				"a": 1,
				"b": map[string]any{
					"x": 1,
					"y": 3,
					"z": 4,
				},
				"c": 5,
			},
		},
		{
			name: "Deep merge without overwrite",
			dst: map[string]any{
				"a": 1,
				"b": map[string]any{
					"x": 1,
				},
			},
			src: map[string]any{
				"b": map[string]any{
					"x": 2,
					"y": 3,
				},
			},
			overwrite: false,
			expected: map[string]any{
				"a": 1,
				"b": map[string]any{
					"x": 1,
					"y": 3,
				},
			},
		},
		{
			name:      "Nil src map",
			dst:       map[string]any{"a": 1},
			src:       nil,
			overwrite: true,
			expected:  map[string]any{"a": 1},
		},
		{
			name:      "Nil dst map",
			dst:       nil,
			src:       map[string]any{"a": 1},
			overwrite: true,
			expected:  map[string]any{"a": 1},
		},
		{
			name:      "Overwrite nested non-map",
			dst:       map[string]any{"a": map[string]any{"x": 1}},
			src:       map[string]any{"a": 2},
			overwrite: true,
			expected:  map[string]any{"a": 2},
		},
		{
			name:      "Do not overwrite nested non-map",
			dst:       map[string]any{"a": map[string]any{"x": 1}},
			src:       map[string]any{"a": 2},
			overwrite: false,
			expected:  map[string]any{"a": map[string]any{"x": 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMapsDeep(tt.dst, tt.src, tt.overwrite)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
