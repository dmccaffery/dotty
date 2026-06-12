// MIT License
//
// Copyright (c) 2026 Bitwise Media Group
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

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
