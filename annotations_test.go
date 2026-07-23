package supercargo

import "testing"

func TestDataTypeHints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{name: "HintNone", actual: string(HintNone), expected: "NONE"},
		{name: "HintTimestamp", actual: string(HintTimestamp), expected: "TIMESTAMP"},
		{name: "HintMapStringString", actual: string(HintMapStringString), expected: "MAP_STRING_STRING"},
		{name: "HintUUID", actual: string(HintUUID), expected: "UUID"},
		{name: "TagKey", actual: TagKey, expected: "supercargo"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.actual != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.actual)
			}
		})
	}
}
