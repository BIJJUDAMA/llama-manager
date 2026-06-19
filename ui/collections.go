package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"llama-manager/config"
)

type CollectionsMode int

const (
	CollectionsListMode CollectionsMode = iota
	CollectionsCreateMode
)

type CollectionsModel struct {
	ModelPath string
	ModelName string
	config    *config.Config
	cursor    int
	mode      CollectionsMode
	textInput textinput.Model
}

func NewCollectionsModel(modelPath string, modelName string, cfg *config.Config) *CollectionsModel {
	ti := textinput.New()
	ti.Placeholder = "Collection Name"
	ti.CharLimit = 30
	ti.Width = 20

	return &CollectionsModel{
		ModelPath: modelPath,
		ModelName: modelName,
		config:    cfg,
		cursor:    0,
		mode:      CollectionsListMode,
		textInput: ti,
	}
}

func (c *CollectionsModel) getCollectionNames() []string {
	var names []string
	for k := range c.config.Collections {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func (c *CollectionsModel) Update(msg tea.Msg) tea.Cmd {
	if c.mode == CollectionsCreateMode {
		var cmd tea.Cmd
		c.textInput, cmd = c.textInput.Update(msg)

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				name := strings.TrimSpace(c.textInput.Value())
				if name != "" {
					c.config.AddToCollection(name, c.ModelPath)
					_ = c.config.Save()
					c.textInput.SetValue("")
					c.mode = CollectionsListMode
					// Set cursor to the newly created collection
					names := c.getCollectionNames()
					for idx, n := range names {
						if n == name {
							c.cursor = idx
							break
						}
					}
				}
			case "esc":
				c.textInput.SetValue("")
				c.mode = CollectionsListMode
			}
		}
		return cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		names := c.getCollectionNames()
		totalItems := len(names) + 1

		switch msg.String() {
		case "up", "k":
			if c.cursor > 0 {
				c.cursor--
			}
		case "down", "j":
			if c.cursor < totalItems-1 {
				c.cursor++
			}
		case "n", "N":
			c.mode = CollectionsCreateMode
			c.textInput.Focus()
			c.textInput.SetValue("")
		case "enter":
			if c.cursor == len(names) {
				c.mode = CollectionsCreateMode
				c.textInput.Focus()
				c.textInput.SetValue("")
			} else {
				colName := names[c.cursor]
				// Check membership
				isMember := false
				for _, p := range c.config.Collections[colName] {
					if p == c.ModelPath {
						isMember = true
						break
					}
				}
				if isMember {
					c.config.RemoveFromCollection(colName, c.ModelPath)
				} else {
					c.config.AddToCollection(colName, c.ModelPath)
				}
				_ = c.config.Save()
			}
		}
	}
	return nil
}

func (c *CollectionsModel) View(width int, height int) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("COLLECTIONS MANAGER")))
	sb.WriteString(fmt.Sprintf("  Model: %s\n\n", lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(c.ModelName)))

	if c.mode == CollectionsCreateMode {
		sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Create New Collection:") + "\n\n")
		sb.WriteString("  Name: " + c.textInput.View() + "\n\n")

		helpStr := fmt.Sprintf("%s Create Collection  %s Cancel",
			StyleHelpKey.Render("[Enter]"),
			StyleHelpKey.Render("[Esc]"),
		)
		sb.WriteString("  " + helpStr + "\n")
	} else {
		names := c.getCollectionNames()

		sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Select collections to assign:") + "\n\n")
		for idx, name := range names {
			// Check membership
			isMember := false
			for _, p := range c.config.Collections[name] {
				if p == c.ModelPath {
					isMember = true
					break
				}
			}

			checkbox := "[ ]"
			if isMember {
				checkbox = StyleSuccess.Render("[x]")
			}

			cursorStr := "  "
			if idx == c.cursor {
				cursorStr = StyleHelpKey.Render("▸ ")
				sb.WriteString(fmt.Sprintf(" %s %s %s\n", cursorStr, checkbox, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(name)))
			} else {
				sb.WriteString(fmt.Sprintf(" %s %s %s\n", cursorStr, checkbox, name))
			}
		}

		// Render virtual item
		cursorStr := "  "
		itemText := "+ Create New Collection..."
		if c.cursor == len(names) {
			cursorStr = StyleHelpKey.Render("▸ ")
			sb.WriteString(fmt.Sprintf(" %s %s\n\n", cursorStr, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(itemText)))
		} else {
			sb.WriteString(fmt.Sprintf(" %s %s\n\n", cursorStr, lipgloss.NewStyle().Foreground(ColorMuted).Render(itemText)))
		}

		helpStr := fmt.Sprintf("%s Toggle/Select  %s New  %s Close",
			StyleHelpKey.Render("[Enter]"),
			StyleHelpKey.Render("[N]"),
			StyleHelpKey.Render("[Esc/C]"),
		)
		sb.WriteString("  " + helpStr + "\n")
	}

	boxWidth := width - 4
	if boxWidth < 50 {
		boxWidth = 50
	}
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Width(boxWidth).
		Render(sb.String())
}
