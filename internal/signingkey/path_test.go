// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStubPath(t *testing.T) {
	got := StubPath("/data", "12345678", "ed25519", "deavon")
	want := filepath.Join("/data", "security-key", "12345678", "id_ed25519_sk_deavon")
	if got != want {
		t.Errorf("StubPath() = %q, want %q", got, want)
	}
}

func TestParseStubName(t *testing.T) {
	tests := []struct {
		name     string
		wantType string
		wantUser string
		wantOK   bool
	}{
		{name: "id_ed25519_sk_deavon", wantType: "ed25519", wantUser: "deavon", wantOK: true},
		{name: "id_ecdsa_sk_deavon", wantType: "ecdsa", wantUser: "deavon", wantOK: true},
		{name: "id_ed25519_sk_my_user", wantType: "ed25519", wantUser: "my_user", wantOK: true},
		{name: "id_ed25519_sk_deavon.pub", wantOK: false},
		{name: "id_rsa", wantOK: false},
		{name: "id_dsa_sk_x", wantOK: false},
		{name: "random.txt", wantOK: false},
		{name: "id_ed25519_sk_", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, user, ok := ParseStubName(tt.name)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && (typ != tt.wantType || user != tt.wantUser) {
				t.Errorf("= (%q, %q), want (%q, %q)", typ, user, tt.wantType, tt.wantUser)
			}
		})
	}
}

func FuzzParseStubName(f *testing.F) {
	f.Add("id_ed25519_sk_deavon")
	f.Add("id_ecdsa_sk_a_b_c.pub")
	f.Add("")
	f.Fuzz(func(t *testing.T, name string) {
		typ, user, ok := ParseStubName(name)
		if !ok {
			return
		}
		// Round-trip invariant: a parsed name rebuilds to itself, so Scan and
		// StubPath can never disagree about a stub's identity.
		if rebuilt := "id_" + typ + "_sk_" + user; rebuilt != name {
			t.Errorf("rebuilt %q != input %q", rebuilt, name)
		}
	})
}

// writeKey creates a stub + pub pair for tests.
func writeKey(t *testing.T, dataDir, serial, typ, user, pubLine string) KeyRef {
	t.Helper()
	dir := KeyDir(dataDir, serial)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	priv := StubPath(dataDir, serial, typ, user)
	if err := os.WriteFile(priv, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nstub\n-----END OPENSSH PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(priv+".pub", []byte(pubLine+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return KeyRef{Serial: serial, Type: typ, User: user, PrivPath: priv, PubPath: priv + ".pub"}
}

func TestScan(t *testing.T) {
	dataDir := t.TempDir()
	writeKey(t, dataDir, "111", "ed25519", "alice", "sk-ssh-ed25519@openssh.com AAAA1 alice")
	writeKey(t, dataDir, "111", "ecdsa", "alice", "sk-ecdsa-sha2-nistp256@openssh.com AAAA2 alice")
	writeKey(t, dataDir, "222", "ed25519", "bob", "sk-ssh-ed25519@openssh.com AAAA3 bob")

	t.Run("all", func(t *testing.T) {
		refs, err := Scan(dataDir, nil, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 3 {
			t.Fatalf("refs = %d, want 3", len(refs))
		}
		// Sorted: serial, user, type.
		if refs[0].Type != "ecdsa" || refs[1].Type != "ed25519" || refs[2].Serial != "222" {
			t.Errorf("order = %+v", refs)
		}
	})

	t.Run("filter by serial and user", func(t *testing.T) {
		refs, err := Scan(dataDir, []string{"111"}, "alice")
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 2 {
			t.Errorf("refs = %+v, want 2", refs)
		}
		refs, err = Scan(dataDir, []string{"222"}, "alice")
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 0 {
			t.Errorf("refs = %+v, want none", refs)
		}
	})

	t.Run("missing data dir is empty", func(t *testing.T) {
		refs, err := Scan(filepath.Join(t.TempDir(), "none"), nil, "")
		if err != nil || refs != nil {
			t.Errorf("= %v, %v; want nil, nil", refs, err)
		}
	})
}

func TestRead(t *testing.T) {
	dataDir := t.TempDir()
	ref := writeKey(t, dataDir, "111", "ed25519", "alice", "sk-ssh-ed25519@openssh.com AAAA1 alice")

	priv, pub, err := Read(ref)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(priv) == 0 || len(pub) == 0 {
		t.Error("empty contents")
	}

	missing := KeyRef{PrivPath: filepath.Join(dataDir, "nope"), PubPath: filepath.Join(dataDir, "nope.pub")}
	if _, _, err := Read(missing); !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("error = %v, want ErrKeyNotFound", err)
	}
}

func TestMatchByPublicKey(t *testing.T) {
	dataDir := t.TempDir()
	alice := writeKey(t, dataDir, "111", "ed25519", "alice", "sk-ssh-ed25519@openssh.com AAAA1 alice@host")
	writeKey(t, dataDir, "222", "ed25519", "bob", "sk-ssh-ed25519@openssh.com AAAA3 bob@host")
	refs, err := Scan(dataDir, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("matches on algorithm and blob, ignoring comment", func(t *testing.T) {
		got, ok := MatchByPublicKey(refs, "sk-ssh-ed25519@openssh.com AAAA1 different-comment")
		if !ok || got.PrivPath != alice.PrivPath {
			t.Errorf("= %+v, %v; want alice's stub", got, ok)
		}
	})

	t.Run("no match", func(t *testing.T) {
		if _, ok := MatchByPublicKey(refs, "sk-ssh-ed25519@openssh.com UNKNOWN x"); ok {
			t.Error("unexpected match")
		}
	})

	t.Run("garbage line", func(t *testing.T) {
		if _, ok := MatchByPublicKey(refs, "one-field"); ok {
			t.Error("unexpected match for malformed line")
		}
	})
}
