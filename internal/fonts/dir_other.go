// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

//go:build !darwin

package fonts

import (
	"errors"
	"fmt"
	"runtime"
)

// Dir returns the per-user font directory fonts install into. Only darwin is
// wired up so far; Linux (~/.local/share/fonts + fc-cache) is a future goal.
func Dir(home string) (string, error) {
	return "", fmt.Errorf("font install on %s: %w", runtime.GOOS, errors.ErrUnsupported)
}
