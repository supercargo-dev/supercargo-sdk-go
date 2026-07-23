package supercargo

import "testing"

func TestDataTypeHints(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"HintNone", string(HintNone), "NONE"},
		{"HintTimestamp", string(HintTimestamp), "TIMESTAMP"},
		{"HintMapStringString", string(HintMapStringString), "MAP_STRING_STRING"},
		{"HintUUID", string(HintUUID), "UUID"},
		{"TagKey", TagKey, "supercargo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.actual)
			}
		})
	}
}
