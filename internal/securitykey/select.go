// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package securitykey

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bitwise-media-group/dotty/internal/cli"
	"github.com/bitwise-media-group/dotty/internal/tui"
)

// Device pairs a YubiKey serial with its FIDO HID path. Path is empty when
// the serial-to-device mapping is unknown — callers then let ssh-keygen's own
// touch-select pick the hardware.
type Device struct {
	Serial string
	Path   string
}

// replug timing: 500ms keeps the poll responsive without hammering the HID
// stack; 60s is ample for a physical unplug/replug.
const (
	replugInterval = 500 * time.Millisecond
	replugTimeout  = 60 * time.Second
)

// ResolveSerial turns a --serial / --security-key value into a serial number.
// An all-digits ref is a serial as given; anything else is an alias. An empty
// ref resolves from the plugged-in keys: one key wins outright, several
// present a fuzzy picker (or ErrAmbiguousKey without a terminal), none is
// ErrNoKeyPresent.
func ResolveSerial(ctx context.Context, r Runner, store *Store, ios cli.IOStreams, ref string) (string, error) {
	if ref != "" {
		if IsSerial(ref) {
			return ref, nil
		}
		return store.ResolveName(ref)
	}
	serials, err := ListSerials(ctx, r)
	if err != nil {
		return "", err
	}
	switch len(serials) {
	case 0:
		return "", ErrNoKeyPresent
	case 1:
		return serials[0], nil
	}
	if !ios.IsInteractive() {
		return "", ErrAmbiguousKey
	}
	return PickSerial(ios, store, serials)
}

// PickSerial presents a fuzzy picker over the given serials, labelled with
// their aliases.
func PickSerial(ios cli.IOStreams, store *Store, serials []string) (string, error) {
	sort.Strings(serials)
	options := make([]tui.Option, len(serials))
	for i, serial := range serials {
		options[i] = tui.Option{Label: SerialLabel(store, serial), Value: serial}
	}
	return tui.Select(ios, "Which YubiKey?", options)
}

// SerialLabel renders a serial with its aliases for pickers and tables.
func SerialLabel(store *Store, serial string) string {
	aliases := store.AliasesBySerial()[serial]
	names := make([]string, 0, len(aliases))
	for _, a := range aliases {
		names = append(names, a.Name)
	}
	if len(names) == 0 {
		return serial
	}
	return fmt.Sprintf("%s — %s", serial, strings.Join(names, ", "))
}

// SelectDeviceForEnroll picks the YubiKey that `signing-key new` will write a
// resident credential to. With one key plugged in the answer is immediate and
// no device path is needed. With several, the user replugs the intended key —
// the serial and HID path that vanish and reappear together are the only
// reliable mapping, since YubiKeys expose no USB serial. Backing out of the
// replug (esc) falls back to a picker; the returned Device then has no Path
// and ssh-keygen's native touch-select chooses the hardware.
//
// wantSerial, when non-empty, pre-asserts which key must be chosen; replugging
// a different key is an error.
func SelectDeviceForEnroll(
	ctx context.Context, r Runner, store *Store, ios cli.IOStreams, wantSerial string,
) (Device, error) {
	serials, err := ListSerials(ctx, r)
	if err != nil {
		return Device{}, err
	}
	if wantSerial != "" && !contains(serials, wantSerial) {
		return Device{}, fmt.Errorf("YubiKey %s is not connected", wantSerial)
	}
	switch len(serials) {
	case 0:
		return Device{}, ErrNoKeyPresent
	case 1:
		return Device{Serial: serials[0]}, nil
	}

	devices, err := ListFIDODevices(ctx, r)
	if err != nil {
		return Device{}, err
	}
	paths := YubicoPaths(devices)
	if len(paths) != len(serials) {
		tui.Warnf(ios,
			"%d YubiKey serials but %d FIDO2 devices — an NFC-attached or serial-less key may be present",
			len(serials), len(paths))
	}

	tracker := NewReplugTracker(serials, paths)
	var picked Device
	title := "Unplug and re-insert the YubiKey you want to use"
	if wantSerial != "" {
		title = fmt.Sprintf("Unplug and re-insert YubiKey %s", wantSerial)
	}
	poll := func() (bool, error) {
		curSerials, err := ListSerials(ctx, r)
		if err != nil {
			return false, err
		}
		curDevices, err := ListFIDODevices(ctx, r)
		if err != nil {
			return false, err
		}
		dev, ok := tracker.Observe(curSerials, YubicoPaths(curDevices))
		if !ok {
			return false, nil
		}
		if wantSerial != "" && dev.Serial != wantSerial {
			return false, fmt.Errorf("replugged YubiKey %s, but --security-key names %s", dev.Serial, wantSerial)
		}
		picked = dev
		return true, nil
	}
	err = tui.RunPoll(ios, title, "esc to choose from a list instead", replugInterval, replugTimeout, poll)
	switch err {
	case nil:
		return picked, nil
	case tui.ErrAborted, tui.ErrTimeout, tui.ErrNotInteractive:
		// Fall back to picking by serial; ssh-keygen's touch-select will
		// choose the hardware, guided by an on-screen instruction.
		serial := wantSerial
		if serial == "" {
			if !ios.IsInteractive() {
				return Device{}, ErrAmbiguousKey
			}
			serial, err = PickSerial(ios, store, serials)
			if err != nil {
				return Device{}, err
			}
		}
		return Device{Serial: serial}, nil
	default:
		return Device{}, err
	}
}

// ReplugTracker watches successive (serials, paths) snapshots for the
// vanish-then-reappear pattern that identifies one physical key.
type ReplugTracker struct {
	prevSerials map[string]bool
	prevPaths   map[string]bool
	goneSerial  string
}

// NewReplugTracker seeds the tracker with the current plugged-in state.
func NewReplugTracker(serials, paths []string) *ReplugTracker {
	return &ReplugTracker{prevSerials: toSet(serials), prevPaths: toSet(paths)}
}

// Observe feeds the tracker a new snapshot. It reports the identified device
// once the previously vanished serial has reappeared; the device's Path is
// the HID path that appeared with it (empty if none did — possible when the
// OS reuses the old registry entry, in which case the caller falls back to
// touch-select).
func (t *ReplugTracker) Observe(serials, paths []string) (Device, bool) {
	curSerials, curPaths := toSet(serials), toSet(paths)
	defer func() {
		t.prevSerials, t.prevPaths = curSerials, curPaths
	}()

	if t.goneSerial == "" {
		for s := range t.prevSerials {
			if !curSerials[s] {
				t.goneSerial = s
				break
			}
		}
		return Device{}, false
	}
	if !curSerials[t.goneSerial] {
		return Device{}, false // still unplugged
	}
	dev := Device{Serial: t.goneSerial}
	for p := range curPaths {
		if !t.prevPaths[p] {
			dev.Path = p
			break
		}
	}
	return dev, true
}

func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
