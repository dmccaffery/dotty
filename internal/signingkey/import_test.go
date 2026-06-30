// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// writeStub writes a source stub named name in dir. An empty pubLine omits the
// .pub sibling (the not-importable case).
func writeStub(t *testing.T, dir, name, pubLine string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	priv := filepath.Join(dir, name)
	body := "-----BEGIN OPENSSH PRIVATE KEY-----\nstub\n-----END OPENSSH PRIVATE KEY-----\n"
	if err := os.WriteFile(priv, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if pubLine != "" {
		if err := os.WriteFile(priv+".pub", []byte(pubLine+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return priv
}

func TestPubKeyID(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"sk-ssh-ed25519@openssh.com AAAA1 comment@host", "sk-ssh-ed25519@openssh.com AAAA1"},
		{"sk-ssh-ed25519@openssh.com AAAA1", "sk-ssh-ed25519@openssh.com AAAA1"},
		{"one-field", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := PubKeyID(tt.in); got != tt.want {
			t.Errorf("PubKeyID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestScanDir(t *testing.T) {
	t.Run("flat directory pairs stubs with their .pub", func(t *testing.T) {
		dir := t.TempDir()
		writeStub(t, dir, "id_ed25519_sk_alice", "sk-ssh-ed25519@openssh.com AAAA1 alice")
		writeStub(t, dir, "id_ecdsa_sk_alice", "sk-ecdsa-sha2-nistp256@openssh.com AAAA2 alice")
		writeStub(t, dir, "id_ed25519_sk_nopub", "") // missing .pub: not importable
		writeStub(t, dir, "notes.txt", "irrelevant") // not a stub name

		refs, err := ScanDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 2 {
			t.Fatalf("refs = %+v, want 2", refs)
		}
		// Sorted by user then type: ecdsa before ed25519.
		if refs[0].Type != "ecdsa" || refs[1].Type != "ed25519" {
			t.Errorf("order = %q, %q", refs[0].Type, refs[1].Type)
		}
		if refs[0].User != "alice" || refs[0].Serial != "" {
			t.Errorf("ref = %+v, want user alice and empty serial", refs[0])
		}
	})

	t.Run("walks a <serial>/id_* layout recursively", func(t *testing.T) {
		root := t.TempDir()
		writeStub(t, filepath.Join(root, "111"), "id_ed25519_sk_bob", "sk-ssh-ed25519@openssh.com AAAA3 bob")
		refs, err := ScanDir(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 1 || refs[0].User != "bob" {
			t.Fatalf("refs = %+v, want one bob stub", refs)
		}
	})

	t.Run("single stub file", func(t *testing.T) {
		dir := t.TempDir()
		priv := writeStub(t, dir, "id_ed25519_sk_carol", "sk-ssh-ed25519@openssh.com AAAA4 carol")
		refs, err := ScanDir(priv)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 1 || refs[0].PrivPath != priv {
			t.Fatalf("refs = %+v, want one ref at %s", refs, priv)
		}
	})

	t.Run("single stub file without .pub yields nothing", func(t *testing.T) {
		dir := t.TempDir()
		priv := writeStub(t, dir, "id_ed25519_sk_dave", "")
		refs, err := ScanDir(priv)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 0 {
			t.Fatalf("refs = %+v, want none", refs)
		}
	})

	t.Run("missing path errors", func(t *testing.T) {
		if _, err := ScanDir(filepath.Join(t.TempDir(), "nope")); err == nil {
			t.Error("ScanDir(missing) error = nil")
		}
	})
}

// fakeResident stands in for ssh-keygen -K: it records its invocation and
// writes the configured public keys (plus matching private files, which must be
// ignored) into the working directory.
type fakeResident struct {
	pubs []string
	err  error
	dir  string
	name string
	args []string
}

func (f *fakeResident) RunInteractiveDir(_ context.Context, dir, name string, args ...string) error {
	f.dir, f.name, f.args = dir, name, args
	if f.err != nil {
		return f.err
	}
	for i, line := range f.pubs {
		base := filepath.Join(dir, fmt.Sprintf("id_ed25519_sk_rk_%d", i))
		if err := os.WriteFile(base+".pub", []byte(line+"\n"), 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(base, []byte("stub"), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func TestResidentPubKeys(t *testing.T) {
	ctx := context.Background()

	t.Run("parses downloaded public keys", func(t *testing.T) {
		f := &fakeResident{pubs: []string{
			"sk-ssh-ed25519@openssh.com AAAA1 ssh:alice",
			"sk-ecdsa-sha2-nistp256@openssh.com AAAA2 ssh:bob",
		}}
		dir := t.TempDir()
		ids, err := ResidentPubKeys(ctx, f, dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 2 {
			t.Fatalf("ids = %v, want 2", ids)
		}
		if f.name != "ssh-keygen" || f.dir != dir {
			t.Errorf("ran %q in %q, want ssh-keygen in %q", f.name, f.dir, dir)
		}
		if f.args[0] != "-K" {
			t.Errorf("args = %v, want -K first", f.args)
		}
		if !slices.Contains(ids, "sk-ssh-ed25519@openssh.com AAAA1") {
			t.Errorf("ids = %v, missing ed25519 id", ids)
		}
	})

	t.Run("propagates the ssh-keygen error", func(t *testing.T) {
		f := &fakeResident{err: errors.New("boom")}
		if _, err := ResidentPubKeys(ctx, f, t.TempDir()); err == nil {
			t.Error("error = nil, want boom")
		}
	})
}

const (
	importPubA = "sk-ssh-ed25519@openssh.com AAAAA alice@host"
	importPubB = "sk-ssh-ed25519@openssh.com BBBBB bob@host"
	importPubC = "sk-ssh-ed25519@openssh.com CCCCC carol@host"
)

// importSource lays down alice, bob, and carol stubs in a scratch dir and
// returns the discovered refs.
func importSource(t *testing.T) []KeyRef {
	t.Helper()
	src := t.TempDir()
	writeStub(t, src, "id_ed25519_sk_alice", importPubA)
	writeStub(t, src, "id_ed25519_sk_bob", importPubB)
	writeStub(t, src, "id_ed25519_sk_carol", importPubC)
	refs, err := ScanDir(src)
	if err != nil {
		t.Fatal(err)
	}
	return refs
}

// importResident maps alice's credential to YubiKey 111 and bob's to 222;
// carol's is on no connected key.
func importResident() map[string][]string {
	return map[string][]string{
		"111": {PubKeyID(importPubA)},
		"222": {PubKeyID(importPubB)},
	}
}

// seedExisting writes an OLD stub at dest so replace-on-conflict can be tested.
func seedExisting(t *testing.T, dest string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("OLD"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestImportFilesMatchesAndSkipsRest(t *testing.T) {
	dataDir := t.TempDir()
	imported, skipped, err := Import(importSource(t), importResident(), dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 2 || len(skipped) != 1 {
		t.Fatalf("imported %d, skipped %d; want 2 and 1", len(imported), len(skipped))
	}
	if skipped[0].User != "carol" {
		t.Errorf("skipped %+v, want carol", skipped[0])
	}
	wantAlice := StubPath(dataDir, "111", "ed25519", "alice")
	if !fileEquals(t, wantAlice, "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Errorf("alice stub not written to %s", wantAlice)
	}
	if !fileEquals(t, wantAlice+".pub", importPubA) {
		t.Errorf("alice .pub not written to %s", wantAlice+".pub")
	}
	// Confirm serial attribution: bob lands under 222, not 111.
	if _, err := os.Stat(StubPath(dataDir, "222", "ed25519", "bob")); err != nil {
		t.Errorf("bob stub missing under 222: %v", err)
	}
}

func TestImportReplaceDeclinedKeepsExisting(t *testing.T) {
	dataDir := t.TempDir()
	dest := StubPath(dataDir, "111", "ed25519", "alice")
	seedExisting(t, dest)

	imported, skipped, err := Import(importSource(t), importResident(), dataDir, func(string) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// alice declined, bob imported; carol skipped (no match) — alice is not a skip.
	if len(imported) != 1 || imported[0].Source.User != "bob" {
		t.Errorf("imported = %+v, want only bob", imported)
	}
	if len(skipped) != 1 || skipped[0].User != "carol" {
		t.Errorf("skipped = %+v, want only carol", skipped)
	}
	if !fileEquals(t, dest, "OLD") {
		t.Error("alice stub was overwritten despite replace=false")
	}
}

func TestImportReplaceAcceptedOverwrites(t *testing.T) {
	dataDir := t.TempDir()
	dest := StubPath(dataDir, "111", "ed25519", "alice")
	seedExisting(t, dest)

	if _, _, err := Import(importSource(t), importResident(), dataDir, func(string) (bool, error) {
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if fileEquals(t, dest, "OLD") {
		t.Error("alice stub not overwritten despite replace=true")
	}
}

// fileEquals reports whether the file at path contains want (substring match,
// so callers can assert on a stable prefix without trailing-newline fuss).
func fileEquals(t *testing.T, path, want string) bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), want)
}
