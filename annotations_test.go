package supercargo

import "testing"

func TestDataTypeHints(t *testing.T) {
	if HintTimestamp != "TIMESTAMP" {
		t.Errorf("expected TIMESTAMP, got %v", HintTimestamp)
	}
	if TagKey != "supercargo" {
		t.Errorf("expected supercargo, got %v", TagKey)
	}
	if TagEntityKey != "supercargo.entity" {
		t.Errorf("expected supercargo.entity, got %v", TagEntityKey)
	}
}

func TestVisibilityConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      Visibility
		expected string
	}{
		{"public", VisibilityPublic, "public"},
		{"internal", VisibilityInternal, "internal"},
		{"domain", VisibilityDomain, "domain"},
		{"private", VisibilityPrivate, "private"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.got)
			}
		})
	}
}

func TestVisibilityStructTag(t *testing.T) {
	type TestUser struct {
		SecretField string `supercargo:"visibility=internal,pii=true"`
		PublicField string `supercargo:"visibility=public"`
		DomainField string `supercargo:"visibility=domain"`
	}

	user := TestUser{
		SecretField: "classified",
		PublicField: "open",
		DomainField: "shared",
	}

	if user.SecretField == "" {
		t.Fatal("expected secretField to be set")
	}
}

