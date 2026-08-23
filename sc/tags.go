package sc

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

// FieldMetadata represents extracted metadata from a field constraint.
type FieldMetadata struct {
	IsPII             *bool
	ContextID         string
	IdentityDomainURI string
	IdentifierRank    *int
	IsPrimaryKey      *bool
	SortRank          *int
}

func parseTagRules(f reflect.StructField) (*FieldRuleBuilder, error) {
	scTag := f.Tag.Get("supercargo.field")
	valTag := f.Tag.Get("validate")
	if scTag == "" && valTag == "" {
		return nil, nil
	}

	builder := Field(f.Name)

	// Determine type kind for type-aware constraint mapping
	t := f.Type
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	kind := t.Kind()
	isString := kind == reflect.String
	isCollection := kind == reflect.Slice || kind == reflect.Array || kind == reflect.Map

	// 1. Parse standard 'validate' tag first (if present)
	if valTag != "" {
		parts := parseTagKeyValue(valTag)
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := ""
			if len(kv) > 1 {
				val = strings.TrimSpace(kv[1])
			}

			switch key {
			case "required":
				builder = builder.NotEmpty()
			case "email":
				builder = builder.Pattern(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
			case "min":
				if isString || isCollection {
					i, err := strconv.Atoi(val)
					if err != nil {
						return nil, fmt.Errorf("supercargo: invalid int for min on field %s: %w", f.Name, err)
					}
					builder = builder.MinLength(i)
				} else {
					builder = builder.GreaterThanOrEqual(val)
				}
			case "max":
				if isString || isCollection {
					i, err := strconv.Atoi(val)
					if err != nil {
						return nil, fmt.Errorf("supercargo: invalid int for max on field %s: %w", f.Name, err)
					}
					builder = builder.MaxLength(i)
				} else {
					builder = builder.LessThanOrEqual(val)
				}
			case "len":
				if isString || isCollection {
					i, err := strconv.Atoi(val)
					if err != nil {
						return nil, fmt.Errorf("supercargo: invalid int for len on field %s: %w", f.Name, err)
					}
					builder = builder.MinLength(i).MaxLength(i)
				} else {
					builder = builder.GreaterThanOrEqual(val).LessThanOrEqual(val)
				}
			case "gte", "greater_than_or_equal_to":
				builder = builder.GreaterThanOrEqual(val)
			case "lte", "less_than_or_equal_to":
				builder = builder.LessThanOrEqual(val)
			case "gt", "greater_than":
				builder = builder.GreaterThan(val)
			case "lt", "less_than":
				builder = builder.LessThan(val)
			case "oneof":
				vals := parseOneOfValues(val)
				builder = builder.OneOf(vals...)
			}
		}
	}

	// 2. Parse sovereign 'supercargo.field' tag (explicit overrides)
	if scTag != "" {
		parts := parseTagKeyValue(scTag)
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := ""
			if len(kv) > 1 {
				val = strings.TrimSpace(kv[1])
			}

			switch key {
			case "not_empty":
				builder = builder.NotEmpty()
			case "pii":
				b, err := strconv.ParseBool(val)
				if err != nil {
					return nil, fmt.Errorf("supercargo: invalid boolean for pii on field %s: %w", f.Name, err)
				}
				builder = builder.PII(b)
			case "rank":
				i, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("supercargo: invalid int for rank on field %s: %w", f.Name, err)
				}
				builder = builder.Rank(i)
			case "context_id":
				builder = builder.ContextID(val)
			case "domain", "identity_domain":
				builder = builder.IdentityDomain(val)
			case "greater_than", "gt":
				builder = builder.GreaterThan(val)
			case "less_than", "lt":
				builder = builder.LessThan(val)
			case "greater_than_or_equal_to", "gte":
				builder = builder.GreaterThanOrEqual(val)
			case "less_than_or_equal_to", "lte":
				builder = builder.LessThanOrEqual(val)
			case "max_length", "max":
				i, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("supercargo: invalid int for max_length on field %s: %w", f.Name, err)
				}
				builder = builder.MaxLength(i)
			case "min_length", "min":
				i, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("supercargo: invalid int for min_length on field %s: %w", f.Name, err)
				}
				builder = builder.MinLength(i)
			case "len":
				i, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("supercargo: invalid int for len on field %s: %w", f.Name, err)
				}
				builder = builder.MinLength(i).MaxLength(i)
			case "pattern":
				builder = builder.Pattern(val)
			case "oneof":
				vals := parseOneOfValues(val)
				builder = builder.OneOf(vals...)
			case "primary_key", "pk":
				b := true
				if val != "" {
					parsed, err := strconv.ParseBool(val)
					if err != nil {
						return nil, fmt.Errorf("supercargo: invalid boolean for primary_key on field %s: %w", f.Name, err)
					}
					b = parsed
				}
				builder = builder.PrimaryKey(b)
			case "sort_rank", "sort_key":
				i, err := strconv.Atoi(val)
				if err != nil || i < 0 {
					return nil, fmt.Errorf("supercargo: invalid int for sort_rank on field %s: must be non-negative integer", f.Name)
				}
				builder = builder.SortRank(i)
			case "type":
				// parsed correctly, keeping for compatibility
			}
		}
	}

	return builder, nil
}

func parseOneOfValues(s string) []string {
	vals := make([]string, 0, 4)
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c == '\'' || c == '"') && !inQuote {
			inQuote = true
			quoteChar = c
		} else if inQuote && c == quoteChar {
			inQuote = false
			quoteChar = 0
		} else if !inQuote && (c == ' ' || c == '\t' || c == ',') {
			if current.Len() > 0 {
				vals = append(vals, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		vals = append(vals, current.String())
	}
	return vals
}

func parseTagKeyValue(tag string) []string {
	parts := make([]string, 0, 4)
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)
	for i := 0; i < len(tag); i++ {
		c := tag[i]
		if (c == '\'' || c == '"') && !inQuote {
			inQuote = true
			quoteChar = c
		} else if inQuote && c == quoteChar {
			inQuote = false
			quoteChar = 0
		} else if c == ',' && !inQuote {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

type numericBound struct {
	hasI64 bool
	i64    int64
	hasF64 bool
	f64    float64
	bf     *big.Float
	str    string
}

func parseNumericBound(s string) (*numericBound, error) {
	bf, _, err := big.ParseFloat(s, 10, 0, big.ToNearestEven)
	if err != nil {
		return nil, err
	}
	b := &numericBound{bf: bf, str: s}

	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		b.hasI64 = true
		b.i64 = i
	} else if f, err := strconv.ParseFloat(s, 64); err == nil {
		bfFloat := big.NewFloat(f)
		if bfFloat.Cmp(bf) == 0 {
			b.hasF64 = true
			b.f64 = f
		}
	}
	return b, nil
}

func (b *numericBound) cmp(s string) (int, error) {
	if b.hasI64 {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			if i < b.i64 {
				return -1, nil
			} else if i > b.i64 {
				return 1, nil
			}
			return 0, nil
		}
	}
	if b.hasF64 || b.hasI64 {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			// If it's short enough to not lose precision in float64
			if len(s) < 15 {
				boundF := b.f64
				if b.hasI64 {
					boundF = float64(b.i64)
				}
				if f < boundF {
					return -1, nil
				} else if f > boundF {
					return 1, nil
				}
				return 0, nil
			}
		}
	}
	// Fallback to big.Float
	parsed, _, err := big.ParseFloat(s, 10, 0, big.ToNearestEven)
	if err != nil {
		return 0, err
	}
	return parsed.Cmp(b.bf), nil
}
