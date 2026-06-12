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
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrNoKeyPresent reports that no YubiKey is plugged in.
	ErrNoKeyPresent = errors.New("no YubiKey detected")
	// ErrAmbiguousKey reports that several YubiKeys are plugged in and none
	// was specified in a context that cannot prompt.
	ErrAmbiguousKey = errors.New("multiple YubiKeys present; specify one with --serial or --security-key")
)

// Runner executes ykman and fido2-token on behalf of this package.
type Runner interface {
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// FIDODevice is one entry from `fido2-token -L`.
type FIDODevice struct {
	Path    string // e.g. "ioreg://4295277255" on darwin
	Vendor  string // 4 hex digits, lowercase, e.g. "1050"
	Product string
	Label   string
}

// yubicoVendorID is Yubico's USB vendor id as fido2-token prints it.
const yubicoVendorID = "1050"

// ListSerials returns the serials of all plugged-in YubiKeys via
// `ykman list --serials` (one decimal serial per line; keys without a
// readable serial are omitted by ykman).
func ListSerials(ctx context.Context, r Runner) ([]string, error) {
	out, err := r.Output(ctx, "ykman", "list", "--serials")
	if err != nil {
		return nil, fmt.Errorf("list YubiKeys: %w", err)
	}
	var serials []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !IsSerial(line) {
			return nil, fmt.Errorf("unexpected ykman serial line %q", line)
		}
		serials = append(serials, line)
	}
	return serials, nil
}

// fidoLineRe matches `fido2-token -L` output, e.g.
// "ioreg://4295277255: vendor=0x1050, product=0x0407 (Yubico YubiKey OTP+FIDO+CCID)".
var fidoLineRe = regexp.MustCompile(`^(\S+): vendor=0x([0-9a-fA-F]{4}), product=0x([0-9a-fA-F]{4}) \((.*)\)$`)

// ListFIDODevices returns all FIDO2 HID devices via `fido2-token -L`.
func ListFIDODevices(ctx context.Context, r Runner) ([]FIDODevice, error) {
	out, err := r.Output(ctx, "fido2-token", "-L")
	if err != nil {
		return nil, fmt.Errorf("list FIDO2 devices: %w", err)
	}
	var devices []FIDODevice
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := fidoLineRe.FindStringSubmatch(line)
		if m == nil {
			continue // fido2-token warnings and unrelated noise
		}
		devices = append(devices, FIDODevice{
			Path:    m[1],
			Vendor:  strings.ToLower(m[2]),
			Product: strings.ToLower(m[3]),
			Label:   m[4],
		})
	}
	return devices, nil
}

// YubicoPaths filters FIDO devices down to Yubico hardware and returns their
// HID paths.
func YubicoPaths(devices []FIDODevice) []string {
	var paths []string
	for _, d := range devices {
		if d.Vendor == yubicoVendorID {
			paths = append(paths, d.Path)
		}
	}
	return paths
}
