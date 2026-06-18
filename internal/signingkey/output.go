// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import "strings"

// FormatGitKey renders a public key as the literal-key line git's
// gpg.ssh.defaultKeyCommand expects. sk public keys do not start with "ssh-",
// so the key:: prefix is mandatory for git to recognize them.
func FormatGitKey(pub []byte) string {
	return "key::" + strings.TrimSpace(firstLine(string(pub)))
}
