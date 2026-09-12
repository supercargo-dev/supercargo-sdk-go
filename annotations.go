package supercargo

// DataTypeHint defines the basic types for the Supercargo schema.
type DataTypeHint string

const (
	// HintNone represents no specific type hint.
	HintNone DataTypeHint = "NONE"
	// HintTimestamp represents a timestamp data type.
	HintTimestamp DataTypeHint = "TIMESTAMP"
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
