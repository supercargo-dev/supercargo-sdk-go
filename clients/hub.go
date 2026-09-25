package clients

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	hubv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/hub/v1"
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

// computeBackoff calculates jittered exponential backoff: delay * (0.75 + 0.5 * rand.Float64()) capped at 5 seconds.
func computeBackoff(baseDelay time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	shift := attempt - 1
	if shift > 6 {
		shift = 6
	}
	delay := baseDelay * (1 << shift)
	jitter := 0.75 + 0.5*rand.Float64()
	jittered := time.Duration(float64(delay) * jitter)
	if jittered > 5*time.Second || jittered < 0 {
		return 5 * time.Second
	}
	return jittered
}

// GetContract retrieves a DataContract by URN and optional version with automatic retry logic.
func (c *HubClient) GetContract(ctx context.Context, urn string, version string) (*hubv1.DataContract, error) {
	if strings.TrimSpace(urn) == "" {
		return nil, fmt.Errorf("contract urn cannot be empty")
	}

	authCtx, err := attachAuthMetadata(ctx, c.opts)
	if err != nil {
		return nil, err
	}

	req := &hubv1.GetContractRequest{
		ContractUrn: urn,
		Version:     version,
	}

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
			if resp == nil || resp.Contract == nil {
				return nil, fmt.Errorf("received nil response from hub service for contract %q", urn)
			}
			return resp.Contract, nil
		}

		st, ok := status.FromError(err)
		if ok && isRetryableStatusCode(st.Code()) {
			lastErr = err
			if attempt < c.opts.maxRetries {
				delay := computeBackoff(c.opts.retryDelay, attempt)
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}
				continue
			}
			break
		}
		return nil, fmt.Errorf("failed to fetch contract %q: %w", urn, err)
	}

	return nil, fmt.Errorf("exhausted retries fetching contract %q: %w", urn, lastErr)
}

// ReportAnomaly reports a health degradation or recovery for a product or contract with automatic retry logic.
func (c *HubClient) ReportAnomaly(ctx context.Context, req *hubv1.ReportAnomalyRequest) (*hubv1.ReportAnomalyResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("report anomaly request cannot be nil")
	}
	if strings.TrimSpace(req.Urn) == "" {
		return nil, fmt.Errorf("urn cannot be empty")
	}

	authCtx, err := attachAuthMetadata(ctx, c.opts)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 1; attempt <= c.opts.maxRetries; attempt++ {
		callCtx := authCtx
		var cancel context.CancelFunc
		if c.opts.timeout > 0 {
			callCtx, cancel = context.WithTimeout(authCtx, c.opts.timeout)
		}

		resp, err := c.stub.ReportAnomaly(callCtx, req)
		if cancel != nil {
			cancel()
		}

		if err == nil {
			if resp == nil {
				return nil, fmt.Errorf("received nil response from hub service for anomaly report on %q", req.Urn)
			}
			return resp, nil
		}

		st, ok := status.FromError(err)
		if ok && isRetryableStatusCode(st.Code()) {
			lastErr = err
			if attempt < c.opts.maxRetries {
				delay := computeBackoff(c.opts.retryDelay, attempt)
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}
				continue
			}
			break
		}
		return nil, fmt.Errorf("failed to report anomaly on %q: %w", req.Urn, err)
	}

	return nil, fmt.Errorf("exhausted retries reporting anomaly on %q: %w", req.Urn, lastErr)
}

// GetBlastRadius analyzes downstream blast radius for a given URN and max depth with automatic retry logic.
func (c *HubClient) GetBlastRadius(ctx context.Context, urn string, maxDepth int32) (*hubv1.GetBlastRadiusResponse, error) {
	if strings.TrimSpace(urn) == "" {
		return nil, fmt.Errorf("urn cannot be empty")
	}
	if maxDepth < 0 {
		return nil, fmt.Errorf("maxDepth cannot be negative")
	}
	return c.GetBlastRadiusRequest(ctx, &hubv1.GetBlastRadiusRequest{
		Urn:      urn,
		MaxDepth: maxDepth,
	})
}

// GetBlastRadiusRequest analyzes downstream blast radius using a proto request with automatic retry logic.
func (c *HubClient) GetBlastRadiusRequest(ctx context.Context, req *hubv1.GetBlastRadiusRequest) (*hubv1.GetBlastRadiusResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get blast radius request cannot be nil")
	}
	if strings.TrimSpace(req.Urn) == "" {
		return nil, fmt.Errorf("urn cannot be empty")
	}
	if req.MaxDepth < 0 {
		return nil, fmt.Errorf("maxDepth cannot be negative")
	}

	authCtx, err := attachAuthMetadata(ctx, c.opts)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 1; attempt <= c.opts.maxRetries; attempt++ {
		callCtx := authCtx
		var cancel context.CancelFunc
		if c.opts.timeout > 0 {
			callCtx, cancel = context.WithTimeout(authCtx, c.opts.timeout)
		}

		resp, err := c.stub.GetBlastRadius(callCtx, req)
		if cancel != nil {
			cancel()
		}

		if err == nil {
			if resp == nil {
				return nil, fmt.Errorf("received nil response from hub service for blast radius on %q", req.Urn)
			}
			return resp, nil
		}

		st, ok := status.FromError(err)
		if ok && isRetryableStatusCode(st.Code()) {
			lastErr = err
			if attempt < c.opts.maxRetries {
				delay := computeBackoff(c.opts.retryDelay, attempt)
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}
				continue
			}
			break
		}
		return nil, fmt.Errorf("failed to get blast radius for %q: %w", req.Urn, err)
	}

	return nil, fmt.Errorf("exhausted retries getting blast radius for %q: %w", req.Urn, lastErr)
}

// Ping sends a ping message to verify Hub service health.
func (c *HubClient) Ping(ctx context.Context, message string) (*hubv1.PingResponse, error) {
	authCtx, err := attachAuthMetadata(ctx, c.opts)
	if err != nil {
		return nil, err
	}

	req := &hubv1.PingRequest{
		Message: message,
	}

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
