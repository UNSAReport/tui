package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestRootNavigation(t *testing.T) {
	m := NewRootModel(RootOptions{Project: &ProjectContext{IsProject: false}})
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
