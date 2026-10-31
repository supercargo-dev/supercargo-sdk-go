package supercargo

// DataTypeHint defines the basic types for the Supercargo schema.
type DataTypeHint string

const (
	// Canonical DataTypeHint constants corresponding to hub.v1.DataType
	DataTypeString     DataTypeHint = "STRING"
	DataTypeBytes      DataTypeHint = "BYTES"
	DataTypeInt64      DataTypeHint = "INT64"
	DataTypeFloat64    DataTypeHint = "FLOAT64"
	DataTypeBool       DataTypeHint = "BOOL"
	DataTypeTimestamp  DataTypeHint = "TIMESTAMP"
	DataTypeDate       DataTypeHint = "DATE"
	DataTypeTime       DataTypeHint = "TIME"
	DataTypeDatetime   DataTypeHint = "DATETIME"
	DataTypeGeography  DataTypeHint = "GEOGRAPHY"
	DataTypeNumeric    DataTypeHint = "NUMERIC"
	DataTypeBignumeric DataTypeHint = "BIGNUMERIC"
	DataTypeStruct     DataTypeHint = "STRUCT"
	DataTypeJson       DataTypeHint = "JSON"
	DataTypeVector     DataTypeHint = "VECTOR"

	// HintNone represents no specific type hint.
	HintNone DataTypeHint = "NONE"
	// HintTimestamp represents a timestamp data type (alias to DataTypeTimestamp).
	HintTimestamp = DataTypeTimestamp
	// HintMapStringString represents a map of string to string.
	HintMapStringString DataTypeHint = "MAP_STRING_STRING"
	// HintUUID represents a UUID.
	HintUUID DataTypeHint = "UUID"
)

// Common tag keys for Supercargo struct tags.
const (
	TagKey       = "supercargo"
	TagEntityKey = "supercargo.entity"
)

// Visibility defines the visibility tier of a field across domain boundaries.
type Visibility string

const (
	// VisibilityPublic exposes the field across all boundaries.
	VisibilityPublic Visibility = "public"
	// VisibilityInternal limits field visibility to internal domain services.
	VisibilityInternal Visibility = "internal"
	// VisibilityDomain limits field visibility strictly within its owning domain.
	VisibilityDomain Visibility = "domain"
	// VisibilityPrivate marks the field as private to the producing service.
	VisibilityPrivate Visibility = "private"
)

// SensitivityLevel defines the confidentiality tier of a field or contract.
type SensitivityLevel string

const (
	// SensitivityPublic marks data intended for public consumption.
	SensitivityPublic SensitivityLevel = "public"
	// SensitivityInternal marks data restricted to internal organizational use.
	SensitivityInternal SensitivityLevel = "internal"
	// SensitivityConfidential marks sensitive business or personal data.
	SensitivityConfidential SensitivityLevel = "confidential"
	// SensitivityRestricted marks highly sensitive or regulated data.
	SensitivityRestricted SensitivityLevel = "restricted"
)
