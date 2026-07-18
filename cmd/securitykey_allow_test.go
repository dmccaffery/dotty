// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bitwise-media-group/dotty/internal/scaffold"
	"github.com/bitwise-media-group/dotty/internal/signingkey"
)

// allowEnv points the XDG dirs at scratch space, creates the named profiles
// (with "work" active), and undoes the persistent flag state the verbs share
// with the rest of the tree.
func allowEnv(t *testing.T, profiles ...string) (configDir, dataDir string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Cleanup(func() {
		rootFlags.Profile = ""
		signingKeyFlags.SecurityKey = ""
		signingKeyFlags.Username = ""
	})

	configDir = filepath.Join(home, ".config", "dotty")
	for _, name := range profiles {
		if err := os.MkdirAll(filepath.Join(configDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if len(profiles) > 0 {
		if err := os.Symlink(profiles[0], filepath.Join(configDir, "active-profile")); err != nil {
			t.Fatal(err)
		}
	}
	return configDir, filepath.Join(home, ".local", "share", "dotty")
}

func TestSecurityKeyAllowAndDisallow(t *testing.T) {
	configDir, _ := allowEnv(t, "work")

	// Allow two serials plus one resolved through an alias; both the alias
	// store and the allowlist land in the profile, not the data dir.
	if err := execDotty(t, "--profile=work", "security-key", "add", "--serial=333", "--name=extra"); err != nil {
		t.Fatalf("add alias: %v", err)
	}
	if err := execDotty(t, "--profile=work", "security-key", "allow", "111", "222", "extra"); err != nil {
		t.Fatalf("allow: %v", err)
	}

	answers, err := scaffold.LoadAnswers(filepath.Join(configDir, "work"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(answers.AllowedSerials, []string{"111", "222", "333"}) {
		t.Fatalf("work allowlist = %v", answers.AllowedSerials)
	}
	if _, err := os.Stat(filepath.Join(configDir, "work", "security-keys.json")); err != nil {
		t.Fatalf("alias store not in profile: %v", err)
	}

	// Disallowing removes one; the rest stand.
	if err := execDotty(t, "--profile=work", "security-key", "disallow", "222"); err != nil {
		t.Fatalf("disallow: %v", err)
	}
	answers, err = scaffold.LoadAnswers(filepath.Join(configDir, "work"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(answers.AllowedSerials, []string{"111", "333"}) {
		t.Fatalf("after disallow = %v", answers.AllowedSerials)
	}
}

func TestSigningKeyGetEnforcesAllowlist(t *testing.T) {
	_, dataDir := allowEnv(t, "work")

	// A stub for YubiKey 123 (stubs stay in the private data dir).
	stub := signingkey.StubPath(dataDir, "123", "ed25519", "u")
	if err := os.MkdirAll(filepath.Dir(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{stub: "stub", stub + ".pub": "ssh-ed25519 AAAA u"} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Unrestricted profile: the key resolves.
	if err := execDotty(t, "signing-key", "get", "--security-key=123", "--username=u"); err != nil {
		t.Fatalf("get without allowlist: %v", err)
	}

	// Restricted to another key: the same get is refused, naming the profile.
	if err := execDotty(t, "security-key", "allow", "999"); err != nil {
		t.Fatalf("allow: %v", err)
	}
	err := execDotty(t, "signing-key", "get", "--security-key=123", "--username=u")
	if !errors.Is(err, errKeyNotAllowed) || !strings.Contains(err.Error(), "work") {
		t.Fatalf("get err = %v, want allowlist refusal naming the profile", err)
	}

	// Allowing the key lifts the refusal.
	if err := execDotty(t, "security-key", "allow", "123"); err != nil {
		t.Fatalf("allow 123: %v", err)
	}
	if err := execDotty(t, "signing-key", "get", "--security-key=123", "--username=u"); err != nil {
		t.Fatalf("get after allowing: %v", err)
	}
}

// TestAllowlistSwapsWithProfile pins the point of profile-hosted key state:
// activating another profile swaps both the allowlist and the aliases.
func TestAllowlistSwapsWithProfile(t *testing.T) {
	configDir, dataDir := allowEnv(t, "work", "personal")

	stub := signingkey.StubPath(dataDir, "123", "ed25519", "u")
	if err := os.MkdirAll(filepath.Dir(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{stub: "stub", stub + ".pub": "ssh-ed25519 AAAA u"} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// work forbids 123; personal allows it.
	if err := execDotty(t, "--profile=work", "security-key", "allow", "999"); err != nil {
		t.Fatal(err)
	}
	if err := execDotty(t, "--profile=personal", "security-key", "allow", "123"); err != nil {
		t.Fatal(err)
	}
	rootFlags.Profile = ""

	if err := execDotty(t, "signing-key", "get", "--security-key=123", "--username=u"); err == nil {
		t.Fatal("work active: disallowed key resolved")
	}

	// Swap the active profile: same machine, same stub, different verdict.
	if err := os.Remove(filepath.Join(configDir, "active-profile")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("personal", filepath.Join(configDir, "active-profile")); err != nil {
		t.Fatal(err)
	}
	if err := execDotty(t, "signing-key", "get", "--security-key=123", "--username=u"); err != nil {
		t.Fatalf("personal active: allowed key refused: %v", err)
	}
}
