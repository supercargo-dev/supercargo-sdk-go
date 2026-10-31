package sc

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTagRules_SupercargoTag(t *testing.T) {
	t.Parallel()

	type Sample struct {
		Name string `supercargo:"not_empty,min_length=3,max_length=50"`
		Rank int    `supercargo:"primary_key,rank=1,pii"`
	}

	typ := reflect.TypeOf(Sample{})

	// 1. Name field
	nameField := typ.Field(0)
	builder, err := parseTagRules(nameField)
	require.NoError(t, err)
	require.NotNil(t, builder)
	assert.True(t, builder.mustNotBeEmpty)
	require.NotNil(t, builder.minLen)
	assert.Equal(t, 3, *builder.minLen)
	require.NotNil(t, builder.maxLen)
	assert.Equal(t, 50, *builder.maxLen)

	// 2. Rank field (metadata extraction)
	rankField := typ.Field(1)
	builderRank, err := parseTagRules(rankField)
	require.NoError(t, err)
	require.NotNil(t, builderRank)
	require.NotNil(t, builderRank.isPrimaryKey)
	assert.True(t, *builderRank.isPrimaryKey)
	require.NotNil(t, builderRank.identifierRank)
	assert.Equal(t, 1, *builderRank.identifierRank)
	require.NotNil(t, builderRank.isPII)
	assert.True(t, *builderRank.isPII)
}

func TestParseTagRules_DualTagPrecedence(t *testing.T) {
	t.Parallel()

	type OverrideSample struct {
		// validate: min=2, supercargo: min=5, supercargo.field: min=10
		// supercargo.field must win!
		FieldA string `validate:"min=2" supercargo:"min_length=5" supercargo.field:"min_length=10"`
		// validate: min=1, supercargo: min=5 (supercargo overrides validate)
		FieldB string `validate:"min=1" supercargo:"min_length=5"`
		// oneof override: supercargo.field overrides supercargo
		Role string `supercargo:"oneof=guest user" supercargo.field:"oneof=admin superadmin"`
	}

	typ := reflect.TypeOf(OverrideSample{})

	// FieldA: supercargo.field wins
	fieldA := typ.Field(0)
	builderA, err := parseTagRules(fieldA)
	require.NoError(t, err)
	require.NotNil(t, builderA)
	require.NotNil(t, builderA.minLen)
	assert.Equal(t, 10, *builderA.minLen)

	// FieldB: supercargo overrides validate
	fieldB := typ.Field(1)
	builderB, err := parseTagRules(fieldB)
	require.NoError(t, err)
	require.NotNil(t, builderB)
	require.NotNil(t, builderB.minLen)
	assert.Equal(t, 5, *builderB.minLen)

	// Role: supercargo.field oneof wins
	fieldRole := typ.Field(2)
	builderRole, err := parseTagRules(fieldRole)
	require.NoError(t, err)
	require.NotNil(t, builderRole)
	assert.Equal(t, []string{"admin", "superadmin"}, builderRole.oneof)
}

func TestParseTagRules_MarkerStructIgnored(t *testing.T) {
	t.Parallel()

	type ContractMarker struct {
		_    struct{} `supercargo:"urn=urn:sc:contract:test:v1,version=1.0.0"`
		Data string   `supercargo:"not_empty"`
	}

	typ := reflect.TypeOf(ContractMarker{})

	// Marker field '_' must return nil rule
	markerField := typ.Field(0)
	builder, err := parseTagRules(markerField)
	require.NoError(t, err)
	assert.Nil(t, builder, "marker struct field '_' must not produce a field rule")

	// Data field returns rule
	dataField := typ.Field(1)
	dataBuilder, err := parseTagRules(dataField)
	require.NoError(t, err)
	require.NotNil(t, dataBuilder)
	assert.True(t, dataBuilder.mustNotBeEmpty)
}

func TestParseTagRules_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		tag         reflect.StructTag
		kind        reflect.Kind
		errContains string
	}{
		{
			name:        "invalid rank int",
			tag:         `supercargo:"rank=abc"`,
			kind:        reflect.Int,
			errContains: "invalid int for rank",
		},
		{
			name:        "invalid primary_key bool",
			tag:         `supercargo:"primary_key=not-a-bool"`,
			kind:        reflect.String,
			errContains: "invalid boolean for primary_key",
		},
		{
			name:        "invalid max_length int",
			tag:         `supercargo:"max=xyz"`,
			kind:        reflect.String,
			errContains: "invalid int for max_length",
		},
		{
			name:        "invalid min_length int",
			tag:         `supercargo:"min=xyz"`,
			kind:        reflect.String,
			errContains: "invalid int for min_length",
		},
		{
			name:        "invalid sort_rank int",
			tag:         `supercargo:"sort_rank=-1"`,
			kind:        reflect.Int,
			errContains: "must be non-negative integer",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sf := reflect.StructField{
				Name: "TestField",
				Type: reflect.TypeOf(""),
				Tag:  tt.tag,
			}
			_, err := parseTagRules(sf)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}
