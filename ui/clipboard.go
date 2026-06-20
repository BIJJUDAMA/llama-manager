package ui

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
)

// pasteFromClipboard inserts the clipboard text at the current cursor position of the given text input.
func pasteFromClipboard(ti *textinput.Model) {
	text, err := clipboard.ReadAll()
	if err == nil && text != "" {
		// Single line inputs should not contain newlines or carriage returns
		text = strings.ReplaceAll(text, "\r", "")
		text = strings.ReplaceAll(text, "\n", " ")
		text = strings.TrimSpace(text)

		val := ti.Value()
		pos := ti.Position()
		runes := []rune(val)
		if pos < 0 {
			pos = 0
		}
		if pos > len(runes) {
			pos = len(runes)
		}
		newRunes := append(runes[:pos], append([]rune(text), runes[pos:]...)...)
		ti.SetValue(string(newRunes))
		ti.SetCursor(pos + len([]rune(text)))
	}
}
