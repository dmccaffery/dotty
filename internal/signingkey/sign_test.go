// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestRewriteSignArgs(t *testing.T) {
	dataDir := t.TempDir()
	alice := writeKey(t, dataDir, "111", "ed25519", "alice", "sk-ssh-ed25519@openssh.com AAAA1 alice@host")
	scan := func() ([]KeyRef, error) { return Scan(dataDir, nil, "") }
	noDefault := func() (string, error) { return "", errors.New("default key resolution must not run") }

	// A temp file standing in for git's literal-pubkey key file.
	gitKeyFile := dataDir + "/.git_signing_key_tmp"
	if err := os.WriteFile(gitKeyFile, []byte("sk-ssh-ed25519@openssh.com AAAA1 alice@host\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("git literal-pubkey flow: -f rewritten to the stub, -U dropped", func(t *testing.T) {
		args := []string{"-Y", "sign", "-n", "git", "-f", gitKeyFile, "-U", "/tmp/buffer"}
		got, err := RewriteSignArgs(args, noDefault, scan, os.ReadFile)
		if err != nil {
			t.Fatalf("RewriteSignArgs() error: %v", err)
		}
		want := []string{"-Y", "sign", "-n", "git", "-f", alice.PrivPath, "/tmp/buffer"}
		if !slices.Equal(got, want) {
			t.Errorf("args =\n%v\nwant\n%v", got, want)
		}
	})

	t.Run("unknown literal pubkey is a hard error", func(t *testing.T) {
		unknown := dataDir + "/unknown_key"
		if err := os.WriteFile(unknown, []byte("sk-ssh-ed25519@openssh.com NOPE x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := RewriteSignArgs([]string{"-Y", "sign", "-n", "git", "-f", unknown, "/buf"}, noDefault, scan, os.ReadFile)
		if err == nil || !strings.Contains(err.Error(), "signing-key list") {
			t.Errorf("error = %v, want unknown-key guidance", err)
		}
	})

	t.Run("static stub via user.signingKey passes through untouched", func(t *testing.T) {
		args := []string{"-Y", "sign", "-n", "git", "-f", alice.PrivPath, "/tmp/buffer"}
		got, err := RewriteSignArgs(args, noDefault, scan, os.ReadFile)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if !slices.Equal(got, args) {
			t.Errorf("args = %v, want untouched %v", got, args)
		}
	})

	t.Run("unreadable -f passes through for ssh-keygen to report", func(t *testing.T) {
		args := []string{"-Y", "sign", "-f", "/does/not/exist", "/buf"}
		got, err := RewriteSignArgs(args, noDefault, scan, os.ReadFile)
		if err != nil || !slices.Equal(got, args) {
			t.Errorf("= %v, %v; want untouched", got, err)
		}
	})

	t.Run("human invocation gets -Y sign, -n file, and the resolved stub", func(t *testing.T) {
		resolve := func() (string, error) { return alice.PrivPath, nil }
		got, err := RewriteSignArgs([]string{"document.txt"}, resolve, scan, os.ReadFile)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		want := []string{"-Y", "sign", "-n", "file", "-f", alice.PrivPath, "document.txt"}
		if !slices.Equal(got, want) {
			t.Errorf("args =\n%v\nwant\n%v", got, want)
		}
	})

	t.Run("explicit -n is respected", func(t *testing.T) {
		resolve := func() (string, error) { return alice.PrivPath, nil }
		got, err := RewriteSignArgs([]string{"-n", "release", "artifact.tgz"}, resolve, scan, os.ReadFile)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		want := []string{"-Y", "sign", "-f", alice.PrivPath, "-n", "release", "artifact.tgz"}
		if !slices.Equal(got, want) {
			t.Errorf("args =\n%v\nwant\n%v", got, want)
		}
	})

	t.Run("default resolution failure surfaces", func(t *testing.T) {
		boom := func() (string, error) { return "", errors.New("no key") }
		if _, err := RewriteSignArgs([]string{"f.txt"}, boom, scan, os.ReadFile); err == nil {
			t.Error("error = nil, want resolution failure")
		}
	})
}

func TestFormatGitKey(t *testing.T) {
	pub := []byte("sk-ssh-ed25519@openssh.com AAAA1 alice@host\n")
	want := "key::sk-ssh-ed25519@openssh.com AAAA1 alice@host"
	if got := FormatGitKey(pub); got != want {
		t.Errorf("FormatGitKey() = %q, want %q", got, want)
	}
}
