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

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testRows() []TableRow {
	return []TableRow{
		{Cells: []string{"111", "work", "ed25519", "alice"}, Value: "/k/alice"},
		{Cells: []string{"222", "backup", "ed25519", "bob"}, Value: "/k/bob"},
		{Cells: []string{"333", "", "ecdsa", "carol"}, Value: "/k/carol"},
	}
}

func pressTable(m tableModel, keys ...string) tableModel {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "backspace":
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, _ := m.Update(msg)
		m = next.(tableModel)
	}
	return m
}

func TestTableModelNavigationAndSelect(t *testing.T) {
	m := newTableModel("t", []string{"SERIAL", "ALIASES", "TYPE", "USERNAME"}, testRows())
	m = pressTable(m, "down", "enter")
	if !m.accepted {
		t.Fatal("enter did not accept")
	}
	if got := m.rows[m.visible[m.cursor]].Value; got != "/k/bob" {
		t.Errorf("selected = %q, want /k/bob", got)
	}
}

func TestTableModelFilter(t *testing.T) {
	m := newTableModel("t", []string{"SERIAL", "ALIASES", "TYPE", "USERNAME"}, testRows())
	m = pressTable(m, "c", "a", "r")
	if len(m.visible) != 1 {
		t.Fatalf("visible = %v, want 1 row", m.visible)
	}
	m = pressTable(m, "enter")
	if got := m.rows[m.visible[m.cursor]].Value; got != "/k/carol" {
		t.Errorf("selected = %q, want /k/carol", got)
	}

	t.Run("backspace widens the filter", func(t *testing.T) {
		m := pressTable(newTableModel("t", []string{"A"}, testRows()), "x", "y")
		if len(m.visible) != 0 {
			t.Fatalf("visible = %v, want none", m.visible)
		}
		m = pressTable(m, "backspace", "backspace")
		if len(m.visible) != 3 {
			t.Errorf("visible = %v, want all rows back", m.visible)
		}
	})

	t.Run("enter with no matches does not accept", func(t *testing.T) {
		m := pressTable(newTableModel("t", []string{"A"}, testRows()), "z", "z", "enter")
		if m.accepted {
			t.Error("accepted with no visible rows")
		}
	})
}

func TestTableModelEscape(t *testing.T) {
	m := pressTable(newTableModel("t", []string{"A"}, testRows()), "esc")
	if m.accepted {
		t.Error("esc accepted")
	}
}

func TestRenderTable(t *testing.T) {
	out := RenderTable([]string{"SERIAL", "ALIASES"}, testRows())
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want header + 3 rows", len(lines))
	}
	if !strings.HasPrefix(lines[0], "SERIAL") || !strings.Contains(lines[1], "111") {
		t.Errorf("table:\n%s", out)
	}
}
