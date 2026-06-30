// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

const (
	// KeyInfoEnv carries the signing key's SHA256 fingerprint to the ask-pass
	// bridge so pinentry caches the PIN against the right key.
	KeyInfoEnv = "DOTTY_SSH_KEYINFO"
	// AskPassEnv marks the dotty binary as an $SSH_ASKPASS invocation; the argv
	// dispatcher routes such a call to `signing-key ask-pass`. A prompt
	// argument can't otherwise be told apart from an ordinary mistyped command.
	AskPassEnv = "DOTTY_ASKPASS"
)

// SignEnv rebuilds the environment for an agentless signing run, derived from
// base (normally os.Environ()). SSH_AUTH_SOCK is dropped so ssh-keygen signs
// with the stub itself rather than the agent; SSH_ASKPASS is pointed at dotty's
// own pinentry bridge and forced so PIN prompts route there even with a TTY;
// and the key fingerprint rides along in DOTTY_SSH_KEYINFO so pinentry can cache
// the PIN in the macOS keychain. askpass is the dotty binary path; "" leaves the
// inherited SSH_ASKPASS untouched (e.g. when the executable can't be resolved).
// keyinfo "" omits the fingerprint — signing still works, only without caching.
//
// The environment is rebuilt rather than appended to because the managed
// variables must be removed or reliably overridden, and a duplicate appended
// entry can't do either: getenv returns the first occurrence.
func SignEnv(base []string, askpass, keyinfo string) []string {
	drop := map[string]bool{
		"SSH_AUTH_SOCK": true,
		KeyInfoEnv:      true,
		AskPassEnv:      true,
	}
	if askpass != "" {
		drop["SSH_ASKPASS"] = true
		drop["SSH_ASKPASS_REQUIRE"] = true
	}

	env := make([]string, 0, len(base)+4)
	for _, kv := range base {
		name, _, _ := strings.Cut(kv, "=")
		if drop[name] {
			continue
		}
		env = append(env, kv)
	}
	if askpass != "" {
		env = append(env, "SSH_ASKPASS="+askpass, "SSH_ASKPASS_REQUIRE=force", AskPassEnv+"=1")
	}
	if keyinfo != "" {
		env = append(env, KeyInfoEnv+"="+keyinfo)
	}
	return env
}

// KeyInfoForArgs returns the OpenSSH SHA256 fingerprint (unpadded base64) of the
// key a rewritten sign argv will use, read from the -f stub's public-key
// sidecar. It returns "" when there is no -f or the public key can't be read:
// signing still works, only without PIN caching. read keeps the file probe
// injectable for tests.
func KeyInfoForArgs(args []string, read func(string) ([]byte, error)) string {
	stub := flagValue(args, "-f")
	if stub == "" {
		return ""
	}
	pub, err := read(stub + ".pub")
	if err != nil {
		return ""
	}
	return fingerprintB64(pubKeyBlob(string(pub)))
}

// flagValue returns the value following flag in args, or "" when flag is absent
// or has no following token.
func flagValue(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

// pubKeyBlob returns the base64 key blob (second field) of an authorized-keys
// style line, or "" when the line doesn't look like a public key.
func pubKeyBlob(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return ""
	}
	if _, err := base64.StdEncoding.DecodeString(fields[1]); err != nil {
		return ""
	}
	return fields[1]
}

// fingerprintB64 returns the OpenSSH SHA256 fingerprint (unpadded base64) of a
// base64 key blob — the value pinentry's keychain cache is keyed by ("" when the
// blob is empty or undecodable).
func fingerprintB64(blob string) string {
	if blob == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return base64.RawStdEncoding.EncodeToString(sum[:])
}
