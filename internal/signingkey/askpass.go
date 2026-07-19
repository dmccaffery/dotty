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

// PromptHintEnv is the variable OpenSSH sets to tell an askpass program what
// kind of reply it wants: "confirm" for a yes/no permission prompt, "none" for
// a display-only notification, unset when a secret is requested. The hint is
// not set for every non-secret prompt — the host-authenticity question is a
// plain echo prompt with no hint — so it is combined with prompt-shape
// detection in AskPassReply.
const PromptHintEnv = "SSH_ASKPASS_PROMPT"

// assuanRunner feeds an Assuan command block to pinentry's stdin and returns
// its stdout. A non-zero exit is not an error: pinentry exits non-zero when the
// user cancels, which surfaces here as an empty PIN.
type assuanRunner interface {
	RunAssuan(ctx context.Context, stdin, name string, args ...string) (string, error)
}

// AskPassReply computes the reply for an OpenSSH SSH_ASKPASS prompt. FIDO2
// user-presence prompts and display-only notifications (hint "none") only need
// a non-error reply, so the reply is empty and pinentry is never run; yes/no
// questions (hint "confirm", or an unhinted echo prompt such as the
// host-authenticity check) become a pinentry CONFIRM dialog answered with the
// literal "yes"/"no" ssh parses; only genuine secret requests reach GETPIN.
// hint is $SSH_ASKPASS_PROMPT, keyinfo the signing key's SHA256 fingerprint
// (from $DOTTY_SSH_KEYINFO) and read the file probe resolveKeyInfo uses to
// fingerprint the key file a client-auth prompt names — together they identify
// the key so pinentry can cache its PIN in the macOS keychain.
func AskPassReply(
	ctx context.Context,
	r assuanRunner,
	prompt, hint, keyinfo string,
	read func(string) ([]byte, error),
) string {
	// FIDO2 user-presence: ssh only needs a non-error reply, not a PIN.
	if hint == "none" || strings.HasPrefix(prompt, "Confirm user presence") {
		return ""
	}
	if hint == "confirm" || isYesNoPrompt(prompt) {
		return confirmReply(ctx, r, prompt)
	}
	// Ignore pinentry's exit status: a cancel yields no data line and an empty
	// PIN, which ssh treats as a failed unlock.
	out, _ := r.RunAssuan(ctx, buildAssuan(prompt, resolveKeyInfo(prompt, keyinfo, read)), PinentryPath)
	return extractPin(out)
}

// isYesNoPrompt reports whether an unhinted prompt expects a typed yes/no
// answer rather than a secret. ssh's host-authenticity check ("… continue
// connecting (yes/no/[fingerprint])?") and its re-ask ("Please type 'yes' or
// 'no': ") are echo prompts that carry no $SSH_ASKPASS_PROMPT hint, so the
// question shape in the text is the only signal.
func isYesNoPrompt(prompt string) bool {
	return strings.Contains(prompt, "yes/no") || strings.Contains(prompt, "'yes'")
}

// confirmReply renders a yes/no prompt as a pinentry CONFIRM dialog and
// returns the literal answer ssh expects: "yes" when the user confirms, "no"
// on cancel or any pinentry failure — rejection is the safe default for
// prompts like the host-authenticity check.
func confirmReply(ctx context.Context, r assuanRunner, prompt string) string {
	out, err := r.RunAssuan(ctx, "SETDESC "+escapeAssuan(prompt)+"\nCONFIRM\n", PinentryPath)
	if err != nil {
		return "no"
	}
	// pinentry answers CONFIRM with a bare OK, and a cancel with an ERR line.
	for line := range strings.Lines(out) {
		if strings.HasPrefix(line, "ERR") {
			return "no"
		}
	}
	return "yes"
}

// resolveKeyInfo returns the SHA256 fingerprint of the key a PIN prompt
// unlocks, or "" when the key can't be identified. The fingerprint comes from
// the first of: keyinfo, supplied out-of-band by `signing-key sign` because
// ssh-keygen's agentless prompt names no key; the prompt text, where
// agent-style prompts embed "SHA256:<fp>"; or the prompt's key path — ssh's
// client-auth prompt names the key file, whose .pub sidecar is fingerprinted.
// read keeps the file probe injectable for tests.
func resolveKeyInfo(prompt, keyinfo string, read func(string) ([]byte, error)) string {
	if keyinfo != "" {
		return keyinfo
	}
	if fp := promptFingerprint(prompt); fp != "" {
		return fp
	}
	path := promptKeyPath(prompt)
	if path == "" {
		return ""
	}
	pub, err := read(path + ".pub")
	if err != nil {
		return "" // no caching, but the prompt still works
	}
	return fingerprintB64(pubKeyBlob(string(pub)))
}

// promptKeyPath extracts the key file path from ssh's client-auth PIN prompt
// ("Enter PIN for <TYPE> key <path>: "), or "" for any other prompt shape —
// ssh-keygen's agentless variant ends at "key:" and names no path.
func promptKeyPath(prompt string) string {
	rest, ok := strings.CutPrefix(prompt, "Enter PIN for ")
	if !ok {
		return ""
	}
	_, path, ok := strings.Cut(rest, " key ")
	if !ok {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(path), ":")
}

// buildAssuan renders the pinentry Assuan command block for a PIN prompt. A
// non-empty keyinfo (the key's SHA256 fingerprint) is sent as SETKEYINFO
// together with allow-external-password-cache — what lets pinentry-mac store
// and reuse the PIN in the macOS keychain. Without it the prompt is bare and
// the keychain is never consulted.
func buildAssuan(prompt, keyinfo string) string {
	if keyinfo != "" {
		return "SETDESC " + escapeAssuan(prompt) + "\n" +
			"OPTION allow-external-password-cache\n" +
			"SETKEYINFO s/" + keyinfo + "\n" +
			"GETPIN\n"
	}
	return "SETDESC " + escapeAssuan(prompt) + "\nGETPIN\n"
}

// escapeAssuan percent-escapes the characters an Assuan command argument
// cannot carry verbatim: '%' itself and line breaks, which would otherwise
// terminate SETDESC early and leave the rest of a multi-line prompt (ssh's
// host-authenticity text, for one) interpreted as commands.
func escapeAssuan(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
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
