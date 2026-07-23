package supercargo

import "testing"

func TestDataTypeHints(t *testing.T) {
	if HintNone != "NONE" {
		t.Errorf("expected NONE, got %v", HintNone)
	}
	if HintTimestamp != "TIMESTAMP" {
		t.Errorf("expected TIMESTAMP, got %v", HintTimestamp)
	}
	if HintMapStringString != "MAP_STRING_STRING" {
		t.Errorf("expected MAP_STRING_STRING, got %v", HintMapStringString)
	}
	if HintUUID != "UUID" {
		t.Errorf("expected UUID, got %v", HintUUID)
	}
	if TagKey != "supercargo" {
		t.Errorf("expected supercargo, got %v", TagKey)
	}
}
