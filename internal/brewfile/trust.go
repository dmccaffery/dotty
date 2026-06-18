// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package brewfile

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// NeedsTrust reports whether a brew name is tap-qualified deeply enough to be
// subject to Homebrew's tap-trust gate: more than one forward slash, e.g.
// "anomalyco/tap/opencode".
func NeedsTrust(name string) bool {
	return strings.Count(name, "/") > 1
}

// trustList mirrors the `brew trust --json v1` document: flat name lists per
// kind.
type trustList struct {
	Taps     []string `json:"taps"`
	Formulae []string `json:"formulae"`
	Casks    []string `json:"casks"`
	Commands []string `json:"commands"`
}

// decodeTrustList parses `brew trust --json v1` output.
func decodeTrustList(data []byte) (trustList, error) {
	var t trustList
	if err := json.Unmarshal(data, &t); err != nil {
		return trustList{}, fmt.Errorf("decode brew trust JSON: %w", err)
	}
	return t, nil
}

// IsTrusted reports whether name is already in Homebrew's trust store for the
// given kind.
func IsTrusted(ctx context.Context, r Runner, kind Kind, name string) (bool, error) {
	out, err := r.Output(ctx, "brew", "trust", "--json", "v1")
	if err != nil {
		return false, fmt.Errorf("read brew trust store: %w", err)
	}
	t, err := decodeTrustList(out)
	if err != nil {
		return false, err
	}
	switch kind {
	case KindFormula:
		return slices.Contains(t.Formulae, name), nil
	case KindCask:
		return slices.Contains(t.Casks, name), nil
	case KindTap:
		return slices.Contains(t.Taps, name), nil
	default:
		return false, fmt.Errorf("kind %s is not trustable", kind)
	}
}

// Trust records name in Homebrew's trust store for the given kind.
func Trust(ctx context.Context, r Runner, kind Kind, name string) error {
	if !kind.Trustable() {
		return fmt.Errorf("kind %s is not trustable", kind)
	}
	return r.Run(ctx, "brew", "trust", kind.flag(), name)
}
