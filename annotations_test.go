package supercargo

import "testing"

func TestDataTypeHints(t *testing.T) {
	if HintTimestamp != "TIMESTAMP" {
		t.Errorf("expected TIMESTAMP, got %v", HintTimestamp)
	}
	if TagKey != "supercargo" {
		t.Errorf("expected supercargo, got %v", TagKey)
	}
}
