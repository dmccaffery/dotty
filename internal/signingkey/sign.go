// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"context"
	"fmt"
	"strings"
)

// sk public key algorithm prefixes; a file starting with one of these is a
// literal public key (what git's gpg.ssh.defaultKeyCommand flow writes to a
// temp file), not a private key stub.
var skPubPrefixes = []string{
	"sk-ssh-ed25519@openssh.com ",
	"sk-ecdsa-sha2-nistp256@openssh.com ",
}

// RewriteSignArgs adapts an ssh-keygen sign argv to hardware-backed signing.
// args is the verbatim passthrough (dotty-owned flags already extracted);
// resolveDefault lazily resolves a stub path for invocations with no -f;
// scan lists the stub inventory for literal-pubkey matching; readFile keeps
// the file probes injectable for tests.
//
// Three cases:
//  1. -f names a literal sk public key: git resolved the key via
//     defaultKeyCommand and added -U (agent signing). The -f value is
//     replaced with the matching stub and -U dropped — the hardware signs,
//     not an agent. No matching stub is an error.
//  2. -f names anything else (a stub via user.signingKey, or unreadable):
//     pure passthrough — ssh-keygen gives the authoritative error.
//  3. No -f (a human ran `dotty signing-key sign file`): -Y sign and a
//     default -n namespace are prepended when missing, and the resolved
//     stub is injected.
func RewriteSignArgs(
	args []string,
	resolveDefault func() (string, error),
	scan func() ([]KeyRef, error),
	readFile func(string) ([]byte, error),
) ([]string, error) {
	fileIdx := -1
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-f" {
			fileIdx = i + 1
			break
		}
	}

	if fileIdx >= 0 {
		content, err := readFile(args[fileIdx])
		if err != nil {
			return args, nil // let ssh-keygen report the unreadable file
		}
		line := firstLine(string(content))
		if !isSKPublicKey(line) {
			return args, nil
		}
		refs, err := scan()
		if err != nil {
			return nil, err
		}
		ref, ok := MatchByPublicKey(refs, line)
		if !ok {
			return nil, fmt.Errorf(
				"public key in %s does not correspond to any dotty-managed signing key (run `dotty signing-key list`)",
				args[fileIdx])
		}
		out := make([]string, 0, len(args))
		for i := 0; i < len(args); i++ {
			switch {
			case i == fileIdx:
				out = append(out, ref.PrivPath)
			case args[i] == "-U": // hardware signs, not an agent
			default:
				out = append(out, args[i])
			}
		}
		return out, nil
	}

	stub, err := resolveDefault()
	if err != nil {
		return nil, err
	}
	var prefix []string
	if !hasFlag(args, "-Y") {
		prefix = append(prefix, "-Y", "sign")
		if !hasFlag(args, "-n") {
			prefix = append(prefix, "-n", "file")
		}
	}
	prefix = append(prefix, "-f", stub)
	return append(prefix, args...), nil
}

// Sign execs ssh-keygen with the rewritten argv and inherited stdio. A
// non-zero exit surfaces as *cli.ExitError so main can mirror the code —
// git checks it.
func Sign(ctx context.Context, r interactiveRunner, args []string) error {
	return r.RunInteractive(ctx, "ssh-keygen", args...)
}

func isSKPublicKey(line string) bool {
	for _, prefix := range skPubPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
