// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"strings"
	"testing"
)

// TestImportRequiresInteractive pins the early guard: import drives ssh-keygen
// -K, which needs a terminal for the FIDO PIN and touch, so a non-interactive
// invocation (as in tests and scripts) must fail fast before touching disk or
// the hardware. The test process has no TTY, so this exercises that branch.
func TestImportRequiresInteractive(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := execDotty(t, "signing-key", "import", t.TempDir())
	if err == nil {
		t.Fatal("import without a terminal succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "interactive") {
		t.Errorf("error = %q, want mention of an interactive terminal", err)
	}
}
