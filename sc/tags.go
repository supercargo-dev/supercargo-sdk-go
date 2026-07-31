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
}

func parseTagRules(f reflect.StructField) (*FieldRuleBuilder, error) {
	tag := f.Tag.Get("supercargo.field")
	if tag == "" {
		return nil, nil
	}

	builder := Field(f.Name)
	parts := []string{}
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(tag); i++ {
		c := tag[i]
		if c == '\'' {
			inQuote = !inQuote
			// We can omit the quote itself from the value, or keep it.
			// Let's omit the quote.
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
		case "greater_than":
			builder = builder.GreaterThan(val)
		case "less_than":
			builder = builder.LessThan(val)
		case "max_length":
			i, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("supercargo: invalid int for max_length on field %s: %w", f.Name, err)
			}
			builder = builder.MaxLength(i)
		case "min_length":
			i, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("supercargo: invalid int for min_length on field %s: %w", f.Name, err)
			}
			builder = builder.MinLength(i)
		case "pattern":
			builder = builder.Pattern(val)
		case "type":
			// parsed correctly, no specific builder method yet, keeping for compatibility
		}
	}
	return builder, nil
}

func parseTagKeyValue(tag string) []string {
	parts := []string{}
	var current strings.Builder
	inQuote := false
	for i := 0; i < len(tag); i++ {
		c := tag[i]
		if c == '\'' || c == '"' {
			inQuote = !inQuote
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
