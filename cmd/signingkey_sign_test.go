// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestSignHelpInterception pins DESIGN's proxy rule: --help is always handled
// by dotty, never forwarded to ssh-keygen — even though flag parsing is off.
func TestSignHelpInterception(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			out := &bytes.Buffer{}
			rootCmd.SetOut(out)
			rootCmd.SetErr(out)
			defer func() {
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
			}()

			if err := execDotty(t, "signing-key", "sign", flag); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if !strings.Contains(out.String(), "Proxy to ssh-keygen") {
				t.Errorf("help output missing command Long text:\n%s", out.String())
			}
		})
	}
}

// TestSignHelpAfterDoubleDash pins the inverse: -h after -- belongs to the
// proxied program, so dotty must NOT print help. It will instead try to sign
// and fail (no key in the scratch dirs) — that failure is the assertion.
func TestSignHelpAfterDoubleDash(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := execDotty(t, "signing-key", "sign", "--", "-h")
	if err == nil {
		t.Fatal("expected an error (no key resolvable), got help-style success")
	}
}
