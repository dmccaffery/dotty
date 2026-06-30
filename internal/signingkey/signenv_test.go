// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"slices"
	"testing"
)

func TestSignEnv(t *testing.T) {
	base := []string{
		"PATH=/bin",
		"SSH_AUTH_SOCK=/agent.sock",
		"SSH_ASKPASS=/old/askpass",
		"SSH_ASKPASS_REQUIRE=never",
		"DOTTY_ASKPASS=stale",
		"DOTTY_SSH_KEYINFO=stale",
	}

	t.Run("askpass forced, agent dropped, keyinfo set", func(t *testing.T) {
		got := SignEnv(base, "/usr/local/bin/dotty", "fp123")
		want := []string{
			"PATH=/bin",
			"SSH_ASKPASS=/usr/local/bin/dotty",
			"SSH_ASKPASS_REQUIRE=force",
			"DOTTY_ASKPASS=1",
			"DOTTY_SSH_KEYINFO=fp123",
		}
		if !slices.Equal(got, want) {
			t.Errorf("SignEnv =\n%v\nwant\n%v", got, want)
		}
	})

	t.Run("empty askpass leaves inherited SSH_ASKPASS untouched", func(t *testing.T) {
		got := SignEnv(base, "", "fp123")
		want := []string{
			"PATH=/bin",
			"SSH_ASKPASS=/old/askpass",
			"SSH_ASKPASS_REQUIRE=never",
			"DOTTY_SSH_KEYINFO=fp123",
		}
		if !slices.Equal(got, want) {
			t.Errorf("SignEnv =\n%v\nwant\n%v", got, want)
		}
	})

	t.Run("empty keyinfo omits the fingerprint var", func(t *testing.T) {
		got := SignEnv(base, "/usr/local/bin/dotty", "")
		if slices.ContainsFunc(got, func(kv string) bool { return len(kv) >= 17 && kv[:17] == "DOTTY_SSH_KEYINFO" }) {
			t.Errorf("SignEnv leaked a keyinfo var: %v", got)
		}
	})
}

func TestKeyInfoForArgs(t *testing.T) {
	dir := t.TempDir()
	raw := []byte("example sk-ed25519 key blob bytes, exactly as decoded")
	blob := base64.StdEncoding.EncodeToString(raw)
	stub := dir + "/id_ed25519_sk_alice"
	if err := os.WriteFile(stub+".pub", []byte("sk-ssh-ed25519@openssh.com "+blob+" alice@host\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(raw)
	wantFP := base64.RawStdEncoding.EncodeToString(sum[:])

	t.Run("reads the -f stub's sidecar and computes the fingerprint", func(t *testing.T) {
		got := KeyInfoForArgs([]string{"-Y", "sign", "-f", stub, "doc"}, os.ReadFile)
		if got != wantFP {
			t.Errorf("KeyInfoForArgs = %q, want %q", got, wantFP)
		}
	})

	t.Run("no -f yields empty keyinfo", func(t *testing.T) {
		if got := KeyInfoForArgs([]string{"-Y", "sign", "doc"}, os.ReadFile); got != "" {
			t.Errorf("KeyInfoForArgs = %q, want empty", got)
		}
	})

	t.Run("unreadable sidecar yields empty keyinfo", func(t *testing.T) {
		boom := func(string) ([]byte, error) { return nil, errors.New("nope") }
		if got := KeyInfoForArgs([]string{"-f", stub}, boom); got != "" {
			t.Errorf("KeyInfoForArgs = %q, want empty", got)
		}
	})
}
