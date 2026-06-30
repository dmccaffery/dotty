// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package signingkey

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// residentRunner runs ssh-keygen -K in a chosen directory with inherited stdio,
// so its PIN and touch prompts reach the user directly.
type residentRunner interface {
	RunInteractiveDir(ctx context.Context, dir, name string, args ...string) error
}

// ScanDir discovers importable signing-key stubs under root. root may be a
// single stub file or a directory, which is walked recursively so both flat and
// <serial>/id_*_sk_* layouts work. A file is importable when its name parses via
// ParseStubName and a sibling .pub exists; Serial is left empty because the
// owning YubiKey is unknown until the stub is matched against hardware. Results
// are sorted by user then type for stable output.
func ScanDir(root string) ([]KeyRef, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", root, err)
	}

	var refs []KeyRef
	add := func(path string) {
		typ, user, ok := ParseStubName(filepath.Base(path))
		if !ok {
			return
		}
		pub := path + ".pub"
		if _, err := os.Stat(pub); err != nil {
			return // need the public key to match the stub against hardware
		}
		refs = append(refs, KeyRef{Type: typ, User: user, PrivPath: path, PubPath: pub})
	}

	if info.IsDir() {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				add(path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", root, err)
		}
	} else {
		add(root)
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].User != refs[j].User {
			return refs[i].User < refs[j].User
		}
		return refs[i].Type < refs[j].Type
	})
	return refs, nil
}

// ResidentPubKeys downloads the resident credentials from the touched FIDO
// authenticator and returns their public-key identities (PubKeyID). ssh-keygen
// -K writes the downloaded key files to its working directory, so dir should be
// a throwaway directory; -N "" keeps the discarded private files passphrase-free
// so only the FIDO PIN and a touch are prompted. With several authenticators
// attached, ssh-keygen downloads from the first one touched.
func ResidentPubKeys(ctx context.Context, r residentRunner, dir string) ([]string, error) {
	if err := r.RunInteractiveDir(ctx, dir, "ssh-keygen", "-K", "-N", ""); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read downloaded keys: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read downloaded key %s: %w", e.Name(), err)
		}
		if id := PubKeyID(firstLine(string(content))); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// Imported records one stub filed into the data dir.
type Imported struct {
	Source KeyRef // the source stub that was copied; its paths are the originals
	Serial string // the YubiKey whose hardware holds the credential
	Dest   string // the stub's new path in the data dir (.pub is Dest+".pub")
}

// Import files each source stub under the serial whose hardware holds its
// credential. residentBySerial maps a connected serial to the PubKeyIDs of the
// credentials resident on that key. A stub whose public key matches no serial is
// returned in skipped and left untouched on disk. When a destination already
// exists, replace decides whether to overwrite it (a declined replace drops the
// stub without reporting it as a non-match); a nil replace overwrites. Copies
// are written 0600 under a 0700 serial directory, matching the rest of the
// private data store.
func Import(
	srcRefs []KeyRef,
	residentBySerial map[string][]string,
	dataDir string,
	replace func(dest string) (bool, error),
) (imported []Imported, skipped []KeyRef, err error) {
	index := make(map[string]string) // PubKeyID -> serial
	for serial, ids := range residentBySerial {
		for _, id := range ids {
			index[id] = serial
		}
	}

	for _, ref := range srcRefs {
		pub, err := os.ReadFile(ref.PubPath)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", ref.PubPath, err)
		}
		serial, ok := index[PubKeyID(firstLine(string(pub)))]
		if !ok {
			skipped = append(skipped, ref)
			continue
		}

		dest := StubPath(dataDir, serial, ref.Type, ref.User)
		if _, statErr := os.Stat(dest); statErr == nil && replace != nil {
			ok, err := replace(dest)
			if err != nil {
				return nil, nil, err
			}
			if !ok {
				continue
			}
		}
		if err := copyStub(ref, dest); err != nil {
			return nil, nil, err
		}
		imported = append(imported, Imported{Source: ref, Serial: serial, Dest: dest})
	}
	return imported, skipped, nil
}

// copyStub writes a source stub and its public key to dest, creating the serial
// directory if needed.
func copyStub(ref KeyRef, dest string) error {
	priv, pub, err := Read(ref)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("create key dir: %w", err)
	}
	if err := os.WriteFile(dest, priv, 0o600); err != nil {
		return fmt.Errorf("write stub: %w", err)
	}
	if err := os.WriteFile(dest+".pub", pub, 0o600); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	return nil
}
