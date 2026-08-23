package clients

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// Option configures client behavior.
type Option func(*clientOptions)

type clientOptions struct {
	token       string
	timeout     time.Duration
	maxRetries  int
	retryDelay  time.Duration
	chunkSize   int
	insecure    bool
	creds       credentials.TransportCredentials
	conn        *grpc.ClientConn
	dialOptions []grpc.DialOption
}

func defaultOptions() *clientOptions {
	return &clientOptions{
		timeout:    10 * time.Second,
		maxRetries: 3,
		retryDelay: 500 * time.Millisecond,
		chunkSize:  1000,
		insecure:   false,
	}
}

// WithToken sets the Bearer authorization token.
func WithToken(token string) Option {
	return func(o *clientOptions) {
		o.token = token
	}
}

// WithTimeout sets the per-call RPC timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *clientOptions) {
		o.timeout = d
	}
}

// WithMaxRetries sets the maximum number of retry attempts for transient errors.
func WithMaxRetries(n int) Option {
	return func(o *clientOptions) {
		o.maxRetries = n
	}
}

// WithRetryDelay sets the initial backoff delay for retries.
func WithRetryDelay(d time.Duration) Option {
	return func(o *clientOptions) {
		o.retryDelay = d
	}
}

// WithChunkSize sets the maximum number of items per batch RPC (used by VaultClient).
func WithChunkSize(size int) Option {
	return func(o *clientOptions) {
		if size > 0 {
			o.chunkSize = size
		}
	}
}

// WithInsecure specifies insecure gRPC transport credentials.
func WithInsecure() Option {
	return func(o *clientOptions) {
		o.insecure = true
	}
}

// WithTransportCredentials sets custom transport credentials.
func WithTransportCredentials(creds credentials.TransportCredentials) Option {
	return func(o *clientOptions) {
		o.creds = creds
	}
}

// WithGRPCConn provides an existing grpc.ClientConn to reuse.
func WithGRPCConn(conn *grpc.ClientConn) Option {
	return func(o *clientOptions) {
		o.conn = conn
	}
}

// WithGRPCDialOptions adds extra grpc.DialOption values.
func WithGRPCDialOptions(opts ...grpc.DialOption) Option {
	return func(o *clientOptions) {
		o.dialOptions = append(o.dialOptions, opts...)
	}
}

func attachAuthMetadata(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", fmt.Sprintf("Bearer %s", token))
}
