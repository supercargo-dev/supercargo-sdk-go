package supercargo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestSensitivityConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, SensitivityLevel("public"), SensitivityPublic)
	assert.Equal(t, SensitivityLevel("internal"), SensitivityInternal)
	assert.Equal(t, SensitivityLevel("confidential"), SensitivityConfidential)
	assert.Equal(t, SensitivityLevel("restricted"), SensitivityRestricted)
}

func TestSensitivityStructTag(t *testing.T) {
	t.Parallel()
	type TestContract struct {
		PublicData       string `supercargo:"sensitivity=public"`
		InternalData     string `supercargo:"sensitivity=internal"`
		ConfidentialData string `supercargo:"sensitivity=confidential,pii=true"`
		RestrictedData   string `supercargo:"sensitivity=restricted"`
	}

	record := TestContract{
		PublicData:       "pub",
		InternalData:     "int",
		ConfidentialData: "conf",
		RestrictedData:   "rest",
	}

	assert.Equal(t, "pub", record.PublicData)
	assert.Equal(t, "int", record.InternalData)
	assert.Equal(t, "conf", record.ConfidentialData)
	assert.Equal(t, "rest", record.RestrictedData)
}

