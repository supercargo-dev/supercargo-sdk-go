package clients

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	hubv1 "github.com/supercargo-dev/core/gen/go/hub/v1"
)

var _ io.Closer = (*HubClient)(nil)

// HubClient provides an idiomatic Go client for the Supercargo Hub Service.
type HubClient struct {
	stub     hubv1.HubServiceClient
	conn     *grpc.ClientConn
	ownsConn bool
	opts     *clientOptions
}

// NewHubClient creates a new HubClient for the given target address.
func NewHubClient(target string, opts ...Option) (*HubClient, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	var conn *grpc.ClientConn
	var ownsConn bool

	if options.conn != nil {
		conn = options.conn
		ownsConn = false
	} else {
		dialOpts := options.dialOptions
		if options.insecure {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		} else if options.creds != nil {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(options.creds))
		} else {
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
		}

		c, err := grpc.NewClient(target, dialOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to dial hub service at %s: %w", target, err)
		}
		conn = c
		ownsConn = true
	}

	return &HubClient{
		stub:     hubv1.NewHubServiceClient(conn),
		conn:     conn,
		ownsConn: ownsConn,
		opts:     options,
	}, nil
}

// isRetryableStatusCode checks if a gRPC status code is transient and retryable.
func isRetryableStatusCode(code codes.Code) bool {
	return code == codes.Unavailable || code == codes.ResourceExhausted || code == codes.DeadlineExceeded
}

// GetContract retrieves a DataContract by URN and optional version with automatic retry logic.
func (c *HubClient) GetContract(ctx context.Context, urn string, version string) (*hubv1.DataContract, error) {
	req := &hubv1.GetContractRequest{
		ContractUrn: urn,
		Version:     version,
	}

	authCtx := attachAuthMetadata(ctx, c.opts.token)

	var lastErr error
	for attempt := 1; attempt <= c.opts.maxRetries; attempt++ {
		callCtx := authCtx
		var cancel context.CancelFunc
		if c.opts.timeout > 0 {
			callCtx, cancel = context.WithTimeout(authCtx, c.opts.timeout)
		}

		resp, err := c.stub.GetContract(callCtx, req)
		if cancel != nil {
			cancel()
		}

		if err == nil {
			if resp == nil {
				return nil, fmt.Errorf("received nil response from hub service for contract %q", urn)
			}
			return resp.Contract, nil
		}

		st, ok := status.FromError(err)
		if ok && isRetryableStatusCode(st.Code()) && attempt < c.opts.maxRetries {
			delay := c.opts.retryDelay * (1 << (attempt - 1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			lastErr = err
			continue
		}
		return nil, fmt.Errorf("failed to fetch contract %q: %w", urn, err)
	}

	return nil, fmt.Errorf("exhausted retries fetching contract %q: %w", urn, lastErr)
}

// Ping sends a ping message to verify Hub service health.
func (c *HubClient) Ping(ctx context.Context, message string) (*hubv1.PingResponse, error) {
	req := &hubv1.PingRequest{
		Message: message,
	}

	authCtx := attachAuthMetadata(ctx, c.opts.token)
	if c.opts.timeout > 0 {
		var cancel context.CancelFunc
		authCtx, cancel = context.WithTimeout(authCtx, c.opts.timeout)
		defer cancel()
	}

	resp, err := c.stub.Ping(authCtx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("received nil ping response from hub service")
	}
	return resp, nil
}

// Close closes the underlying gRPC connection if owned by this client.
func (c *HubClient) Close() error {
	if c.ownsConn && c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
