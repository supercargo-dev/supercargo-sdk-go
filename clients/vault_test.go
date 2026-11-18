package clients_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/supercargo-dev/supercargo-sdk-go/clients"
	vaultv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/vault/v1"
)

type mockVaultServer struct {
	vaultv1.UnimplementedVaultServiceServer
	mu                  sync.Mutex
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

	m.mu.Lock()
	m.chunkSizes = append(m.chunkSizes, len(req.Cascades))
	m.mu.Unlock()

	results := make([]*vaultv1.EntityCascadeResult, 0, len(req.Cascades))
	for _, c := range req.Cascades {
		tokens := make(map[string]string, len(c.Identifiers)*2)
		for idx, ident := range c.Identifiers {
			if ident == nil {
				continue
			}
			tok := fmt.Sprintf("tok-%s-%d", c.ContextId, idx)
			if ident.Urn != "" {
				tokens[ident.Urn+":"+ident.Value] = tok
			}
			if _, exists := tokens[ident.Value]; !exists {
				tokens[ident.Value] = tok
			}
		}
		results = append(results, &vaultv1.EntityCascadeResult{
			ContextId:    c.ContextId,
			Tokens:       tokens,
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
					Urn:   "urn:supercargo:hub:entity:user",
					Value: fmt.Sprintf("user-%d@example.com", i),
				},
			},
		}
	}

	ctx := context.Background()
	results, err := client.BatchTokenize(ctx, "urn:supercargo:hub:identity_domain:default", cascades)
	require.NoError(t, err)
	assert.Len(t, results, total)

	// With total=1250 and chunkSize=500, we expect 3 chunks: 500, 500, 250
	assert.Equal(t, int32(3), mockSrv.batchCalls.Load())
	assert.Equal(t, []int{500, 500, 250}, mockSrv.chunkSizes)
}

func TestVaultClient_BatchTokenize_DefaultChunkSize_250Cascades(t *testing.T) {
	mockSrv := &mockVaultServer{}
	conn, stop := startMockVaultServer(t, mockSrv)
	defer stop()

	// Initialize VaultClient without WithChunkSize to verify default chunking (100).
	client, err := clients.NewVaultClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithMaxRetries(3),
		clients.WithRetryDelay(10*time.Millisecond),
	)
	require.NoError(t, err)
	defer client.Close()

	total := 250
	cascades := make([]*vaultv1.EntityCascade, total)
	for i := 0; i < total; i++ {
		cascades[i] = &vaultv1.EntityCascade{
			ContextId: fmt.Sprintf("user-%d", i),
			Identifiers: []*vaultv1.EntityIdentifier{
				{
					Urn:   "urn:supercargo:hub:entity:user",
					Value: fmt.Sprintf("user-%d@example.com", i),
				},
			},
		}
	}

	ctx := context.Background()
	results, err := client.BatchTokenize(ctx, "urn:supercargo:hub:identity_domain:default", cascades)
	require.NoError(t, err)
	assert.Len(t, results, total)

	// With total=250 and default chunkSize=100, we expect 3 chunks: 100, 100, 50
	assert.Equal(t, int32(3), mockSrv.batchCalls.Load())
	assert.Equal(t, []int{100, 100, 50}, mockSrv.chunkSizes)
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
					Urn:   "urn:supercargo:hub:entity:user",
					Value: "user-1@example.com",
				},
			},
		},
	}

	ctx := context.Background()
	results, err := client.BatchTokenize(ctx, "urn:supercargo:hub:identity_domain:default", cascades)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tok-user-1-0", results[0].Tokens["user-1@example.com"])
	assert.Equal(t, "tok-user-1-0", results[0].Tokens["urn:supercargo:hub:entity:user:user-1@example.com"])
	assert.Equal(t, int32(2), mockSrv.batchCalls.Load())
}

func TestVaultClient_BatchTokenize_Empty(t *testing.T) {
	mockSrv := &mockVaultServer{}
	conn, stop := startMockVaultServer(t, mockSrv)
	defer stop()

	client, err := clients.NewVaultClient("bufnet", clients.WithGRPCConn(conn))
	require.NoError(t, err)
	defer client.Close()

	results, err := client.BatchTokenize(context.Background(), "urn:supercargo:hub:identity_domain:default", nil)
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
						{Urn: "urn:supercargo:hub:entity:email", Value: "a@example.com"},
					},
				},
				{
					ContextId: "ctx-2",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "b@example.com"},
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
						{Urn: "urn:supercargo:hub:entity:email", Value: "a@example.com"},
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
						{Urn: "urn:supercargo:hub:entity:email", Value: "a@example.com"},
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
						{Urn: "urn:supercargo:hub:entity:email", Value: "a@example.com"},
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
						{Urn: "urn:supercargo:hub:entity:email", Value: "a@example.com"},
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
						{Urn: "urn:supercargo:hub:entity:email", Value: "a@example.com"},
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
						{Urn: "urn:supercargo:hub:entity:email", Value: "a@example.com"},
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

			results, err := client.BatchTokenize(context.Background(), "urn:supercargo:hub:identity_domain:default", tt.cascades)
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

