// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

// Package securitykey manages hardware security keys: named aliases for
// YubiKey serial numbers (kept in a private JSON store under
// $XDG_DATA_HOME/dotty/security-key) and the device discovery and selection
// flows shared by the security-key and signing-key commands.
//
// All hardware interaction shells out to ykman (serials) and fido2-token
// (FIDO HID enumeration); YubiKeys expose no USB serial number, so the only
// reliable serial-to-device-path mapping is watching both lists while the
// user replugs the intended key.
package securitykey
