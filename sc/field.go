package sc

// FieldRuleBuilder is a fluent builder for defining rules on a specific field.
type FieldRuleBuilder struct {
	fieldName         string
	regexPattern      string
	maxLen            *int
	minLen            *int
	greater           *string
	less              *string
	greaterOrEqual    *string
	lessOrEqual       *string
	oneof             []string
	mustNotBeEmpty    bool
	isPII             *bool
	ctxID             string
	identityDomainURI string
	identifierRank    *int
}

// ruleMarker implements the Rule interface.
func (f *FieldRuleBuilder) ruleMarker() {}

// Clone performs a deep copy of the FieldRuleBuilder to prevent shared-state poisoning.
func (f *FieldRuleBuilder) Clone() *FieldRuleBuilder {
	c := *f
	if f.maxLen != nil {
		v := *f.maxLen
		c.maxLen = &v
	}
	if f.minLen != nil {
		v := *f.minLen
		c.minLen = &v
	}
	if f.greater != nil {
		v := *f.greater
		c.greater = &v
	}
	if f.less != nil {
		v := *f.less
		c.less = &v
	}
	if f.greaterOrEqual != nil {
		v := *f.greaterOrEqual
		c.greaterOrEqual = &v
	}
	if f.lessOrEqual != nil {
		v := *f.lessOrEqual
		c.lessOrEqual = &v
	}
	if f.oneof != nil {
		c.oneof = make([]string, len(f.oneof))
		copy(c.oneof, f.oneof)
	}
	if f.isPII != nil {
		v := *f.isPII
		c.isPII = &v
	}
	if f.identifierRank != nil {
		v := *f.identifierRank
		c.identifierRank = &v
	}
	return &c
}

// Field starts a new rule builder for the specified field name.
func Field(name string) *FieldRuleBuilder {
	return &FieldRuleBuilder{
		fieldName: name,
	}
}

// NotEmpty enforces that the value is not empty.
func (b *FieldRuleBuilder) NotEmpty() *FieldRuleBuilder {
	c := b.Clone()
	c.mustNotBeEmpty = true
	return c
}

// MaxLength enforces a maximum length (rune count for strings, elements for collections).
func (b *FieldRuleBuilder) MaxLength(max int) *FieldRuleBuilder {
	c := b.Clone()
	c.maxLen = &max
	return c
}

// MinLength enforces a minimum length (rune count for strings, elements for collections).
func (b *FieldRuleBuilder) MinLength(min int) *FieldRuleBuilder {
	c := b.Clone()
	c.minLen = &min
	return c
}

// Pattern enforces that the string value matches the given regular expression.
func (b *FieldRuleBuilder) Pattern(regex string) *FieldRuleBuilder {
	c := b.Clone()
	c.regexPattern = regex
	return c
}

// GreaterThan enforces that the value is strictly greater than the given bound.
func (b *FieldRuleBuilder) GreaterThan(bound string) *FieldRuleBuilder {
	c := b.Clone()
	c.greater = &bound
	return c
}

// GreaterThanOrEqual enforces that the value is greater than or equal to the given bound.
func (b *FieldRuleBuilder) GreaterThanOrEqual(bound string) *FieldRuleBuilder {
	c := b.Clone()
	c.greaterOrEqual = &bound
	return c
}

// LessThan enforces that the value is strictly less than the given bound.
func (b *FieldRuleBuilder) LessThan(bound string) *FieldRuleBuilder {
	c := b.Clone()
	c.less = &bound
	return c
}

// LessThanOrEqual enforces that the value is less than or equal to the given bound.
func (b *FieldRuleBuilder) LessThanOrEqual(bound string) *FieldRuleBuilder {
	c := b.Clone()
	c.lessOrEqual = &bound
	return c
}

// OneOf enforces that the value matches one of the specified allowed values.
func (b *FieldRuleBuilder) OneOf(values ...string) *FieldRuleBuilder {
	c := b.Clone()
	c.oneof = append([]string(nil), values...)
	return c
}

// PII marks the field as containing Personally Identifiable Information.
func (b *FieldRuleBuilder) PII(isPII bool) *FieldRuleBuilder {
	c := b.Clone()
	c.isPII = &isPII
	return c
}

// ContextID tags the field with a specific context identifier.
func (b *FieldRuleBuilder) ContextID(ctxID string) *FieldRuleBuilder {
	c := b.Clone()
	c.ctxID = ctxID
	return c
}

// IdentityDomain tags the field with an identity domain URI.
func (b *FieldRuleBuilder) IdentityDomain(domainURI string) *FieldRuleBuilder {
	c := b.Clone()
	c.identityDomainURI = domainURI
	return c
}

// Rank tags the field with an identifier rank (e.g., 1 for primary key).
func (b *FieldRuleBuilder) Rank(rank int) *FieldRuleBuilder {
	c := b.Clone()
	c.identifierRank = &rank
	return c
}

