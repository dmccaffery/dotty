// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package securitykey

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bitwise-media-group/dotty/internal/cli"
)

var (
	// ErrNameTaken reports an alias name that is already in use.
	ErrNameTaken = errors.New("alias name already in use")
	// ErrUnknownAlias reports an alias name with no entry behind it.
	ErrUnknownAlias = errors.New("unknown alias")
)

// storeVersion is the aliases.json schema version this build writes.
const storeVersion = 1

// Store is the on-disk register of security-key aliases. Alias names are
// globally unique — across serials — so a name resolves to exactly one key.
type Store struct {
	path string
	doc  storeDoc
}

type storeDoc struct {
	Version int                  `json:"version"`
	Keys    map[string]*keyEntry `json:"keys"`
}

type keyEntry struct {
	Aliases []Alias `json:"aliases"`
}

// StorePath returns the alias store's location inside a profile directory.
// The mapping travels with the profile (serials identify hardware a machine
// class owns); only the key stubs stay in the private data directory.
func StorePath(profileDir string) string {
	return filepath.Join(profileDir, "security-keys.json")
}

// LoadStore reads the alias store at path. A missing file is an empty store;
// a corrupt or duplicate-ridden file is a hard error naming the file — never
// silently recreated, it travels with the dotfiles repository.
func LoadStore(path string) (*Store, error) {
	s := &Store{path: path, doc: storeDoc{Version: storeVersion, Keys: map[string]*keyEntry{}}}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read alias store %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &s.doc); err != nil {
		return nil, fmt.Errorf("decode alias store %s: %w", path, err)
	}
	if s.doc.Keys == nil {
		s.doc.Keys = map[string]*keyEntry{}
	}
	seen := map[string]string{}
	for serial, entry := range s.doc.Keys {
		if !IsSerial(serial) {
			return nil, fmt.Errorf("alias store %s: key %q is not a serial number", path, serial)
		}
		for _, a := range entry.Aliases {
			if err := ValidateName(a.Name); err != nil {
				return nil, fmt.Errorf("alias store %s: %w", path, err)
			}
			if other, dup := seen[a.Name]; dup {
				return nil, fmt.Errorf("alias store %s: name %q appears under serials %s and %s", path, a.Name, other, serial)
			}
			seen[a.Name] = serial
		}
	}
	return s, nil
}

// Save writes the store atomically. It is profile content that travels with
// the dotfiles repository, so it takes ordinary repo permissions.
func (s *Store) Save() error {
	if err := cli.EnsureDir(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	s.doc.Version = storeVersion
	data, err := json.MarshalIndent(s.doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode alias store: %w", err)
	}
	return cli.AtomicWriteFile(s.path, append(data, '\n'), 0o644)
}

// Add registers an alias for serial, enforcing global name uniqueness.
func (s *Store) Add(serial, name, description string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if !IsSerial(serial) {
		return fmt.Errorf("serial %q must be numeric", serial)
	}
	if owner, ok := s.owner(name); ok {
		return fmt.Errorf("alias %q already names serial %s: %w", name, owner, ErrNameTaken)
	}
	entry := s.doc.Keys[serial]
	if entry == nil {
		entry = &keyEntry{}
		s.doc.Keys[serial] = entry
	}
	entry.Aliases = append(entry.Aliases, Alias{Name: name, Description: description, CreatedAt: time.Now().UTC()})
	return nil
}

// Remove drops the named aliases, returning how many were found. Serials left
// with no aliases are removed entirely.
func (s *Store) Remove(names ...string) int {
	removed := 0
	for _, name := range names {
		for serial, entry := range s.doc.Keys {
			if i := slices.IndexFunc(entry.Aliases, func(a Alias) bool { return a.Name == name }); i >= 0 {
				entry.Aliases = slices.Delete(entry.Aliases, i, i+1)
				removed++
			}
			if len(entry.Aliases) == 0 {
				delete(s.doc.Keys, serial)
			}
		}
	}
	return removed
}

// ResolveName returns the serial an alias names.
func (s *Store) ResolveName(name string) (string, error) {
	if serial, ok := s.owner(name); ok {
		return serial, nil
	}
	return "", fmt.Errorf("alias %q: %w", name, ErrUnknownAlias)
}

// AliasesBySerial returns all aliases grouped by serial, names sorted within
// each serial.
func (s *Store) AliasesBySerial() map[string][]Alias {
	out := make(map[string][]Alias, len(s.doc.Keys))
	for serial, entry := range s.doc.Keys {
		aliases := append([]Alias(nil), entry.Aliases...)
		slices.SortFunc(aliases, func(a, b Alias) int { return strings.Compare(a.Name, b.Name) })
		out[serial] = aliases
	}
	return out
}

// Names returns every alias name in the store, sorted.
func (s *Store) Names() []string {
	var names []string
	for _, entry := range s.doc.Keys {
		for _, a := range entry.Aliases {
			names = append(names, a.Name)
		}
	}
	slices.Sort(names)
	return names
}

func (s *Store) owner(name string) (string, bool) {
	for serial, entry := range s.doc.Keys {
		if slices.ContainsFunc(entry.Aliases, func(a Alias) bool { return a.Name == name }) {
			return serial, true
		}
	}
	return "", false
}
