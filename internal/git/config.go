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

// ConfigLookupBool reads key as a git boolean, canonicalized to "true" or
// "false". It understands git's full boolean vocabulary (true/false, yes/no,
// on/off, numbers) including the valueless `[section] key` form, which git
// defines as true. A value git cannot read as a boolean is an error. As with
// ConfigLookup, an unset key yields ("", false, nil).
func ConfigLookupBool(ctx context.Context, r Runner, key string) (value string, found bool, err error) {
	// The raw lookup distinguishes a stored value from nothing; the typed
	// lookup canonicalizes it. Both are needed: --type=bool alone folds the
	// unset case into "false" (via --default ""), while the raw form reads a
	// valueless key — implicit true — as empty output.
	_, found, err = ConfigLookup(ctx, r, key)
	if err != nil {
		return "", false, err
	}
	out, err := r.Output(ctx, "git", "config", "--default", "", "--type=bool", "--get", key)
	if err != nil {
		return "", false, fmt.Errorf("read git config %s as bool: %w", key, err)
	}
	value = strings.TrimSpace(string(out))
	if !found && value != "true" {
		return "", false, nil // no raw value and no valueless key: unset
	}
	return value, true, nil
}
