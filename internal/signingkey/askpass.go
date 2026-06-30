// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"context"
	"strings"
)

// PinentryPath is the pinentry-mac binary the ask-pass bridge drives. The
// bridge runs in the minimal environment ssh hands $SSH_ASKPASS, so the path is
// absolute rather than resolved on PATH; it is a var so tests can point it
// elsewhere.
var PinentryPath = "/opt/homebrew/bin/pinentry-mac"

// assuanRunner feeds an Assuan command block to pinentry's stdin and returns
// its stdout. A non-zero exit is not an error: pinentry exits non-zero when the
// user cancels, which surfaces here as an empty PIN.
type assuanRunner interface {
	RunAssuan(ctx context.Context, stdin, name string, args ...string) (string, error)
}

// AskPassReply computes the reply for an OpenSSH SSH_ASKPASS prompt. FIDO2
// user-presence prompts only need a non-error reply, so the reply is empty and
// pinentry is never run; PIN prompts are forwarded to pinentry-mac and the
// entered PIN is returned. keyinfo is the signing key's SHA256 fingerprint
// (from $DOTTY_SSH_KEYINFO), passed through to pinentry so it can name — and
// cache in the macOS keychain — the specific key being unlocked.
func AskPassReply(ctx context.Context, r assuanRunner, prompt, keyinfo string) string {
	// FIDO2 user-presence: ssh only needs a non-error reply, not a PIN.
	if strings.HasPrefix(prompt, "Confirm user presence") {
		return ""
	}
	// Ignore pinentry's exit status: a cancel yields no data line and an empty
	// PIN, which ssh treats as a failed unlock.
	out, _ := r.RunAssuan(ctx, buildAssuan(prompt, keyinfo), PinentryPath)
	return extractPin(out)
}

// buildAssuan renders the pinentry Assuan command block for a PIN prompt. When
// the signing key's fingerprint is known, SETKEYINFO together with
// allow-external-password-cache is what lets pinentry-mac store and reuse the
// PIN in the macOS keychain. `signing-key sign` supplies the fingerprint via
// $DOTTY_SSH_KEYINFO because ssh-keygen's own agentless PIN prompt names no key;
// agent-style prompts embed it in their text instead, recovered as a fallback.
func buildAssuan(prompt, keyinfo string) string {
	if keyinfo == "" {
		keyinfo = promptFingerprint(prompt)
	}
	if keyinfo != "" {
		return "SETDESC " + prompt + "\n" +
			"OPTION allow-external-password-cache\n" +
			"SETKEYINFO s/" + keyinfo + "\n" +
			"GETPIN\n"
	}
	return "SETDESC " + prompt + "\nGETPIN\n"
}

// promptFingerprint extracts the SHA256 fingerprint an agent-style prompt
// embeds ("… SHA256:<fp>: …"), or "" when none is present.
func promptFingerprint(prompt string) string {
	hashType, rest, _ := strings.Cut(prompt, ":")
	if !strings.HasSuffix(hashType, "SHA256") {
		return ""
	}
	fp, _, _ := strings.Cut(rest, ":")
	return fp
}

// extractPin pulls the PIN out of pinentry's Assuan response: the data lines
// (those containing "D"), joined and stripped of the leading "D " marker.
func extractPin(out string) string {
	var b strings.Builder
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "D") {
			b.WriteString(line)
		}
	}
	joined := b.String()
	if len(joined) < 2 {
		return ""
	}
	return joined[2:]
}
