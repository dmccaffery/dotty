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
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bitwise-media-group/dotty/internal/cli"
)

// ErrTimeout is returned when a polled wait expires before completing.
var ErrTimeout = errors.New("timed out")

// RunPoll shows a spinner with title and hint while calling poll every
// interval until poll reports done (nil return), poll errors, the timeout
// expires (ErrTimeout), or the user presses esc (ErrAborted). Results travel
// through the poll closure.
func RunPoll(ios cli.IOStreams, title, hint string, interval, timeout time.Duration, poll func() (bool, error)) error {
	if !ios.IsInteractive() {
		return ErrNotInteractive
	}
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	m := pollModel{spinner: sp, title: title, hint: hint, interval: interval, remaining: timeout, poll: poll}
	p := tea.NewProgram(m, tea.WithInput(ios.In), tea.WithOutput(ios.ErrOut))
	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("run poll UI: %w", err)
	}
	return final.(pollModel).result
}

type pollTickMsg struct{}

type pollModel struct {
	spinner   spinner.Model
	title     string
	hint      string
	interval  time.Duration
	remaining time.Duration
	poll      func() (bool, error)
	result    error
	done      bool
}

// Init starts the spinner and the poll cadence.
func (m pollModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.tick())
}

func (m pollModel) tick() tea.Cmd {
	return tea.Tick(m.interval, func(time.Time) tea.Msg { return pollTickMsg{} })
}

// Update drives the poll loop: esc aborts, each tick polls once and counts
// down the timeout.
func (m pollModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			m.result = ErrAborted
			m.done = true
			return m, tea.Quit
		}
	case pollTickMsg:
		done, err := m.poll()
		switch {
		case err != nil:
			m.result = err
			m.done = true
			return m, tea.Quit
		case done:
			m.done = true
			return m, tea.Quit
		}
		m.remaining -= m.interval
		if m.remaining <= 0 {
			m.result = ErrTimeout
			m.done = true
			return m, tea.Quit
		}
		return m, m.tick()
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// View renders the spinner line and hint.
func (m pollModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("\n  %s %s\n  %s\n", m.spinner.View(), m.title, infoStyle.Render(m.hint))
}
