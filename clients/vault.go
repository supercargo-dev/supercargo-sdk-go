package clients

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	vaultv1 "github.com/supercargo-dev/supercargo-sdk-go/gen/go/vault/v1"
)

var (
	_ io.Closer = (*VaultClient)(nil)

	bearerTokenPattern = regexp.MustCompile(`(?i)\bBearer\s+(?:\[REDACTED\]|"[^"]*"|'[^']*'|[^\s"',;\)\]]+)`)
)

// VaultClient provides an idiomatic Go client for the Supercargo Vault Service.
type VaultClient struct {
	stub     vaultv1.VaultServiceClient
	conn     *grpc.ClientConn
	ownsConn bool
	opts     *clientOptions
}

// WithVaultStub specifies a custom VaultServiceClient stub, primarily for testing mock behaviors.
func WithVaultStub(stub vaultv1.VaultServiceClient) Option {
	return func(o *clientOptions) {
		o.vaultStub = stub
	}
}

// NewVaultClient creates a new VaultClient for the given target address.
func NewVaultClient(target string, opts ...Option) (*VaultClient, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	if stub, ok := options.vaultStub.(vaultv1.VaultServiceClient); ok && stub != nil {
		return &VaultClient{
			stub:     stub,
			conn:     nil,
			ownsConn: false,
			opts:     options,
		}, nil
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

	authCtx, err := attachAuthMetadata(ctx, c.opts)
	if err != nil {
		return nil, sanitizeVaultError(err)
	}
	chunkSize := c.opts.chunkSize
	if chunkSize <= 0 {
		chunkSize = 100
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
					return nil, fmt.Errorf("%w: received nil response from vault service", ErrSystemUnavailable)
				}
				if len(resp.Results) != len(chunk) {
					return nil, fmt.Errorf("%w: vault response cardinality mismatch: expected %d results, received %d", ErrSystemUnavailable, len(chunk), len(resp.Results))
				}
				for i, cascade := range chunk {
					if err := validateCascadeResult(identityDomainURN, offset+i, cascade, resp.Results[i]); err != nil {
						return nil, err
					}
				}
				chunkResults = resp.Results
				chunkSucceeded = true
				break
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
						return nil, sanitizeVaultError(ctx.Err())
					case <-timer.C:
					}
					continue
				}
				break
			}
			return nil, fmt.Errorf("vault BatchTokenize failed: %w", sanitizeVaultError(err))
		}

		if !chunkSucceeded {
			return nil, fmt.Errorf("exhausted retries in vault BatchTokenize: %w", sanitizeVaultError(lastErr))
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

func validateCascadeResult(identityDomainURN string, idx int, cascade *vaultv1.EntityCascade, result *vaultv1.EntityCascadeResult) error {
	if cascade == nil {
		return fmt.Errorf("%w: nil input cascade at index %d", ErrSystemUnavailable, idx)
	}
	if result == nil {
		return fmt.Errorf("%w: nil cascade result at index %d", ErrSystemUnavailable, idx)
	}
	if result.ContextId != cascade.ContextId {
		return fmt.Errorf("%w: vault response context_id mismatch: expected '%s', got '%s'", ErrSystemUnavailable, cascade.ContextId, result.ContextId)
	}
	if len(cascade.Identifiers) == 0 {
		return nil
	}
	if len(result.Tokens) == 0 {
		return fmt.Errorf("%w: fail-closed: vault returned empty token mapping for cascade context_id '%s' (remediation: verify Vault service health and identity domain salt configuration)", ErrSystemUnavailable, cascade.ContextId)
	}

	for _, ident := range cascade.Identifiers {
		urn := "unknown"
		if ident != nil && strings.TrimSpace(ident.Urn) != "" {
			urn = ident.Urn
		}
		if ident == nil || strings.TrimSpace(ident.Value) == "" {
			return fmt.Errorf("%w: fail-closed: invalid or empty identifier '%s' [MASKED] in cascade context_id '%s' (remediation: provide non-empty entity URN and identifier value)", ErrSystemUnavailable, urn, cascade.ContextId)
		}

		tok, ok := resolveIdentifierToken(result.Tokens, ident, cascade.Identifiers)
		if !ok || !isValidToken(tok) {
			return fmt.Errorf("%w: fail-closed: vault returned missing or invalid token for cascade context_id '%s', identifier '%s' [MASKED] in domain '%s' (remediation: verify Vault service health, identity domain salt configuration, and BatchTokenize response integrity)", ErrSystemUnavailable, cascade.ContextId, urn, identityDomainURN)
		}
	}
	return nil
}

func resolveIdentifierToken(tokens map[string]string, ident *vaultv1.EntityIdentifier, allIdentifiers []*vaultv1.EntityIdentifier) (string, bool) {
	if ident.Urn != "" {
		compositeKey := ident.Urn + ":" + ident.Value
		if tok, ok := tokens[compositeKey]; ok {
			return tok, true
		}
		for _, otherIdent := range allIdentifiers {
			if otherIdent == nil || otherIdent.Urn == "" {
				continue
			}
			if _, ok := tokens[otherIdent.Urn+":"+ident.Value]; ok {
				return "", false
			}
		}
	}
	tok, ok := tokens[ident.Value]
	return tok, ok
}

func isValidToken(tok string) bool {
	if strings.TrimSpace(tok) == "" {
		return false
	}
	for i := 0; i < len(tok); i++ {
		b := tok[i]
		if b < 0x20 || b == 0x7f {
			return false
		}
	}
	return true
}

func redactBearer(msg string) string {
	return bearerTokenPattern.ReplaceAllString(msg, "Bearer [REDACTED]")
}

type sanitizedContextError struct {
	msg   string
	cause error
}

func (e *sanitizedContextError) Error() string { return e.msg }
func (e *sanitizedContextError) Unwrap() error { return e.cause }

func sanitizeVaultError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		return status.Error(st.Code(), redactBearer(st.Message()))
	}
	msg := redactBearer(err.Error())
	switch {
	case errors.Is(err, context.Canceled):
		if err == context.Canceled {
			return context.Canceled
		}
		return &sanitizedContextError{msg: msg, cause: context.Canceled}
	case errors.Is(err, context.DeadlineExceeded):
		if err == context.DeadlineExceeded {
			return context.DeadlineExceeded
		}
		return &sanitizedContextError{msg: msg, cause: context.DeadlineExceeded}
	default:
		return errors.New(msg)
	}
}
