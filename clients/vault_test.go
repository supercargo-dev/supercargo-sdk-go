package clients_test

import (
	"context"
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

	vaultv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/vault/v1"
	"github.com/supercargo-dev/supercargo-sdk-go/clients"
)

type mockVaultServer struct {
	vaultv1.UnimplementedVaultServiceServer
	batchCalls atomic.Int32
	failCount  int32
	chunkSizes []int
}

func (m *mockVaultServer) BatchTokenize(ctx context.Context, req *vaultv1.BatchTokenizeRequest) (*vaultv1.BatchTokenizeResponse, error) {
	call := m.batchCalls.Add(1)
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
