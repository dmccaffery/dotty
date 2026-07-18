// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package worktree

import "strings"

// Sanitize encodes a string into a tmux-safe session/worktree name. tmux
// >=3.5 rejects '.' in session names, so dots are encoded (leading ->
// "dot-", trailing -> "-dot", interior -> "-dot-") before every other
// disallowed character collapses to '-'.
func Sanitize(s string) string {
	if strings.HasPrefix(s, ".") {
		s = "dot-" + s[1:]
	}
	if strings.HasSuffix(s, ".") {
		s = s[:len(s)-1] + "-dot"
	}
	s = strings.ReplaceAll(s, ".", "-dot-")

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isAllowed(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

func isAllowed(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z',
		r >= 'A' && r <= 'Z',
		r >= '0' && r <= '9',
		r == '_', r == '-':
		return true
	default:
		return false
	}
}
