package clients_test

import (
	"context"
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
	hubv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/hub/v1"
)

type mockHubServer struct {
	hubv1.UnimplementedHubServiceServer
	getContractCalls atomic.Int32
	failCount        int32
	pingCalls        atomic.Int32
}

func (m *mockHubServer) GetContract(ctx context.Context, req *hubv1.GetContractRequest) (*hubv1.GetContractResponse, error) {
	call := m.getContractCalls.Add(1)
	if call <= m.failCount {
		return nil, status.Error(codes.Unavailable, "transient network failure")
	}
	if req.ContractUrn == "urn:sc:notfound" {
		return nil, status.Error(codes.NotFound, "contract not found")
	}
	return &hubv1.GetContractResponse{
		Contract: &hubv1.DataContract{
			Meta: &hubv1.Meta{
				Urn:     req.ContractUrn,
				Version: req.Version,
			},
		},
	}, nil
}

func (m *mockHubServer) Ping(ctx context.Context, req *hubv1.PingRequest) (*hubv1.PingResponse, error) {
	m.pingCalls.Add(1)
	return &hubv1.PingResponse{
		Message: "pong:" + req.Message,
	}, nil
}

func startMockHubServer(t *testing.T, srv *mockHubServer) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	hubv1.RegisterHubServiceServer(s, srv)

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

func TestHubClient_GetContract_RetrySuccess(t *testing.T) {
	mockSrv := &mockHubServer{failCount: 2} // fail first 2, succeed on 3rd
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	client, err := clients.NewHubClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithMaxRetries(3),
		clients.WithRetryDelay(10*time.Millisecond),
		clients.WithToken("test-bearer-token"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	contract, err := client.GetContract(ctx, "urn:sc:contract:test", "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, contract)
	assert.Equal(t, "urn:sc:contract:test", contract.Meta.Urn)
	assert.Equal(t, int32(3), mockSrv.getContractCalls.Load())
}

func TestHubClient_GetContract_NonRetryableError(t *testing.T) {
	mockSrv := &mockHubServer{failCount: 0}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	client, err := clients.NewHubClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithMaxRetries(3),
		clients.WithRetryDelay(10*time.Millisecond),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	_, err = client.GetContract(ctx, "urn:sc:notfound", "v1.0.0")
	require.Error(t, err)
	assert.Equal(t, int32(1), mockSrv.getContractCalls.Load(), "non-retryable error should not be retried")
}

func TestHubClient_Ping(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	client, err := clients.NewHubClient("bufnet", clients.WithGRPCConn(conn))
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Ping(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "pong:hello", resp.Message)
	assert.Equal(t, int32(1), mockSrv.pingCalls.Load())
}
