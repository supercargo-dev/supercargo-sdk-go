package supercargo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/vault/v1"
)

func TestSearchHashProtoValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     *vaultv1.SearchHashRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &vaultv1.SearchHashRequest{
				IdentityDomainUrn: "urn:sc:hub:identity_domain:rockify",
				EntityUrn:         "urn:supercargo:entity:user:email",
				Value:             "alice@rockify.io",
			},
			wantErr: false,
		},
		{
			name: "invalid identity domain urn",
			req: &vaultv1.SearchHashRequest{
				IdentityDomainUrn: "invalid-urn",
				EntityUrn:         "urn:supercargo:entity:user:email",
				Value:             "alice@rockify.io",
			},
			wantErr: true,
		},
		{
			name: "empty value",
			req: &vaultv1.SearchHashRequest{
				IdentityDomainUrn: "urn:sc:hub:identity_domain:rockify",
				EntityUrn:         "urn:supercargo:entity:user:email",
				Value:             "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.req.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	resp := &vaultv1.SearchHashResponse{
		SearchHash:   "abc",
		IdSearchHash: "def",
		AnonId:       "urn:sc:entity:anon:123",
		Registered:   true,
	}
	assert.Equal(t, "abc", resp.GetSearchHash())
	assert.True(t, resp.GetRegistered())
}
