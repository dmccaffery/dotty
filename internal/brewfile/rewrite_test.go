// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package brewfile

import (
	"slices"
	"testing"
)

func TestMarkTrusted(t *testing.T) {
	t.Run("marks the last occurrence only", func(t *testing.T) {
		path := seedBrewfile(t, "brew \"acme/tap/widget\"\nbrew \"jq\"\nbrew \"acme/tap/widget\"\n")
		unmarked, err := markTrusted(path, KindFormula, []string{"acme/tap/widget"})
		if err != nil {
			t.Fatalf("markTrusted() error: %v", err)
		}
		if unmarked != nil {
			t.Errorf("unmarked = %v, want none", unmarked)
		}
		want := "brew \"acme/tap/widget\"\nbrew \"jq\"\nbrew \"acme/tap/widget\", trusted: true\n"
		if got := readBrewfile(t, path); got != want {
			t.Errorf("Brewfile = %q, want %q", got, want)
		}
	})

	t.Run("lines with options or comments are never touched", func(t *testing.T) {
		content := "# brew \"acme/tap/widget\"\nbrew \"acme/tap/widget\", args: [\"HEAD\"]\n"
		path := seedBrewfile(t, content)
		unmarked, err := markTrusted(path, KindFormula, []string{"acme/tap/widget"})
		if err != nil {
			t.Fatalf("markTrusted() error: %v", err)
		}
		if !slices.Equal(unmarked, []string{"acme/tap/widget"}) {
			t.Errorf("unmarked = %v, want the name reported", unmarked)
		}
		if got := readBrewfile(t, path); got != content {
			t.Errorf("Brewfile = %q, want untouched", got)
		}
	})

	t.Run("indented entry lines still match", func(t *testing.T) {
		path := seedBrewfile(t, "  tap \"fluxcd/tap\"\n")
		if _, err := markTrusted(path, KindTap, []string{"fluxcd/tap"}); err != nil {
			t.Fatalf("markTrusted() error: %v", err)
		}
		if got, want := readBrewfile(t, path), "  tap \"fluxcd/tap\", trusted: true\n"; got != want {
			t.Errorf("Brewfile = %q, want %q", got, want)
		}
	})

	t.Run("no names touches nothing", func(t *testing.T) {
		unmarked, err := markTrusted("/does/not/exist", KindFormula, nil)
		if err != nil || unmarked != nil {
			t.Errorf("markTrusted() = (%v, %v), want (nil, nil)", unmarked, err)
		}
	})

	t.Run("missing file is an error", func(t *testing.T) {
		if _, err := markTrusted("/does/not/exist", KindFormula, []string{"a/b/c"}); err == nil {
			t.Error("markTrusted() error = nil, want read failure")
		}
	})
}
