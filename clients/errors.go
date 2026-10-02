package clients

import "errors"

// ErrSystemUnavailable indicates that the Vault service or governance plane is unavailable,
// returned an inconsistent response, or violated fail-closed tokenization guarantees.
var ErrSystemUnavailable = errors.New("governance plane unavailable")
