package clients_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/supercargo-dev/supercargo-sdk-go/clients"
	vaultv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/vault/v1"
)

type mockVaultServer struct {
	vaultv1.UnimplementedVaultServiceServer
	batchCalls          atomic.Int32
	failCount           int32
	chunkSizes          []int
	overrideResp        *vaultv1.BatchTokenizeResponse
	customBatchTokenize func(ctx context.Context, req *vaultv1.BatchTokenizeRequest) (*vaultv1.BatchTokenizeResponse, error)
}

func (m *mockVaultServer) BatchTokenize(ctx context.Context, req *vaultv1.BatchTokenizeRequest) (*vaultv1.BatchTokenizeResponse, error) {
	call := m.batchCalls.Add(1)
	if m.customBatchTokenize != nil {
		return m.customBatchTokenize(ctx, req)
	}
	if m.overrideResp != nil {
		return m.overrideResp, nil
	}
	if call <= m.failCount {
		return nil, status.Error(codes.ResourceExhausted, "quota exceeded temporarily")
	}

	m.chunkSizes = append(m.chunkSizes, len(req.Cascades))
	var results []*vaultv1.EntityCascadeResult
	for _, c := range req.Cascades {
		results = append(results, &vaultv1.EntityCascadeResult{
			ContextId: c.ContextId,
			Tokens: map[string]string{
				"val": "tok-" + c.ContextId,
			},
			IdSearchHash: "hash-" + c.ContextId,
		})
	}

	return &vaultv1.BatchTokenizeResponse{
		Results: results,
	}, nil
}

func startMockVaultServer(t *testing.T, srv *mockVaultServer) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	vaultv1.RegisterVaultServiceServer(s, srv)

	go func() {
		_ = s.Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	return conn, func() {
		_ = conn.Close()
		s.Stop()
	}
}

func TestVaultClient_BatchTokenize_Chunking(t *testing.T) {
	mockSrv := &mockVaultServer{}
	conn, stop := startMockVaultServer(t, mockSrv)
	defer stop()

	client, err := clients.NewVaultClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithChunkSize(500),
		clients.WithMaxRetries(3),
		clients.WithRetryDelay(10*time.Millisecond),
	)
	require.NoError(t, err)
	defer client.Close()

	total := 1250
	cascades := make([]*vaultv1.EntityCascade, total)
	for i := 0; i < total; i++ {
		cascades[i] = &vaultv1.EntityCascade{
			ContextId: fmt.Sprintf("user-%d", i),
			Identifiers: []*vaultv1.EntityIdentifier{
				{
					Urn:   "urn:sc:entity:user",
					Value: fmt.Sprintf("user-%d@example.com", i),
				},
			},
		}
	}

	ctx := context.Background()
	results, err := client.BatchTokenize(ctx, "urn:sc:domain:default", cascades)
	require.NoError(t, err)
	assert.Len(t, results, total)

	// With total=1250 and chunkSize=500, we expect 3 chunks: 500, 500, 250
	assert.Equal(t, int32(3), mockSrv.batchCalls.Load())
	assert.Equal(t, []int{500, 500, 250}, mockSrv.chunkSizes)
}

func TestVaultClient_BatchTokenize_RetrySuccess(t *testing.T) {
	mockSrv := &mockVaultServer{failCount: 1}
	conn, stop := startMockVaultServer(t, mockSrv)
	defer stop()

	client, err := clients.NewVaultClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithMaxRetries(3),
		clients.WithRetryDelay(10*time.Millisecond),
		clients.WithToken("vault-secret-token"),
	)
	require.NoError(t, err)
	defer client.Close()

	cascades := []*vaultv1.EntityCascade{
		{
			ContextId: "user-1",
			Identifiers: []*vaultv1.EntityIdentifier{
				{
					Urn:   "urn:sc:entity:user",
					Value: "user-1@example.com",
				},
			},
		},
	}

	ctx := context.Background()
	results, err := client.BatchTokenize(ctx, "urn:sc:domain:default", cascades)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tok-user-1", results[0].Tokens["val"])
	assert.Equal(t, int32(2), mockSrv.batchCalls.Load())
}

