package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/BIJJUDAMA/llama-manager/config"
	"github.com/BIJJUDAMA/llama-manager/model"
)

type DownloaderFocus int

const (
	FocusURL DownloaderFocus = iota
	FocusFilename
	FocusQueue
)

type DownloaderModel struct {
	config          *config.Config
	queue           *model.DownloadQueue
	focus           DownloaderFocus
	selectedTaskIdx int
	err             error

	urlInput        textinput.Model
	filenameInput   textinput.Model
}

func NewDownloaderModel(cfg *config.Config, q *model.DownloadQueue) *DownloaderModel {
	urlTi := textinput.New()
	urlTi.Placeholder = "Paste direct GGUF model download URL (http/https)..."
	urlTi.CharLimit = 512
	urlTi.Width = 60
	urlTi.Focus()

	fileTi := textinput.New()
	fileTi.Placeholder = "Enter local filename (optional, e.g. my-model.gguf)..."
	fileTi.CharLimit = 156
	fileTi.Width = 60

	return &DownloaderModel{
		config:        cfg,
		queue:         q,
		focus:         FocusURL,
		urlInput:      urlTi,
		filenameInput: fileTi,
	}
}

func (m *DownloaderModel) Update(msg tea.Msg) (*DownloaderModel, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch m.focus {
	case FocusURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
		cmds = append(cmds, cmd)
	case FocusFilename:
		m.filenameInput, cmd = m.filenameInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.nextFocus()
		case "shift+tab":
			m.prevFocus()

		case "enter":
			if m.focus == FocusURL || m.focus == FocusFilename {
				urlStr := strings.TrimSpace(m.urlInput.Value())
				if urlStr != "" {
					filename := strings.TrimSpace(m.filenameInput.Value())
					if filename == "" {
						parts := strings.Split(urlStr, "/")
						if len(parts) > 0 {
							filename = parts[len(parts)-1]
							if qIdx := strings.Index(filename, "?"); qIdx != -1 {
								filename = filename[:qIdx]
							}
						}
					}
					if filename == "" {
						filename = "downloaded_model.gguf"
					}

					m.queue.AddTask("DirectDownload", filename, 0, urlStr)
					m.urlInput.SetValue("")
					m.filenameInput.SetValue("")
					m.focus = FocusURL
					m.urlInput.Focus()
					m.filenameInput.Blur()
					m.selectedTaskIdx = len(m.queue.GetTasks()) - 1
				}
			}

		case "up", "k":
			if m.focus == FocusQueue {
				m.moveCursor(-1)
			}
		case "down", "j":
			if m.focus == FocusQueue {
				m.moveCursor(1)
			}

		case "p", "P":
			if m.focus == FocusQueue {
				tasks := m.queue.GetTasks()
				if len(tasks) > 0 && m.selectedTaskIdx >= 0 && m.selectedTaskIdx < len(tasks) {
					t := tasks[m.selectedTaskIdx]
					if t.Status == model.StatusDownloading {
						m.queue.PauseTask(t)
					} else {
						m.queue.ResumeTask(t)
					}
				}
			}

		case "c", "C":
			if m.focus == FocusQueue {
				tasks := m.queue.GetTasks()
				if len(tasks) > 0 && m.selectedTaskIdx >= 0 && m.selectedTaskIdx < len(tasks) {
					m.queue.CancelTask(tasks[m.selectedTaskIdx])
					if m.selectedTaskIdx >= len(m.queue.GetTasks()) {
						m.selectedTaskIdx = len(m.queue.GetTasks()) - 1
					}
					if m.selectedTaskIdx < 0 {
						m.selectedTaskIdx = 0
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *DownloaderModel) nextFocus() {
	m.urlInput.Blur()
	m.filenameInput.Blur()
	switch m.focus {
	case FocusURL:
		m.focus = FocusFilename
		m.filenameInput.Focus()
	case FocusFilename:
		m.focus = FocusQueue
	case FocusQueue:
		m.focus = FocusURL
		m.urlInput.Focus()
	}
}

func (m *DownloaderModel) prevFocus() {
	m.urlInput.Blur()
	m.filenameInput.Blur()
	switch m.focus {
	case FocusURL:
		m.focus = FocusQueue
	case FocusFilename:
		m.focus = FocusURL
		m.urlInput.Focus()
	case FocusQueue:
		m.focus = FocusFilename
		m.filenameInput.Focus()
	}
}

func (m *DownloaderModel) moveCursor(dir int) {
	if m.focus == FocusQueue {
		tasks := m.queue.GetTasks()
		if len(tasks) == 0 {
			return
		}
		m.selectedTaskIdx += dir
		if m.selectedTaskIdx < 0 {
			m.selectedTaskIdx = 0
		}
		if m.selectedTaskIdx >= len(tasks) {
			m.selectedTaskIdx = len(tasks) - 1
		}
	}
}

func (m *DownloaderModel) View(width int, height int) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("DIRECT GGUF MODEL DOWNLOADER")))

	// Input form panel
	var directSb strings.Builder

	urlStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	fileStyle := lipgloss.NewStyle().Foreground(ColorWhite)
	if m.focus == FocusURL {
		urlStyle = urlStyle.Foreground(ColorSecondary).Bold(true)
	} else if m.focus == FocusFilename {
		fileStyle = fileStyle.Foreground(ColorSecondary).Bold(true)
	}

	directSb.WriteString("  " + urlStyle.Render("Direct URL:") + "\n")
	directSb.WriteString("  " + m.urlInput.View() + "\n\n")
	directSb.WriteString("  " + fileStyle.Render("Destination Filename (optional):") + "\n")
	directSb.WriteString("  " + m.filenameInput.View() + "\n\n")
	directSb.WriteString("  " + StyleHelp.Render("Leave filename empty to auto-extract from URL.") + "\n")

	panelHeight := 7
	formBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Width(width - 6).
		Height(panelHeight)
	if m.focus == FocusURL || m.focus == FocusFilename {
		formBorder = formBorder.BorderForeground(ColorPrimary)
	}

	sb.WriteString("  " + formBorder.Render(directSb.String()) + "\n\n")

	// Queue panel
	var queueSb strings.Builder
	queueSb.WriteString(lipgloss.NewStyle().Bold(true).Render("Download Queue:") + "\n")
	tasks := m.queue.GetTasks()
	if len(tasks) == 0 {
		queueSb.WriteString("  Queue is empty. Enter a GGUF URL above to start downloading.\n")
	} else {
		for idx, t := range tasks {
			statusStr := ""
			switch t.Status {
			case model.StatusQueued:
				statusStr = StyleBadgeStopped.Render(" QUEUED ")
			case model.StatusDownloading:
				statusStr = StyleBadgeRunning.Render(" DOWNLOADING ") + fmt.Sprintf(" %.1f KB/s", t.SpeedKBps)
			case model.StatusPaused:
				statusStr = StyleBadgeStarting.Render(" PAUSED ")
			case model.StatusCompleted:
				statusStr = StyleBadgeRunning.Render(" COMPLETED ")
			case model.StatusFailed:
				statusStr = StyleBadgeFailed.Render(" FAILED ") + fmt.Sprintf(": %v", t.Error)
			case model.StatusCanceled:
				statusStr = StyleBadgeStopped.Render(" CANCELED ")
			}

			progressFraction := 0.0
			if t.TotalSize > 0 {
				progressFraction = float64(t.Downloaded) / float64(t.TotalSize)
			}
			progressBar := renderProgressBar(20, progressFraction)

			row := fmt.Sprintf("%-20s %s %s (%s / %s)",
				t.FileName, progressBar, statusStr, formatSize(t.Downloaded), formatSize(t.TotalSize),
			)

			if m.focus == FocusQueue && idx == m.selectedTaskIdx {
				queueSb.WriteString(StyleSelectedListItem.Width(width - 8).Render(row) + "\n")
			} else {
				queueSb.WriteString(row + "\n")
			}
		}
	}

	queueHeight := height - panelHeight - 16
	if queueHeight < 4 {
		queueHeight = 4
	}

	panelBorderQueue := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Width(width - 6).
		Height(queueHeight)
	if m.focus == FocusQueue {
		panelBorderQueue = panelBorderQueue.BorderForeground(ColorPrimary)
	}

	sb.WriteString("  " + panelBorderQueue.Render(queueSb.String()) + "\n\n")

	// Help instructions
	var helpKeys []string
	helpKeys = append(helpKeys, fmt.Sprintf("%s Navigation", StyleHelpKey.Render("[Tab/Shift-Tab]")))
	if m.focus == FocusURL || m.focus == FocusFilename {
		helpKeys = append(helpKeys, fmt.Sprintf("%s Start Download", StyleHelpKey.Render("[Enter]")))
	} else if m.focus == FocusQueue {
		helpKeys = append(helpKeys, fmt.Sprintf("%s Pause/Resume", StyleHelpKey.Render("[P]")))
		helpKeys = append(helpKeys, fmt.Sprintf("%s Cancel/Remove", StyleHelpKey.Render("[C]")))
	}
	helpKeys = append(helpKeys, fmt.Sprintf("%s Return to Browser", StyleHelpKey.Render("[Esc]")))

	sb.WriteString("  " + strings.Join(helpKeys, "  ") + "\n")

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
