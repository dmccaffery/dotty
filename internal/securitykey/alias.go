// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package securitykey

import (
	"fmt"
	"regexp"
	"time"
)

// Alias is one named handle for a security key serial.
type Alias struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// nameRe requires a leading letter, so an alias can never be mistaken for a
// serial number (serials are all digits) when both are accepted in one flag.
var nameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`)

// serialRe matches YubiKey serial numbers as ykman prints them.
var serialRe = regexp.MustCompile(`^[0-9]+$`)

// ValidateName rejects alias names that could collide with serials or break
// the store.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("alias name must not be empty")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("alias name %q must start with a letter and contain only letters, digits, '.', '_', or '-'", name)
	}
	return nil
}

// IsSerial reports whether ref looks like a serial number rather than an
// alias name.
func IsSerial(ref string) bool {
	return serialRe.MatchString(ref)
}