func startSubcutaneousVaultServer(t *testing.T, srv *mockVaultServer, expectedToken string) (*grpc.ClientConn, *atomic.Int32) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	var interceptorCalls atomic.Int32

	authInterceptor := func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		interceptorCalls.Add(1)
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
		}
		vals := md.Get("authorization")
		if len(vals) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}
		if vals[0] != "Bearer "+expectedToken {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization header: %s", vals[0])
		}
		return handler(ctx, req)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor))
	vaultv1.RegisterVaultServiceServer(s, srv)

	go func() {
		_ = s.Serve(lis)
	}()

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = conn.Close()
		s.Stop()
		_ = lis.Close()
	})

	return conn, &interceptorCalls
}

func TestVaultClient_Subcutaneous_DualAuthAndRedaction(t *testing.T) {
	t.Parallel()
	const validSecret = "super-valid-vault-token-905"
	const invalidSecret = "bad-vault-secret-905"

	t.Run("authenticated multi-identifier cascade with canonical 64-char hex HMAC tokens succeeds", func(t *testing.T) {
		t.Parallel()
		hexTok1 := strings.Repeat("a1", 32)
		hexTok2 := strings.Repeat("b2", 32)
		hexTok3 := strings.Repeat("c3", 32)
		mockSrv := &mockVaultServer{
			overrideResp: &vaultv1.BatchTokenizeResponse{
				Results: []*vaultv1.EntityCascadeResult{
					{
						ContextId: "ctx-hex-1",
						Tokens: map[string]string{
							"urn:supercargo:hub:entity:email:alice@example.com": hexTok1,
							"alice@example.com":                         hexTok1,
							"urn:supercargo:hub:entity:ssn:123-45-6789": hexTok2,
							"123-45-6789":                               hexTok2,
							"urn:supercargo:hub:entity:phone:+15550199": hexTok3,
							"+15550199":                                 hexTok3,
						},
					},
				},
			},
		}
		conn, interceptorCalls := startSubcutaneousVaultServer(t, mockSrv, validSecret)

		client, err := clients.NewVaultClient(
			"bufnet",
			clients.WithGRPCConn(conn),
			clients.WithToken(validSecret),
			clients.WithMaxRetries(3),
			clients.WithRetryDelay(5*time.Millisecond),
		)
		require.NoError(t, err)
		t.Cleanup(func() { _ = client.Close() })

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := client.BatchTokenize(ctx, "urn:supercargo:hub:identity_domain:default", []*vaultv1.EntityCascade{
			{
				ContextId: "ctx-hex-1",
				Identifiers: []*vaultv1.EntityIdentifier{
					{Urn: "urn:supercargo:hub:entity:email", Value: "alice@example.com"},
					{Urn: "urn:supercargo:hub:entity:ssn", Value: "123-45-6789"},
					{Urn: "urn:supercargo:hub:entity:phone", Value: "+15550199"},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, int32(1), interceptorCalls.Load())
		assert.Equal(t, int32(1), mockSrv.batchCalls.Load())
		assert.Equal(t, hexTok1, results[0].Tokens["urn:supercargo:hub:entity:email:alice@example.com"])
	})

	t.Run("unauthenticated request fails closed in 1 call and redacts Bearer secret", func(t *testing.T) {
		t.Parallel()
		mockSrv := &mockVaultServer{}
		conn, interceptorCalls := startSubcutaneousVaultServer(t, mockSrv, validSecret)

		client, err := clients.NewVaultClient(
			"bufnet",
			clients.WithGRPCConn(conn),
			clients.WithToken(invalidSecret),
			clients.WithMaxRetries(3),
			clients.WithRetryDelay(5*time.Millisecond),
		)
		require.NoError(t, err)
		t.Cleanup(func() { _ = client.Close() })

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.BatchTokenize(ctx, "urn:supercargo:hub:identity_domain:default", []*vaultv1.EntityCascade{
			{
				ContextId: "ctx-unauth",
				Identifiers: []*vaultv1.EntityIdentifier{
					{Urn: "urn:supercargo:hub:entity:email", Value: "alice@example.com"},
				},
			},
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok, "expected gRPC status error, got: %v", err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Equal(t, int32(1), interceptorCalls.Load(), "codes.Unauthenticated must not be retried")
		assert.Equal(t, int32(0), mockSrv.batchCalls.Load())
		assert.Contains(t, err.Error(), "Bearer [REDACTED]")
		assert.NotContains(t, err.Error(), invalidSecret)
		for unwrapped := err; unwrapped != nil; unwrapped = errors.Unwrap(unwrapped) {
			assert.NotContains(t, unwrapped.Error(), invalidSecret)
		}
	})

	t.Run("missing auth metadata fails closed with codes.Unauthenticated in 1 call", func(t *testing.T) {
		t.Parallel()
		mockSrv := &mockVaultServer{}
		conn, interceptorCalls := startSubcutaneousVaultServer(t, mockSrv, validSecret)

		client, err := clients.NewVaultClient(
			"bufnet",
			clients.WithGRPCConn(conn),
			clients.WithMaxRetries(3),
			clients.WithRetryDelay(5*time.Millisecond),
		)
		require.NoError(t, err)
		t.Cleanup(func() { _ = client.Close() })

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.BatchTokenize(ctx, "urn:supercargo:hub:identity_domain:default", []*vaultv1.EntityCascade{
			{
				ContextId: "ctx-no-auth",
				Identifiers: []*vaultv1.EntityIdentifier{
					{Urn: "urn:supercargo:hub:entity:email", Value: "alice@example.com"},
				},
			},
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Equal(t, int32(1), interceptorCalls.Load())
	})

	t.Run("tokenProvider error redacts Bearer secret and preserves context sentinel without leaking secret in %w chain", func(t *testing.T) {
		t.Parallel()
		const providerSecret = "provider-leak-secret-905"
		mockSrv := &mockVaultServer{}
		conn, interceptorCalls := startSubcutaneousVaultServer(t, mockSrv, validSecret)

		client, err := clients.NewVaultClient(
			"bufnet",
			clients.WithGRPCConn(conn),
			clients.WithTokenProvider(func(ctx context.Context) (string, error) {
				return "", fmt.Errorf("oauth refresh failed for Bearer %s: %w", providerSecret, context.DeadlineExceeded)
			}),
		)
		require.NoError(t, err)
		t.Cleanup(func() { _ = client.Close() })

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.BatchTokenize(ctx, "urn:supercargo:hub:identity_domain:default", []*vaultv1.EntityCascade{
			{
				ContextId: "ctx-provider-err",
				Identifiers: []*vaultv1.EntityIdentifier{
					{Urn: "urn:supercargo:hub:entity:email", Value: "alice@example.com"},
				},
			},
		})
		require.Error(t, err)
		assert.Equal(t, int32(0), interceptorCalls.Load())
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "expected context.DeadlineExceeded to be preserved")
		assert.Contains(t, err.Error(), "Bearer [REDACTED]")
		assert.NotContains(t, err.Error(), providerSecret)
		for unwrapped := err; unwrapped != nil; unwrapped = errors.Unwrap(unwrapped) {
			assert.NotContains(t, unwrapped.Error(), providerSecret)
		}
	})
}

func TestVaultClient_Subcutaneous_PerIdentifierFailClosed(t *testing.T) {
	t.Parallel()
	const validSecret = "valid-vault-token-905"

	tests := []struct {
		name           string
		cascades       []*vaultv1.EntityCascade
		tokens         map[string]string
		expectErr      bool
		expectedSubstr string
		forbiddenPII   []string
	}{
		{
			name: "partial token map: 2 identifiers requested and 1 omitted fails closed without retry",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-partial",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "alice@example.com"},
						{Urn: "urn:supercargo:hub:entity:ssn", Value: "123-45-6789"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:email:alice@example.com": "tok-email-1",
				"alice@example.com": "tok-email-1",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"alice@example.com", "123-45-6789"},
		},
		{
			name: "empty string token fails closed",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-empty-tok",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "secret-alice@example.com"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:email:secret-alice@example.com": "",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"secret-alice@example.com"},
		},
		{
			name: "whitespace-only token fails closed",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-ws-tok",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "secret-bob@example.com"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:email:secret-bob@example.com": "   \t",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"secret-bob@example.com"},
		},
		{
			name: "control-char NUL token fails closed",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-ctrl-nul",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "secret-carol@example.com"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:email:secret-carol@example.com": "tok\x00bad",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"secret-carol@example.com"},
		},
		{
			name: "control-char DEL token fails closed",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-ctrl-del",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "secret-dave@example.com"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:email:secret-dave@example.com": "tok\x7fbad",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"secret-dave@example.com"},
		},
		{
			name: "multi-URN same-cleartext partial omission fails closed instead of falling back to bare value",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-multi-urn",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:primary", Value: "pii-100-secret"},
						{Urn: "urn:supercargo:hub:entity:secondary", Value: "pii-100-secret"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:primary:pii-100-secret": "tok-primary",
				"pii-100-secret": "tok-primary",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"pii-100-secret"},
		},
		{
			name: "empty composite shadows bare value and fails closed",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-shadow",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:primary", Value: "pii-200-secret"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:primary:pii-200-secret": "",
				"pii-200-secret": "tok-bare",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"pii-200-secret"},
		},
		{
			name: "nil identifier in cascade fails closed without panic",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-nil-ident",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "valid@example.com"},
						nil,
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:email:valid@example.com": "tok-valid",
				"valid@example.com": "tok-valid",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"valid@example.com"},
		},
		{
			name: "blank identifier value fails closed without echoing PII",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-blank-val",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "valid@example.com"},
						{Urn: "urn:supercargo:hub:entity:ssn", Value: "   \t"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:email:valid@example.com": "tok-valid",
				"valid@example.com": "tok-valid",
			},
			expectErr:      true,
			expectedSubstr: "[MASKED]",
			forbiddenPII:   []string{"valid@example.com"},
		},
		{
			name: "valid: bare-keyed mock response with colon in another bare value does not false-positive",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-bare-colon",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:primary", Value: "100"},
						{Urn: "", Value: "foo:100"},
					},
				},
			},
			tokens: map[string]string{
				"100":     "tok_bare_1",
				"foo:100": "hmac_bare_2",
			},
			expectErr: false,
		},
		{
			name: "valid: duplicate identifiers in cascade succeed",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId: "ctx-dup",
					Identifiers: []*vaultv1.EntityIdentifier{
						{Urn: "urn:supercargo:hub:entity:email", Value: "dup@example.com"},
						{Urn: "urn:supercargo:hub:entity:email", Value: "dup@example.com"},
					},
				},
			},
			tokens: map[string]string{
				"urn:supercargo:hub:entity:email:dup@example.com": "tok_dup_1",
				"dup@example.com": "tok_dup_1",
			},
			expectErr: false,
		},
		{
			name: "valid: empty cascade.Identifiers remains valid with empty token map",
			cascades: []*vaultv1.EntityCascade{
				{
					ContextId:   "ctx-zero-idents",
					Identifiers: []*vaultv1.EntityIdentifier{},
				},
			},
			tokens:    map[string]string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockSrv := &mockVaultServer{
				overrideResp: &vaultv1.BatchTokenizeResponse{
					Results: []*vaultv1.EntityCascadeResult{
						{
							ContextId: tt.cascades[0].ContextId,
							Tokens:    tt.tokens,
						},
					},
				},
			}
			conn, interceptorCalls := startSubcutaneousVaultServer(t, mockSrv, validSecret)

			client, err := clients.NewVaultClient(
				"bufnet",
				clients.WithGRPCConn(conn),
				clients.WithToken(validSecret),
				clients.WithMaxRetries(3),
				clients.WithRetryDelay(5*time.Millisecond),
			)
			require.NoError(t, err)
			t.Cleanup(func() { _ = client.Close() })

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			results, err := client.BatchTokenize(ctx, "urn:supercargo:hub:identity_domain:default", tt.cascades)
			if tt.expectErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, clients.ErrSystemUnavailable), "expected ErrSystemUnavailable, got: %v", err)
				assert.Contains(t, err.Error(), tt.expectedSubstr)
				assert.Contains(t, err.Error(), "remediation:")
				for _, pii := range tt.forbiddenPII {
					assert.NotContains(t, err.Error(), pii, "raw PII must never appear in error messages")
				}
				assert.Nil(t, results)
				assert.Equal(t, int32(1), interceptorCalls.Load(), "fail-closed validation error must not retry")
				assert.Equal(t, int32(1), mockSrv.batchCalls.Load(), "fail-closed validation error must not retry")
			} else {
				require.NoError(t, err)
				assert.Len(t, results, len(tt.cascades))
				assert.Equal(t, int32(1), interceptorCalls.Load())
			}
		})
	}
}
