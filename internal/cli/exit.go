// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package cli

import "fmt"

// ExitError carries a child process's exit code to main, which must terminate
// with the same code. Proxy commands (signing-key sign) depend on this — git
// inspects the signer's exit status.
type ExitError struct {
	Code int
	Err  error
}

// Error returns the wrapped error's message, or a generic exit-status line.
func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit status %d", e.Code)
}

// Unwrap exposes the underlying error for errors.Is/As chains.
func (e *ExitError) Unwrap() error { return e.Err }