func TestVaultClient_BatchTokenize_Empty(t *testing.T) {
	mockSrv := &mockVaultServer{}
	conn, stop := startMockVaultServer(t, mockSrv)
	defer stop()

	client, err := clients.NewVaultClient("bufnet", clients.WithGRPCConn(conn))
	require.NoError(t, err)
	defer client.Close()

	results, err := client.BatchTokenize(context.Background(), "urn:sc:domain:default", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Equal(t, int32(0), mockSrv.batchCalls.Load())
}

type mockVaultServiceClient struct {
	vaultv1.VaultServiceClient
	batchTokenizeFunc func(ctx context.Context, in *vaultv1.BatchTokenizeRequest, opts ...grpc.CallOption) (*vaultv1.BatchTokenizeResponse, error)
}

func (m *mockVaultServiceClient) BatchTokenize(ctx context.Context, in *vaultv1.BatchTokenizeRequest, opts ...grpc.CallOption) (*vaultv1.BatchTokenizeResponse, error) {
	if m.batchTokenizeFunc != nil {
		return m.batchTokenizeFunc(ctx, in, opts...)
	}
	return nil, nil
}

func TestVaultClient_BatchTokenize_FailClosedAndValidation(t *testing.T) {
	tests := []struct {
		name           string
		cascades       []*vaultv1.EntityCascade
		setupMock      func() *mockVaultServer
		customStub     vaultv1.VaultServiceClient
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "cardinality mismatch: fewer results than chunk",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-1",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:sc:entity:email", Value: "a@example.com"},
					},
				},
				{
					ContextId: "ctx-2",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:sc:entity:email", Value: "b@example.com"},
					},
				},
			},
			setupMock: func() *mockVaultServer {
				return &mockVaultServer{
					overrideResp: &vaultv1.BatchTokenizeResponse{
						Results: []*vaultv1.EntityCascadeResult{
							{
								ContextId: "ctx-1",
								Tokens:    map[string]string{"val": "tok-1"},
							},
						},
					},
				}
			},
			expectErr:      true,
			expectedErrMsg: "vault response cardinality mismatch: expected 2 results, received 1",
		},
		{
			name: "cardinality mismatch: more results than chunk",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-1",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:sc:entity:email", Value: "a@example.com"},
					},
				},
			},
			setupMock: func() *mockVaultServer {
				return &mockVaultServer{
					overrideResp: &vaultv1.BatchTokenizeResponse{
						Results: []*vaultv1.EntityCascadeResult{
							{
								ContextId: "ctx-1",
								Tokens:    map[string]string{"val": "tok-1"},
							},
							{
								ContextId: "ctx-extra",
								Tokens:    map[string]string{"val": "tok-2"},
							},
						},
					},
				}
			},
			expectErr:      true,
			expectedErrMsg: "vault response cardinality mismatch: expected 1 results, received 2",
		},
		{
			name: "context_id mismatch between cascade and result",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "expected-ctx-id",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:sc:entity:email", Value: "a@example.com"},
					},
				},
			},
			setupMock: func() *mockVaultServer {
				return &mockVaultServer{
					overrideResp: &vaultv1.BatchTokenizeResponse{
						Results: []*vaultv1.EntityCascadeResult{
							{
								ContextId: "mismatched-ctx-id",
								Tokens:    map[string]string{"val": "tok-1"},
							},
						},
					},
				}
			},
			expectErr:      true,
			expectedErrMsg: "vault response context_id mismatch: expected 'expected-ctx-id', got 'mismatched-ctx-id'",
		},
		{
			name: "fail-closed: cascade has identifiers but result has empty tokens map",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-1",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:sc:entity:email", Value: "a@example.com"},
					},
				},
			},
			setupMock: func() *mockVaultServer {
				return &mockVaultServer{
					overrideResp: &vaultv1.BatchTokenizeResponse{
						Results: []*vaultv1.EntityCascadeResult{
							{
								ContextId: "ctx-1",
								Tokens:    map[string]string{},
							},
						},
					},
				}
			},
			expectErr:      true,
			expectedErrMsg: "fail-closed: vault returned empty token mapping for cascade context_id 'ctx-1'",
		},
		{
			name: "fail-closed: cascade has identifiers but result has nil tokens map",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-nil-tokens",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:sc:entity:email", Value: "a@example.com"},
					},
				},
			},
			setupMock: func() *mockVaultServer {
				return &mockVaultServer{
					overrideResp: &vaultv1.BatchTokenizeResponse{
						Results: []*vaultv1.EntityCascadeResult{
							{
								ContextId: "ctx-nil-tokens",
								Tokens:    nil,
							},
						},
					},
				}
			},
			expectErr:      true,
			expectedErrMsg: "fail-closed: vault returned empty token mapping for cascade context_id 'ctx-nil-tokens'",
		},
		{
			name: "nil input cascade in request",
			cascades: []*vaultv1.EntityCascade{
				nil,
			},
			setupMock: func() *mockVaultServer {
				return &mockVaultServer{
					overrideResp: &vaultv1.BatchTokenizeResponse{
						Results: []*vaultv1.EntityCascadeResult{
							{
								ContextId: "ctx-any",
								Tokens:    map[string]string{"key": "tok"},
							},
						},
					},
				}
			},
			expectErr:      true,
			expectedErrMsg: "nil input cascade at index 0",
		},
		{
			name: "valid: cascade has no identifiers and result has empty tokens map",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId:   "ctx-empty-identifiers",
					Identifiers: []*vaultv1.EntityIdentifier{},
				},
			},
			setupMock: func() *mockVaultServer {
				return &mockVaultServer{
					overrideResp: &vaultv1.BatchTokenizeResponse{
						Results: []*vaultv1.EntityCascadeResult{
							{
								ContextId: "ctx-empty-identifiers",
								Tokens:    map[string]string{},
							},
						},
					},
				}
			},
			expectErr: false,
		},
		{
			name: "nil response from server",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-1",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:sc:entity:email", Value: "a@example.com"},
					},
				},
			},
			customStub: &mockVaultServiceClient{
				batchTokenizeFunc: func(ctx context.Context, in *vaultv1.BatchTokenizeRequest, opts ...grpc.CallOption) (*vaultv1.BatchTokenizeResponse, error) {
					return nil, nil
				},
			},
			expectErr:      true,
			expectedErrMsg: "received nil response from vault service",
		},
		{
			name: "nil cascade result item in results slice",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-1",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:sc:entity:email", Value: "a@example.com"},
					},
				},
			},
			customStub: &mockVaultServiceClient{
				batchTokenizeFunc: func(ctx context.Context, in *vaultv1.BatchTokenizeRequest, opts ...grpc.CallOption) (*vaultv1.BatchTokenizeResponse, error) {
					return &vaultv1.BatchTokenizeResponse{
						Results: []*vaultv1.EntityCascadeResult{nil},
					}, nil
				},
			},
			expectErr:      true,
			expectedErrMsg: "nil cascade result at index 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *clients.VaultClient
			var mockSrv *mockVaultServer
			var stop func()
			if tt.customStub != nil {
				c, err := clients.NewVaultClient(
					"stub",
					clients.WithVaultStub(tt.customStub),
					clients.WithMaxRetries(3),
					clients.WithRetryDelay(5*time.Millisecond),
				)
				require.NoError(t, err)
				client = c
				stop = func() {}
			} else {
				mockSrv = tt.setupMock()
				conn, s := startMockVaultServer(t, mockSrv)
				stop = s
				c, err := clients.NewVaultClient(
					"bufnet",
					clients.WithGRPCConn(conn),
					clients.WithMaxRetries(3),
					clients.WithRetryDelay(5*time.Millisecond),
				)
				require.NoError(t, err)
				client = c
			}
			defer stop()
			defer client.Close()

			results, err := client.BatchTokenize(context.Background(), "urn:sc:domain:default", tt.cascades)
			if tt.expectErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, clients.ErrSystemUnavailable), "expected error to wrap ErrSystemUnavailable, got: %v", err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, results)
				if mockSrv != nil {
					assert.Equal(t, int32(1), mockSrv.batchCalls.Load())
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, results, len(tt.cascades))
			}
		})
	}
}

