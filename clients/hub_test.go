package clients_test

import (
	"context"
	"errors"
	"net"
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
	hubv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/hub/v1"
)

type mockHubServer struct {
	hubv1.UnimplementedHubServiceServer
	getContractCalls    atomic.Int32
	reportAnomalyCalls  atomic.Int32
	getBlastRadiusCalls atomic.Int32
	pingCalls           atomic.Int32
	failCount           int32
	failCode            codes.Code
	receivedAuthTokens  sync.Map
}

func (m *mockHubServer) recordAuth(ctx context.Context) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		vals := md.Get("authorization")
		for _, v := range vals {
			m.receivedAuthTokens.Store(v, true)
		}
	}
}

func (m *mockHubServer) GetContract(ctx context.Context, req *hubv1.GetContractRequest) (*hubv1.GetContractResponse, error) {
	m.recordAuth(ctx)
	call := m.getContractCalls.Add(1)
	if call <= m.failCount {
		code := codes.Unavailable
		if m.failCode != codes.OK {
			code = m.failCode
		}
		return nil, status.Error(code, "transient network failure")
	}
	if req.ContractUrn == "urn:sc:notfound" {
		return nil, status.Error(codes.NotFound, "contract not found")
	}
	if req.ContractUrn == "urn:sc:nilpayload" {
		return &hubv1.GetContractResponse{Contract: nil}, nil
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

func (m *mockHubServer) ReportAnomaly(ctx context.Context, req *hubv1.ReportAnomalyRequest) (*hubv1.ReportAnomalyResponse, error) {
	m.recordAuth(ctx)
	call := m.reportAnomalyCalls.Add(1)
	if call <= m.failCount {
		code := codes.Unavailable
		if m.failCode != codes.OK {
			code = m.failCode
		}
		return nil, status.Error(code, "transient error")
	}
	return &hubv1.ReportAnomalyResponse{
		TransitionId: "trans-12345",
	}, nil
}

func (m *mockHubServer) GetBlastRadius(ctx context.Context, req *hubv1.GetBlastRadiusRequest) (*hubv1.GetBlastRadiusResponse, error) {
	m.recordAuth(ctx)
	call := m.getBlastRadiusCalls.Add(1)
	if call <= m.failCount {
		code := codes.Unavailable
		if m.failCode != codes.OK {
			code = m.failCode
		}
		return nil, status.Error(code, "transient error")
	}
	return &hubv1.GetBlastRadiusResponse{
		RootUrn:         req.Urn,
		TotalDownstream: 2,
		AffectedTeams:   []string{"crm-team", "analytics-team"},
	}, nil
}

func (m *mockHubServer) Ping(ctx context.Context, req *hubv1.PingRequest) (*hubv1.PingResponse, error) {
	m.recordAuth(ctx)
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

func TestHubClient_ReportAnomaly_SuccessAndRetry(t *testing.T) {
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
	req := &hubv1.ReportAnomalyRequest{
		Urn:          "urn:sc:product:orders",
		State:        hubv1.HealthState_HEALTH_STATE_DEGRADED,
		Reason:       "High p99 latency detected",
		IncidentType: "latency",
		Reporter:     "datadog-monitor",
	}

	resp, err := client.ReportAnomaly(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "trans-12345", resp.TransitionId)
	assert.Equal(t, int32(3), mockSrv.reportAnomalyCalls.Load())
}

func TestHubClient_ReportAnomaly_NonRetryableError(t *testing.T) {
	mockSrv := &mockHubServer{failCount: 1, failCode: codes.InvalidArgument}
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
	req := &hubv1.ReportAnomalyRequest{
		Urn: "urn:sc:product:orders",
	}

	_, err = client.ReportAnomaly(ctx, req)
	require.Error(t, err)
	assert.Equal(t, int32(1), mockSrv.reportAnomalyCalls.Load(), "non-retryable error should not be retried")
}

func TestHubClient_ReportAnomaly_NilRequest(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	client, err := clients.NewHubClient("bufnet", clients.WithGRPCConn(conn))
	require.NoError(t, err)
	defer client.Close()

	_, err = client.ReportAnomaly(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report anomaly request cannot be nil")
}

func TestHubClient_GetBlastRadius_SuccessAndRetry(t *testing.T) {
	mockSrv := &mockHubServer{failCount: 1} // fail first, succeed on 2nd
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
	resp, err := client.GetBlastRadius(ctx, "urn:sc:contract:crm-orders", 3)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "urn:sc:contract:crm-orders", resp.RootUrn)
	assert.Equal(t, int32(2), resp.TotalDownstream)
	assert.Equal(t, []string{"crm-team", "analytics-team"}, resp.AffectedTeams)
	assert.Equal(t, int32(2), mockSrv.getBlastRadiusCalls.Load())
}

func TestHubClient_GetBlastRadius_NonRetryableError(t *testing.T) {
	mockSrv := &mockHubServer{failCount: 1, failCode: codes.NotFound}
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
	_, err = client.GetBlastRadius(ctx, "urn:sc:notfound", 1)
	require.Error(t, err)
	assert.Equal(t, int32(1), mockSrv.getBlastRadiusCalls.Load(), "non-retryable error should not be retried")
}

func TestHubClient_WithTokenProvider_DynamicTokenInjection(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	providerCalled := false
	tokenProvider := func(ctx context.Context) (string, error) {
		providerCalled = true
		return "dynamic-jwt-access-token", nil
	}

	client, err := clients.NewHubClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithTokenProvider(tokenProvider),
	)
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Ping(context.Background(), "auth-test")
	require.NoError(t, err)
	assert.Equal(t, "pong:auth-test", resp.Message)
	assert.True(t, providerCalled)

	_, found := mockSrv.receivedAuthTokens.Load("Bearer dynamic-jwt-access-token")
	assert.True(t, found, "server should have received dynamically injected Bearer token")
}

func TestHubClient_WithTokenProvider_ErrorFailFast(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	tokenProvider := func(ctx context.Context) (string, error) {
		return "", errors.New("vault service unreachable")
	}

	client, err := clients.NewHubClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithTokenProvider(tokenProvider),
	)
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Ping(context.Background(), "auth-failure-test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to acquire token from token provider: vault service unreachable")
	assert.Equal(t, int32(0), mockSrv.pingCalls.Load(), "RPC must fail fast without dialing server when token provider errors")
}

func TestHubClient_WithTokenProvider_OverriddenByStaticToken(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	providerCalled := false
	tokenProvider := func(ctx context.Context) (string, error) {
		providerCalled = true
		return "should-be-ignored-token", nil
	}

	client, err := clients.NewHubClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithToken("static-priority-token"),
		clients.WithTokenProvider(tokenProvider),
	)
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Ping(context.Background(), "static-precedence-test")
	require.NoError(t, err)
	assert.Equal(t, "pong:static-precedence-test", resp.Message)
	assert.False(t, providerCalled, "static token must take precedence over dynamic token provider")

	_, found := mockSrv.receivedAuthTokens.Load("Bearer static-priority-token")
	assert.True(t, found, "server should have received static token")
}

