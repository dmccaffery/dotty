// Copyright 2026 Bitwise Media Group Ltd.
// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/bitwise-media-group/dotty/internal/cli"
)

// Theme returns the huh theme every dotty form uses, so prompts look the same
// across commands.
func Theme() *huh.Theme {
	return huh.ThemeCharm()
}

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"})
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#FF6F00", Dark: "#FFB454"}).Bold(true)
)

// Successf prints a styled success notice to ErrOut.
func Successf(ios cli.IOStreams, format string, a ...any) {
	notice(ios.ErrOut, successStyle, "✓", format, a...)
}

// Infof prints a styled informational notice to ErrOut.
func Infof(ios cli.IOStreams, format string, a ...any) {
	notice(ios.ErrOut, infoStyle, "•", format, a...)
}

// Warnf prints a styled warning notice to ErrOut.
func Warnf(ios cli.IOStreams, format string, a ...any) {
	notice(ios.ErrOut, warnStyle, "!", format, a...)
}

func notice(w io.Writer, style lipgloss.Style, glyph, format string, a ...any) {
	_, _ = fmt.Fprintf(w, "%s %s\n", style.Render(glyph), fmt.Sprintf(format, a...))
}
