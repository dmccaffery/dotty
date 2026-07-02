// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitwise-media-group/dotty/internal/cli"
)

// Trusted reports the outcome of Trust: the refs whose entries were appended,
// and those left alone because an identical (principal, key identity) line was
// already on file.
type Trusted struct {
	Added   []KeyRef
	Skipped []KeyRef
}

// Trust appends an OpenSSH allowed_signers entry for each ref's public key to
// the file at path, so git can verify commits and tags those keys sign. Each
// line pairs principal — the committer email — with the key identity (algorithm
// and blob, per PubKeyID); allowed_signers has no comment field, so the stub's
// comment is dropped, and the absence of a namespaces= option lets the entry
// verify signatures in any namespace, which is what git needs.
//
// The parent directory is created 0700 and the file written 0600 when missing;
// existing content, including comments and hand-written entries, is preserved
// verbatim. An entry whose (principal, key identity) already appears is skipped,
// so repeated runs converge. A ref with an unreadable or malformed public key
// aborts the whole write, leaving path untouched.
func Trust(path, principal string, refs []KeyRef) (Trusted, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Trusted{}, fmt.Errorf("read %s: %w", path, err)
	}

	present := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		if p, id, ok := parseAllowedSigner(line); ok {
			present[p+"\x00"+id] = true
		}
	}

	var result Trusted
	var add strings.Builder
	for _, ref := range refs {
		pub, err := os.ReadFile(ref.PubPath)
		if err != nil {
			return Trusted{}, fmt.Errorf("read public key %s: %w", ref.PubPath, err)
		}
		id := PubKeyID(string(pub))
		if id == "" {
			return Trusted{}, fmt.Errorf("public key %s is malformed", ref.PubPath)
		}
		key := principal + "\x00" + id
		if present[key] {
			result.Skipped = append(result.Skipped, ref)
			continue
		}
		present[key] = true
		add.WriteString(principal + " " + id + "\n")
		result.Added = append(result.Added, ref)
	}

	if add.Len() == 0 {
		return result, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return Trusted{}, fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	// Keep existing bytes verbatim, guaranteeing a newline before the appended
	// entries so a file that lacked a trailing newline stays well-formed.
	content := data
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		content = append(content, '\n')
	}
	content = append(content, add.String()...)
	if err := cli.AtomicWriteFile(path, content, 0o600); err != nil {
		return Trusted{}, err
	}
	return result, nil
}

// parseAllowedSigner extracts an allowed_signers line's principal and key
// identity (algorithm and blob, matching PubKeyID) for duplicate detection. It
// skips blanks and comments and tolerates an options field between the
// principals and the key. ok is false for any line it can't read as an entry.
func parseAllowedSigner(line string) (principal, id string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	// principals [options] keytype base64 [comment]: scan past an optional
	// options field to the algorithm, whose blob is the field after it.
	fields := strings.Fields(line)
	for i := 1; i+1 < len(fields); i++ {
		if isPubKeyAlgo(fields[i]) {
			return fields[0], fields[i] + " " + fields[i+1], true
		}
	}
	return "", "", false
}

// isPubKeyAlgo reports whether field names an SSH public-key algorithm — the
// marker that, in an allowed_signers line, the base64 key blob is the next
// field. It covers the sk- FIDO variants dotty enrols and the plain types a
// hand-written entry might carry.
func isPubKeyAlgo(field string) bool {
	switch {
	case strings.HasPrefix(field, "sk-ssh-"), strings.HasPrefix(field, "sk-ecdsa-"),
		strings.HasPrefix(field, "ssh-"), strings.HasPrefix(field, "ecdsa-"):
		return true
	default:
		return false
	}
}