func TestHubClient_ReportAnomaly_EmptyUrn(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	client, err := clients.NewHubClient("bufnet", clients.WithGRPCConn(conn))
	require.NoError(t, err)
	defer client.Close()

	_, err = client.ReportAnomaly(context.Background(), &hubv1.ReportAnomalyRequest{Urn: "  "})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "urn cannot be empty")
}

func TestHubClient_GetBlastRadius_Validation(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	client, err := clients.NewHubClient("bufnet", clients.WithGRPCConn(conn))
	require.NoError(t, err)
	defer client.Close()

	_, err = client.GetBlastRadius(context.Background(), "", 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "urn cannot be empty")

	_, err = client.GetBlastRadius(context.Background(), "urn:sc:test", -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maxDepth cannot be negative")
}

func TestHubClient_GetContract_NilContractPayload(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	client, err := clients.NewHubClient("bufnet", clients.WithGRPCConn(conn))
	require.NoError(t, err)
	defer client.Close()

	_, err = client.GetContract(context.Background(), "urn:sc:nilpayload", "v1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "received nil response from hub service for contract")
}

func TestHubClient_WithTokenProvider_EmptyTokenFailClosed(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	tokenProvider := func(ctx context.Context) (string, error) {
		return "", nil // empty token without error
	}

	client, err := clients.NewHubClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithTokenProvider(tokenProvider),
	)
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Ping(context.Background(), "fail-closed-test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token provider returned empty token")
	assert.Equal(t, int32(0), mockSrv.pingCalls.Load(), "must fail closed before network call")
}

func TestHubClient_WithTokenProvider_BearerPrefixTrimming(t *testing.T) {
	mockSrv := &mockHubServer{}
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	tokenProvider := func(ctx context.Context) (string, error) {
		return "Bearer prefixed-token", nil
	}

	client, err := clients.NewHubClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithTokenProvider(tokenProvider),
	)
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Ping(context.Background(), "prefix-trim-test")
	require.NoError(t, err)

	_, found := mockSrv.receivedAuthTokens.Load("Bearer prefixed-token")
	assert.True(t, found, "server should have received normalized token without double Bearer")
}

func TestHubClient_ReportAnomaly_ExhaustedRetries(t *testing.T) {
	mockSrv := &mockHubServer{failCount: 5} // always fail
	conn, stop := startMockHubServer(t, mockSrv)
	defer stop()

	client, err := clients.NewHubClient(
		"bufnet",
		clients.WithGRPCConn(conn),
		clients.WithMaxRetries(3),
		clients.WithRetryDelay(1*time.Millisecond),
	)
	require.NoError(t, err)
	defer client.Close()

	_, err = client.ReportAnomaly(context.Background(), &hubv1.ReportAnomalyRequest{Urn: "urn:sc:test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exhausted retries reporting anomaly on \"urn:sc:test\"")
	assert.Equal(t, int32(3), mockSrv.reportAnomalyCalls.Load())
}
