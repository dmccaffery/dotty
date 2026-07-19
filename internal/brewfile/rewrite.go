// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package brewfile

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitwise-media-group/dotty/internal/cli"
)

// markTrusted appends `, trusted: true` to the bare entry lines `brew bundle
// add` just wrote for names, so the grant survives `brew bundle install
// --force-cleanup`, which resets Homebrew's trust store to exactly what the
// Brewfile declares. Lines are matched exactly (`<word> "<name>"`, trimmed)
// scanning from the end of the file, so descriptions, comments, and entries
// that already carry options are never touched. Names whose line cannot be
// found are returned for the caller to warn about rather than failing — a
// drift in brew's output format must not fail an add that already succeeded.
func markTrusted(path string, kind Kind, names []string) (unmarked []string, err error) {
	if len(names) == 0 {
		return nil, nil
	}
	want := make(map[string]bool, len(names))
	for _, name := range names {
		want[dslWord(kind)+` "`+name+`"`] = true
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Brewfile %s: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	// brew appends at the end; scanning in reverse keeps an identical
	// hand-authored line earlier in the file untouched.
	for i := len(lines) - 1; i >= 0 && len(want) > 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if !want[trimmed] {
			continue
		}
		lines[i] += ", trusted: true"
		delete(want, trimmed)
		changed = true
	}
	for _, name := range names { // report leftovers in input order
		if want[dslWord(kind)+` "`+name+`"`] {
			unmarked = append(unmarked, name)
		}
	}
	if !changed {
		return unmarked, nil
	}

	perm := os.FileMode(0o644)
	if fi, statErr := os.Stat(path); statErr == nil {
		perm = fi.Mode().Perm()
	}
	if err := cli.AtomicWriteFile(path, []byte(strings.Join(lines, "\n")), perm); err != nil {
		return unmarked, fmt.Errorf("write Brewfile %s: %w", path, err)
	}
	return unmarked, nil
}
