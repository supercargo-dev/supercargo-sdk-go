package sc_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supercargo-dev/supercargo-sdk-go/sc"
)

// Golden UX Contract DTO
type UserSignupEvent struct {
	_         struct{}  `supercargo:"urn=urn:sc:contract:user_signup:v1,version=1.0.0"`
	UserID    string    `supercargo:"primary_key,rank=1" json:"user_id"`
	Email     string    `supercargo:"pii,context_id=user_id,domain=urn:sc:domain:users" json:"email"`
	Score     int       `supercargo:"gte=0,lte=100" json:"score"`
	CreatedAt time.Time `supercargo:"type=TIMESTAMP" json:"created_at"`
}

type DualTagOverrideEvent struct {
	_      struct{} `supercargo.contract:"urn=urn:sc:contract:dual_tag:v1,version=1.0.0"`
	Status string   `supercargo:"oneof=draft published" supercargo.field:"oneof=draft archived" json:"status"`
	Limit  int      `validate:"min=1" supercargo:"gte=5" supercargo.field:"gte=10" json:"limit"`
}

func TestUserSignupEvent_Subcutaneous_HappyPath(t *testing.T) {
	t.Parallel()

	event := UserSignupEvent{
		UserID:    "usr_12345",
		Email:     "user@example.com",
		Score:     85,
		CreatedAt: time.Now(),
	}

	err := sc.Validate(event)
	require.NoError(t, err)

	errPtr := sc.Validate(&event)
	require.NoError(t, errPtr)

	// Verify contract-level metadata compiled via fallback to supercargo tag on marker struct
	v, err := sc.Compile(event)
	require.NoError(t, err)
	require.NotNil(t, v)

	contract := v.Contract()
	assert.Equal(t, "urn:sc:contract:user_signup:v1", contract.URN)
	assert.Equal(t, "1.0.0", contract.Version)

	// Verify field metadata
	metadata := v.Metadata()

	userMeta, ok := metadata["UserID"]
	require.True(t, ok, "UserID metadata must exist")
	require.NotNil(t, userMeta.IsPrimaryKey)
	assert.True(t, *userMeta.IsPrimaryKey)
	require.NotNil(t, userMeta.IdentifierRank)
	assert.Equal(t, 1, *userMeta.IdentifierRank)

	emailMeta, ok := metadata["Email"]
	require.True(t, ok, "Email metadata must exist")
	require.NotNil(t, emailMeta.IsPII)
	assert.True(t, *emailMeta.IsPII)
	assert.Equal(t, "user_id", emailMeta.ContextID)
	assert.Equal(t, "urn:sc:domain:users", emailMeta.IdentityDomainURI)
}

func TestUserSignupEvent_Subcutaneous_NegativeErrorPath(t *testing.T) {
	t.Parallel()

	invalidEvent := UserSignupEvent{
		UserID:    "usr_12345",
		Email:     "user@example.com",
		Score:     -5, // fails gte=0
		CreatedAt: time.Now(),
	}

	err := sc.Validate(invalidEvent)
	require.Error(t, err)

	var valErr *sc.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "Score", valErr.Field)
	assert.Contains(t, err.Error(), "Score")
	assert.Contains(t, err.Error(), "greater than or equal to 0")
}

func TestDualTagOverride_Subcutaneous_Precedence(t *testing.T) {
	t.Parallel()

	// supercargo.field specifies oneof=draft archived; supercargo specifies oneof=draft published
	// "published" should fail because supercargo.field takes precedence
	invalidStatus := DualTagOverrideEvent{
		Status: "published",
		Limit:  15,
	}
	err := sc.Validate(invalidStatus)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Status")

	// "archived" passes supercargo.field
	validEvent := DualTagOverrideEvent{
		Status: "archived",
		Limit:  15,
	}
	require.NoError(t, sc.Validate(validEvent))

	// Limit: supercargo.field gte=10 overrides supercargo gte=5 and validate min=1
	limitFailing := DualTagOverrideEvent{
		Status: "archived",
		Limit:  7, // passes validate (>=1) and supercargo (>=5), but fails supercargo.field (>=10)
	}
	errLimit := sc.Validate(limitFailing)
	require.Error(t, errLimit)
	assert.Contains(t, errLimit.Error(), "Limit")
	assert.Contains(t, errLimit.Error(), "greater than or equal to 10")
}

func TestCanonicalTypeMatrix_Subcutaneous(t *testing.T) {
	t.Parallel()

	type MatrixRecord struct {
		StringField  string    `supercargo:"not_empty" json:"str"`
		Int64Field   int64     `supercargo:"gte=0" json:"i64"`
		Float64Field float64   `supercargo:"gte=0.0,lte=100.0" json:"f64"`
		BoolField    bool      `json:"b"`
		BytesField   []byte    `supercargo:"not_empty" json:"bytes"`
		Timestamp    time.Time `supercargo:"type=TIMESTAMP" json:"ts"`
		ListField    []string  `supercargo:"max=5" json:"list"`
	}

	tests := []struct {
		name    string
		modify  func(r *MatrixRecord)
		wantErr bool
	}{
		{
			name: "valid baseline zero-values where permitted",
			modify: func(r *MatrixRecord) {
				r.StringField = "valid"
				r.Int64Field = 0
				r.Float64Field = 0.0
				r.BoolField = false
				r.BytesField = []byte{0x01}
				r.Timestamp = time.Now()
				r.ListField = []string{"a"}
			},
			wantErr: false,
		},
		{
			name: "empty string fails not_empty",
			modify: func(r *MatrixRecord) {
				r.StringField = ""
				r.Int64Field = 0
				r.Float64Field = 0.0
				r.BytesField = []byte{0x01}
			},
			wantErr: true,
		},
		{
			name: "negative int64 fails gte=0",
			modify: func(r *MatrixRecord) {
				r.StringField = "valid"
				r.Int64Field = -1
				r.Float64Field = 0.0
				r.BytesField = []byte{0x01}
			},
			wantErr: true,
		},
		{
			name: "empty bytes slice fails not_empty",
			modify: func(r *MatrixRecord) {
				r.StringField = "valid"
				r.Int64Field = 0
				r.Float64Field = 0.0
				r.BytesField = []byte{}
			},
			wantErr: true,
		},
		{
			name: "list exceeding max items fails",
			modify: func(r *MatrixRecord) {
				r.StringField = "valid"
				r.Int64Field = 0
				r.Float64Field = 0.0
				r.BytesField = []byte{0x01}
				r.ListField = []string{"1", "2", "3", "4", "5", "6"}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var r MatrixRecord
			tt.modify(&r)
			err := sc.Validate(r)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
