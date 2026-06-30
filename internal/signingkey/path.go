// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ErrKeyNotFound reports that no key stub exists where one was expected.
var ErrKeyNotFound = errors.New("signing key not found")

// KeyRef locates one signing key's on-disk artifacts.
type KeyRef struct {
	Serial   string
	Type     string // ed25519 or ecdsa
	User     string
	PrivPath string // the key-handle stub
	PubPath  string
}

// KeyTypes are the supported -sk key types; the first is the default.
var KeyTypes = []string{"ed25519", "ecdsa"}

// KeyDir returns the directory holding a serial's key stubs.
func KeyDir(dataDir, serial string) string {
	return filepath.Join(dataDir, "security-key", serial)
}

// StubPath returns the stub location ssh-keygen writes for a key, matching
// DESIGN's id_<type>_sk_<user> layout.
func StubPath(dataDir, serial, typ, user string) string {
	return filepath.Join(KeyDir(dataDir, serial), fmt.Sprintf("id_%s_sk_%s", typ, user))
}

// stubNameRe parses stub filenames. The user segment is greedy — usernames
// may themselves contain underscores.
var stubNameRe = regexp.MustCompile(`^id_(ed25519|ecdsa)_sk_(.+)$`)

// ParseStubName splits a stub filename into key type and username. Public-key
// files and anything else return ok == false.
func ParseStubName(name string) (typ, user string, ok bool) {
	if strings.HasSuffix(name, ".pub") {
		return "", "", false
	}
	m := stubNameRe.FindStringSubmatch(name)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// Scan walks the data dir for key stubs. serials filters to those serials
// (nil means all); user filters to one username ("" means all). Results are
// sorted by serial, then user, then type.
func Scan(dataDir string, serials []string, user string) ([]KeyRef, error) {
	root := filepath.Join(dataDir, "security-key")
	entries, err := os.ReadDir(root)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", root, err)
	}

	want := map[string]bool{}
	for _, s := range serials {
		want[s] = true
	}

	var refs []KeyRef
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		serial := entry.Name()
		if len(want) > 0 && !want[serial] {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, serial))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filepath.Join(root, serial), err)
		}
		for _, f := range files {
			typ, u, ok := ParseStubName(f.Name())
			if !ok || (user != "" && u != user) {
				continue
			}
			priv := filepath.Join(root, serial, f.Name())
			refs = append(refs, KeyRef{
				Serial:   serial,
				Type:     typ,
				User:     u,
				PrivPath: priv,
				PubPath:  priv + ".pub",
			})
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		a, b := refs[i], refs[j]
		if a.Serial != b.Serial {
			return a.Serial < b.Serial
		}
		if a.User != b.User {
			return a.User < b.User
		}
		return a.Type < b.Type
	})
	return refs, nil
}

// Read returns the stub and public key contents for a ref, with ErrKeyNotFound
// when the stub is missing.
func Read(ref KeyRef) (priv, pub []byte, err error) {
	priv, err = os.ReadFile(ref.PrivPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("%w: expected stub at %s (run `dotty signing-key new`)", ErrKeyNotFound, ref.PrivPath)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read key stub: %w", err)
	}
	pub, err = os.ReadFile(ref.PubPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read public key: %w", err)
	}
	return priv, pub, nil
}

// MatchByPublicKey finds the ref whose public key matches line (algorithm and
// key blob — the first two fields; comments differ freely). git hands the
// signer a temp file holding just the public key; this maps it back to the
// stub that can actually sign.
func MatchByPublicKey(refs []KeyRef, line string) (KeyRef, bool) {
	want := PubKeyID(line)
	if want == "" {
		return KeyRef{}, false
	}
	for _, ref := range refs {
		pub, err := os.ReadFile(ref.PubPath)
		if err != nil {
			continue
		}
		if PubKeyID(string(pub)) == want {
			return ref, true
		}
	}
	return KeyRef{}, false
}

// PubKeyID reduces a public-key line to its identity — the algorithm and key
// blob (the first two fields), dropping the free-form comment. Two stubs share
// an identity exactly when they name the same credential, so this is what both
// git's literal-pubkey match and import's hardware match compare on.
func PubKeyID(s string) string {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return ""
	}
	return fields[0] + " " + fields[1]
}
