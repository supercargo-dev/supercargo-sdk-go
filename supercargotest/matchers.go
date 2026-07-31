package supercargotest

import (
	"strings"
	"testing"

	"github.com/supercargo-dev/supercargo-sdk-go/sc"
)

// AssertValidates asserts that the given payload passes its defined ContractConstraints.
func AssertValidates(t *testing.T, payload any) {
	t.Helper()
	err := sc.Validate(payload)
	if err != nil {
		t.Fatalf("expected payload to pass validation, but got error: %v", err)
	}
}

// AssertFailsValidation asserts that the given payload fails validation and the error contains the expected string.
func AssertFailsValidation(t *testing.T, payload any, expectedError string) {
	t.Helper()
	err := sc.Validate(payload)
	if err == nil {
		t.Fatalf("expected payload to fail validation, but it passed")
	}
	if !strings.Contains(err.Error(), expectedError) {
		t.Fatalf("expected validation error to contain %q, but got: %v", expectedError, err)
	}
}
