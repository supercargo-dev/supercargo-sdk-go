package clients

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	vaultv1 "github.com/supercargo-dev/core/gen/go/vault/v1"
)

var _ io.Closer = (*VaultClient)(nil)

// VaultClient provides an idiomatic Go client for the Supercargo Vault Service.
type VaultClient struct {
	stub     vaultv1.VaultServiceClient
	conn     *grpc.ClientConn
	ownsConn bool
	opts     *clientOptions
}

// NewVaultClient creates a new VaultClient for the given target address.
func NewVaultClient(target string, opts ...Option) (*VaultClient, error) {
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
			return nil, fmt.Errorf("failed to dial vault service at %s: %w", target, err)
		}
		conn = c
		ownsConn = true
	}

	return &VaultClient{
		stub:     vaultv1.NewVaultServiceClient(conn),
		conn:     conn,
		ownsConn: ownsConn,
		opts:     options,
	}, nil
}

// BatchTokenize executes batch pseudonymization across entity cascades with automatic sub-batch chunking.
func (c *VaultClient) BatchTokenize(ctx context.Context, identityDomainURN string, cascades []*vaultv1.EntityCascade) ([]*vaultv1.EntityCascadeResult, error) {
	if len(cascades) == 0 {
		return []*vaultv1.EntityCascadeResult{}, nil
	}

	authCtx := attachAuthMetadata(ctx, c.opts.token)
	chunkSize := c.opts.chunkSize
	if chunkSize <= 0 {
		chunkSize = 1000
	}

	allResults := make([]*vaultv1.EntityCascadeResult, 0, len(cascades))

	for offset := 0; offset < len(cascades); offset += chunkSize {
		end := offset + chunkSize
		if end > len(cascades) {
			end = len(cascades)
		}
		chunk := cascades[offset:end]

		req := &vaultv1.BatchTokenizeRequest{
			IdentityDomainUrn: identityDomainURN,
			Cascades:          chunk,
		}

		var chunkResults []*vaultv1.EntityCascadeResult
		var lastErr error
		chunkSucceeded := false

		for attempt := 1; attempt <= c.opts.maxRetries; attempt++ {
			callCtx := authCtx
			var cancel context.CancelFunc
			if c.opts.timeout > 0 {
				callCtx, cancel = context.WithTimeout(authCtx, c.opts.timeout)
			}

			resp, err := c.stub.BatchTokenize(callCtx, req)
			if cancel != nil {
				cancel()
			}

			if err == nil {
				if resp == nil {
					return nil, fmt.Errorf("received nil response from vault service")
				}
				chunkResults = resp.Results
				chunkSucceeded = true
				break
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
			return nil, fmt.Errorf("vault BatchTokenize failed: %w", err)
		}

		if !chunkSucceeded {
			return nil, fmt.Errorf("exhausted retries in vault BatchTokenize: %w", lastErr)
		}

		allResults = append(allResults, chunkResults...)
	}

	return allResults, nil
}

// Close closes the underlying gRPC connection if owned by this client.
func (c *VaultClient) Close() error {
	if c.ownsConn && c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
