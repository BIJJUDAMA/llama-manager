package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"llama-manager/config"
)

type EditorType int

const (
	EditorTags EditorType = iota
	EditorNotes
)

type EditorModel struct {
	ModelPath  string
	ModelName  string
	config     *config.Config
	editorType EditorType
	tagInput   textinput.Model
	noteInput  textarea.Model
}

func NewEditorModel(modelPath string, modelName string, cfg *config.Config, editorType EditorType) *EditorModel {
	ti := textinput.New()
	ti.Placeholder = "Enter tags (comma-separated, e.g. Coding, MoE)"
	ti.CharLimit = 156
	ti.Width = 50

	ta := textarea.New()
	ta.Placeholder = "Enter notes..."
	ta.SetWidth(60)
	ta.SetHeight(8)

	if editorType == EditorTags {
		tags := cfg.ModelTags[modelPath]
		ti.SetValue(strings.Join(tags, ", "))
		ti.Focus()
	} else {
		notes := cfg.GetNotes(modelPath)
		ta.SetValue(notes)
		ta.Focus()
	}

	return &EditorModel{
		ModelPath:  modelPath,
		ModelName:  modelName,
		config:     cfg,
		editorType: editorType,
		tagInput:   ti,
		noteInput:  ta,
	}
}

func (e *EditorModel) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch e.editorType {
	case EditorTags:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter":
				val := e.tagInput.Value()
				parts := strings.Split(val, ",")
				var cleaned []string
				seen := make(map[string]bool)
				for _, p := range parts {
					t := strings.TrimSpace(p)
					if t != "" && !seen[t] {
						seen[t] = true
						cleaned = append(cleaned, t)
					}
				}
				if e.config.ModelTags == nil {
					e.config.ModelTags = make(map[string][]string)
				}
				e.config.ModelTags[e.ModelPath] = cleaned
				_ = e.config.Save()
				return nil, true
			case "esc":
				return nil, true
			}
		}
		var cmd tea.Cmd
		e.tagInput, cmd = e.tagInput.Update(msg)
		return cmd, false

	case EditorNotes:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "ctrl+s":
				notes := e.noteInput.Value()
				e.config.SetNotes(e.ModelPath, notes)
				_ = e.config.Save()
				return nil, true
			case "esc":
				return nil, true
			}
		}
		var cmd tea.Cmd
		e.noteInput, cmd = e.noteInput.Update(msg)
		return cmd, false
	}
	return nil, false
}

func (e *EditorModel) View(width int, height int) string {
	var sb strings.Builder
	sb.WriteString("\n")
	if e.editorType == EditorTags {
		sb.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("EDIT MODEL TAGS")))
		sb.WriteString(fmt.Sprintf("  Model: %s\n\n", lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(e.ModelName)))
		sb.WriteString("  Enter tags (comma-separated):\n")
		sb.WriteString("  " + e.tagInput.View() + "\n\n")

		helpStr := fmt.Sprintf("%s Save  %s Cancel",
			StyleHelpKey.Render("[Enter]"),
			StyleHelpKey.Render("[Esc]"),
		)
		sb.WriteString("  " + helpStr + "\n")
	} else {
		sb.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("EDIT MODEL NOTES")))
		sb.WriteString(fmt.Sprintf("  Model: %s\n\n", lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(e.ModelName)))
		sb.WriteString("  Enter notes:\n")
		sb.WriteString("  " + e.noteInput.View() + "\n\n")

		helpStr := fmt.Sprintf("%s Save  %s Cancel",
			StyleHelpKey.Render("[Ctrl+S]"),
			StyleHelpKey.Render("[Esc]"),
		)
		sb.WriteString("  " + helpStr + "\n")
	}

	boxWidth := width - 4
	if boxWidth < 66 {
		boxWidth = 66
	}
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Width(boxWidth).
		Render(sb.String())
}
