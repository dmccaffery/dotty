// MIT License
//
// Copyright (c) 2026 Bitwise Media Group
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
