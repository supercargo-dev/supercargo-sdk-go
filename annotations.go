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
	TagKey = "supercargo"
)
