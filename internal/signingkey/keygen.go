// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// interactiveRunner runs ssh-keygen with inherited stdio so its PIN and touch
// prompts reach the user directly.
type interactiveRunner interface {
	RunInteractive(ctx context.Context, name string, args ...string) error
}

// KeygenOptions parameterize one resident key enrollment.
type KeygenOptions struct {
	Type   string // ed25519 (default) or ecdsa
	User   string
	Path   string // stub destination, from StubPath
	Device string // FIDO HID path (-O device=...); empty lets ssh-keygen pick
}

// ValidateUser rejects usernames that would break the stub filename or the
// FIDO application string.
func ValidateUser(user string) error {
	if user == "" {
		return fmt.Errorf("username must not be empty")
	}
	if strings.ContainsAny(user, "/\\ \t\n") {
		return fmt.Errorf("username %q must not contain spaces or path separators", user)
	}
	return nil
}

// ValidateType rejects unsupported key types (DESIGN's "edcsa" was a typo —
// it is rejected, not aliased).
func ValidateType(typ string) error {
	if !slices.Contains(KeyTypes, typ) {
		return fmt.Errorf("key type %q not supported (use %s)", typ, strings.Join(KeyTypes, " or "))
	}
	return nil
}

// NewKeyArgs builds the exact ssh-keygen invocation from DESIGN: a resident,
// verify-required (PIN), no-touch-required credential under application
// ssh:<user>. -N "" is a deliberate addition — the stub needs no passphrase
// on top of the PIN, and without it ssh-keygen would prompt for one.
func NewKeyArgs(o KeygenOptions) []string {
	args := []string{
		"-t", o.Type + "-sk",
		"-f", o.Path,
		"-O", "resident",
		"-O", "verify-required",
		"-O", "no-touch-required",
		"-O", "application=ssh:" + o.User,
		"-O", "user=" + o.User,
	}
	if o.Device != "" {
		args = append(args, "-O", "device="+o.Device)
	}
	return append(args, "-C", o.User, "-N", "")
}

// Generate enrolls the resident key, inheriting stdio for PIN/touch prompts.
func Generate(ctx context.Context, r interactiveRunner, o KeygenOptions) error {
	if err := ValidateType(o.Type); err != nil {
		return err
	}
	if err := ValidateUser(o.User); err != nil {
		return err
	}
	return r.RunInteractive(ctx, "ssh-keygen", NewKeyArgs(o)...)
}
