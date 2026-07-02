// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package git

import (
	"context"
	"fmt"
	"strings"
)

// ConfigLookup reads a single git config value from the effective config (repo
// overlaying global and system), so a global setting resolves even outside a
// repository. An unset key is not an error: it yields ("", false, nil). Only a
// genuine git failure returns a non-nil error.
//
// --default "" turns an absent key into empty output with a zero exit, so the
// unset case never needs exit-code archaeology; an empty stored value reads the
// same as unset, which is what callers want here.
func ConfigLookup(ctx context.Context, r Runner, key string) (value string, found bool, err error) {
	out, err := r.Output(ctx, "git", "config", "--default", "", "--get", key)
	if err != nil {
		return "", false, fmt.Errorf("read git config %s: %w", key, err)
	}
	value = strings.TrimSpace(string(out))
	return value, value != "", nil
}
