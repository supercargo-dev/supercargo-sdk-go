package sc

import "fmt"

// Rule represents a single constraint or metadata definition.
type Rule interface {
	ruleMarker() // private marker
}

// ContractValidator is an interface that DataContracts can implement
// to provide strongly-typed validation rules and complex constraints
// without relying purely on string-based struct tags.
type ContractValidator interface {
	ContractConstraints() []Rule
}

// ContractMetadata represents contract-level metadata extracted from supercargo.contract tags.
type ContractMetadata struct {
	URN         string
	Version     string
	OwnerTeam   string
	DataAsset   string
	Description string
}

// Schema represents the complete IR schema extracted from a struct type.
type Schema struct {
	Contract ContractMetadata
	Fields   map[string]FieldMetadata
}

// ValidationError represents a structured validation failure.
type ValidationError struct {
	Field  string
	Reason string
	Value  any
	Err    error // Underlying error, if any
	IsPII  bool  // Marks if the field contains PII to redact values
}

func (v *ValidationError) Error() string {
	if v.IsPII {
		if v.Err != nil {
			return fmt.Sprintf("validation failed on field %q: %s (underlying error contains redacted PII)", v.Field, v.Reason)
		}
		return fmt.Sprintf("validation failed on field %q: %s (got: [REDACTED PII])", v.Field, v.Reason)
	}
	if v.Err != nil {
		return fmt.Sprintf("validation failed on field %q: %s (%v)", v.Field, v.Reason, v.Err)
	}
	if v.Value != nil {
		return fmt.Sprintf("validation failed on field %q: %s (got: %v)", v.Field, v.Reason, v.Value)
	}
	return fmt.Sprintf("validation failed on field %q: %s", v.Field, v.Reason)
}

func (v *ValidationError) Unwrap() error {
	if v.IsPII {
		return fmt.Errorf("underlying error contains redacted PII")
	}
	return v.Err
}
