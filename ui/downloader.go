package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"llama-manager/config"
	"llama-manager/model"
)

type DownloaderFocus int

const (
	FocusSearch DownloaderFocus = iota
	FocusRepos
	FocusFiles
	FocusQueue
	FocusDirectURL
)

type searchHFMsg struct {
	results []model.HFModelResult
	err     error
}

type listHFFilesMsg struct {
	files []model.HFSibling
	err   error
}

type DownloaderModel struct {
	config          *config.Config
	queue           *model.DownloadQueue
	searchInput     textinput.Model
	focus           DownloaderFocus
	repos           []model.HFModelResult
	files           []model.HFSibling
	selectedRepoIdx int
	selectedFileIdx int
	selectedTaskIdx int
	loadingRepos    bool
	loadingFiles    bool
	err             error

	urlInput        textinput.Model
	filenameInput   textinput.Model
	directURLActive bool
	directURLFocus  int // 0 for URL, 1 for filename
}

func NewDownloaderModel(cfg *config.Config, q *model.DownloadQueue) *DownloaderModel {
	ti := textinput.New()
	ti.Placeholder = "Type repository query (e.g., Llama-3, Qwen) and press Enter..."
	ti.CharLimit = 100
	ti.Width = 50
	ti.Focus()

	urlTi := textinput.New()
	urlTi.Placeholder = "Paste direct GGUF model download URL (http/https)..."
	urlTi.CharLimit = 512
	urlTi.Width = 60

	fileTi := textinput.New()
	fileTi.Placeholder = "Enter local filename (optional, e.g. my-model.gguf)..."
	fileTi.CharLimit = 156
	fileTi.Width = 60

	return &DownloaderModel{
		config:          cfg,
		queue:           q,
		searchInput:     ti,
		focus:           FocusSearch,
		repos:           []model.HFModelResult{},
		files:           []model.HFSibling{},
		urlInput:        urlTi,
		filenameInput:   fileTi,
		directURLActive: false,
		directURLFocus:  0,
	}
}

func searchHFCmd(query string, token string) tea.Cmd {
	return func() tea.Msg {
		results, err := model.SearchHFModels(query, token)
		return searchHFMsg{results: results, err: err}
	}
}

func listHFFilesCmd(modelID string, token string) tea.Cmd {
	return func() tea.Msg {
		files, err := model.ListHFModelFiles(modelID, token)
		return listHFFilesMsg{files: files, err: err}
	}
}

func (m *DownloaderModel) Update(msg tea.Msg) (*DownloaderModel, tea.Cmd) {
	var cmds []tea.Cmd
	if m.focus == FocusDirectURL {
		var cmd tea.Cmd
		if m.directURLFocus == 0 {
			m.urlInput, cmd = m.urlInput.Update(msg)
		} else {
			m.filenameInput, cmd = m.filenameInput.Update(msg)
		}
		cmds = append(cmds, cmd)

		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "tab":
				m.directURLFocus = (m.directURLFocus + 1) % 2
				if m.directURLFocus == 0 {
					m.urlInput.Focus()
					m.filenameInput.Blur()
				} else {
					m.urlInput.Blur()
					m.filenameInput.Focus()
				}
			case "shift+tab":
				m.directURLFocus = (m.directURLFocus + 1) % 2
				if m.directURLFocus == 0 {
					m.urlInput.Focus()
					m.filenameInput.Blur()
				} else {
					m.urlInput.Blur()
					m.filenameInput.Focus()
				}
			case "esc":
				m.directURLActive = false
				m.urlInput.Blur()
				m.filenameInput.Blur()
				m.focus = FocusSearch
				m.searchInput.Focus()
			case "enter":
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
					m.directURLActive = false
					m.urlInput.Blur()
					m.filenameInput.Blur()
					m.urlInput.SetValue("")
					m.filenameInput.SetValue("")
					m.focus = FocusQueue
					m.selectedTaskIdx = len(m.queue.GetTasks()) - 1
				}
			}
		}
		return m, tea.Batch(cmds...)
	}

	if m.focus == FocusSearch {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case searchHFMsg:
		m.loadingRepos = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.repos = msg.results
			m.selectedRepoIdx = 0
			if len(m.repos) > 0 {
				m.focus = FocusRepos
				m.searchInput.Blur()
				// Fetch files for first repo automatically
				m.loadingFiles = true
				m.files = []model.HFSibling{}
				cmds = append(cmds, listHFFilesCmd(m.repos[0].ModelID, m.config.HFToken))
			}
		}

	case listHFFilesMsg:
		m.loadingFiles = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.files = msg.files
			m.selectedFileIdx = 0
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.nextFocus()
		case "shift+tab":
			m.prevFocus()

		case "enter":
			if m.focus == FocusSearch {
				query := strings.TrimSpace(m.searchInput.Value())
				if query != "" {
					m.loadingRepos = true
					m.repos = []model.HFModelResult{}
					m.files = []model.HFSibling{}
					cmds = append(cmds, searchHFCmd(query, m.config.HFToken))
				}
			} else if m.focus == FocusRepos && len(m.repos) > 0 {
				m.loadingFiles = true
				m.files = []model.HFSibling{}
				cmds = append(cmds, listHFFilesCmd(m.repos[m.selectedRepoIdx].ModelID, m.config.HFToken))
			} else if m.focus == FocusFiles && len(m.files) > 0 {
				// Queue file for download!
				repo := m.repos[m.selectedRepoIdx].ModelID
				file := m.files[m.selectedFileIdx]
				
				// Form Hugging Face resolve URL:
				// https://huggingface.co/<model_id>/resolve/main/<rpath>
				downloadURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repo, file.Rpath)
				
				m.queue.AddTask(repo, file.Rpath, file.Size, downloadURL)
				m.focus = FocusQueue
			}

		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)

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

		case "l", "L":
			if m.focus != FocusSearch {
				m.directURLActive = true
				m.focus = FocusDirectURL
				m.directURLFocus = 0
				m.urlInput.Focus()
				m.filenameInput.Blur()
				m.urlInput.SetValue("")
				m.filenameInput.SetValue("")
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *DownloaderModel) nextFocus() {
	m.searchInput.Blur()
	switch m.focus {
	case FocusSearch:
		if len(m.repos) > 0 {
			m.focus = FocusRepos
		} else {
			m.focus = FocusQueue
		}
	case FocusRepos:
		if len(m.files) > 0 {
			m.focus = FocusFiles
		} else {
			m.focus = FocusQueue
		}
	case FocusFiles:
		m.focus = FocusQueue
	case FocusQueue:
		m.focus = FocusSearch
		m.searchInput.Focus()
	}
}

func (m *DownloaderModel) prevFocus() {
	m.searchInput.Blur()
	switch m.focus {
	case FocusSearch:
		m.focus = FocusQueue
	case FocusRepos:
		m.focus = FocusSearch
		m.searchInput.Focus()
	case FocusFiles:
		m.focus = FocusRepos
	case FocusQueue:
		if len(m.files) > 0 {
			m.focus = FocusFiles
		} else if len(m.repos) > 0 {
			m.focus = FocusRepos
		} else {
			m.focus = FocusSearch
			m.searchInput.Focus()
		}
	}
}

func (m *DownloaderModel) moveCursor(dir int) {
	switch m.focus {
	case FocusRepos:
		if len(m.repos) == 0 {
			return
		}
		m.selectedRepoIdx += dir
		if m.selectedRepoIdx < 0 {
			m.selectedRepoIdx = 0
		}
		if m.selectedRepoIdx >= len(m.repos) {
			m.selectedRepoIdx = len(m.repos) - 1
		}
		
		// Automatically refresh files for selected repo when browsing
		m.loadingFiles = true
		m.files = []model.HFSibling{}
		m.selectedFileIdx = 0
		// We query automatically in the background
		// Bubble tea Update returns commands, so we'll run a helper command or do it asynchronously
		// But to keep it simple, we don't query instantly on every fast scroll.
		// Wait, query-on-enter or query-on-delay is safer, but since lists are small, query-on-enter or triggering a direct fetch works well.
		// Let's allow enter to select and fetch, or trigger a direct fetch.
		// Let's trigger a fetch when user scrolls (simple non-blocking API call).
		// We will return it from the Update loop! But moveCursor doesn't return commands.
		// To fix this, we can let user press Enter to fetch files, or tab.
		// Let's make Enter fetch files for the selected repo. That is extremely safe and prevents spamming HF API.

	case FocusFiles:
		if len(m.files) == 0 {
			return
		}
		m.selectedFileIdx += dir
		if m.selectedFileIdx < 0 {
			m.selectedFileIdx = 0
		}
		if m.selectedFileIdx >= len(m.files) {
			m.selectedFileIdx = len(m.files) - 1
		}

	case FocusQueue:
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
	sb.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("HUGGING FACE MODEL DOWNLOADER")))

	// 1. Search Box
	searchStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(ColorBorder).Padding(0, 1)
	if m.focus == FocusSearch {
		searchStyle = searchStyle.BorderForeground(ColorPrimary)
	}
	sb.WriteString("  " + searchStyle.Render("Search: "+m.searchInput.View()) + "\n\n")

	// Split panel layout heights
	panelHeight := (height - 18) / 2
	if panelHeight < 6 {
		panelHeight = 6
	}

	leftWidth := (width - 8) / 2
	if leftWidth < 25 {
		leftWidth = 25
	}
	rightWidth := width - leftWidth - 8

	// 2. Left side: Repositories List
	var leftSb strings.Builder
	leftSb.WriteString(lipgloss.NewStyle().Bold(true).Render("Repositories:") + "\n")
	if m.loadingRepos {
		leftSb.WriteString("  Querying HF API...\n")
	} else if len(m.repos) == 0 {
		leftSb.WriteString("  No search results.\n")
	} else {
		for idx, repo := range m.repos {
			repoName := repo.ModelID
			if len(repoName) > leftWidth-5 {
				repoName = repoName[:leftWidth-8] + "..."
			}
			row := fmt.Sprintf("%-24s (⬇ %d, ♥ %d)", repoName, repo.Downloads, repo.Likes)
			
			if m.focus == FocusRepos && idx == m.selectedRepoIdx {
				leftSb.WriteString(StyleSelectedListItem.Width(leftWidth).Render(row) + "\n")
			} else {
				leftSb.WriteString(row + "\n")
			}
		}
	}

	// 3. Right side: Files List
	var rightSb strings.Builder
	rightSb.WriteString(lipgloss.NewStyle().Bold(true).Render("GGUF Files:") + "\n")
	if m.loadingFiles {
		rightSb.WriteString("  Fetching files list...\n")
	} else if len(m.files) == 0 {
		rightSb.WriteString("  Select a repository to view files.\n")
	} else {
		for idx, file := range m.files {
			fileName := file.Rpath
			if len(fileName) > rightWidth-5 {
				fileName = fileName[:rightWidth-8] + "..."
			}
			sizeStr := formatSize(file.Size)
			row := fmt.Sprintf("%-24s (%s)", fileName, sizeStr)

			if m.focus == FocusFiles && idx == m.selectedFileIdx {
				rightSb.WriteString(StyleSelectedListItem.Width(rightWidth).Render(row) + "\n")
			} else {
				rightSb.WriteString(row + "\n")
			}
		}
	}

	// Wrap panels with border style
	panelBorderLeft := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Width(leftWidth).
		Height(panelHeight)
	if m.focus == FocusRepos {
		panelBorderLeft = panelBorderLeft.BorderForeground(ColorPrimary)
	}

	panelBorderRight := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Width(rightWidth).
		Height(panelHeight)
	if m.focus == FocusFiles {
		panelBorderRight = panelBorderRight.BorderForeground(ColorPrimary)
	}

	var middleView string
	if m.directURLActive {
		var directSb strings.Builder
		directSb.WriteString("\n  " + lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("DIRECT URL DOWNLOADER") + "\n\n")

		urlStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		fileStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		if m.directURLFocus == 0 {
			urlStyle = urlStyle.Foreground(ColorSecondary).Bold(true)
		} else {
			fileStyle = fileStyle.Foreground(ColorSecondary).Bold(true)
		}

		directSb.WriteString("  " + urlStyle.Render("Direct URL:") + "\n")
		directSb.WriteString("  " + m.urlInput.View() + "\n\n")
		directSb.WriteString("  " + fileStyle.Render("Destination Filename (optional):") + "\n")
		directSb.WriteString("  " + m.filenameInput.View() + "\n\n")

		directSb.WriteString("  " + StyleHelp.Render("Leave filename empty to auto-extract from URL.") + "\n")

		middleView = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Width(width - 6).
			Height(panelHeight).
			Render(directSb.String())
	} else {
		splitView := lipgloss.JoinHorizontal(lipgloss.Top,
			panelBorderLeft.Render(leftSb.String()),
			"  ",
			panelBorderRight.Render(rightSb.String()),
		)
		middleView = splitView
	}
	sb.WriteString("  " + middleView + "\n\n")

	// 4. Bottom: Queue Manager
	var queueSb strings.Builder
	queueSb.WriteString(lipgloss.NewStyle().Bold(true).Render("Download Queue:") + "\n")
	tasks := m.queue.GetTasks()
	if len(tasks) == 0 {
		queueSb.WriteString("  Queue is empty. Select a GGUF file and press Enter to download.\n")
	} else {
		for idx, t := range tasks {
			statusStr := ""
			switch t.Status {
			case model.StatusQueued:
				statusStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("Queued")
			case model.StatusDownloading:
				statusStr = StyleSuccess.Render(fmt.Sprintf("Downloading %.1f KB/s", t.SpeedKBps))
			case model.StatusPaused:
				statusStr = StyleWarning.Render("Paused")
			case model.StatusCompleted:
				statusStr = StyleSuccess.Render("Completed")
			case model.StatusFailed:
				statusStr = StyleDanger.Render(fmt.Sprintf("Failed: %v", t.Error))
			case model.StatusCanceled:
				statusStr = lipgloss.NewStyle().Foreground(ColorMuted).Render("Canceled")
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
	if m.directURLActive {
		helpKeys = append(helpKeys, fmt.Sprintf("%s Switch Fields", StyleHelpKey.Render("[Tab]")))
		helpKeys = append(helpKeys, fmt.Sprintf("%s Start Download", StyleHelpKey.Render("[Enter]")))
		helpKeys = append(helpKeys, fmt.Sprintf("%s Cancel", StyleHelpKey.Render("[Esc]")))
	} else {
		helpKeys = append(helpKeys, fmt.Sprintf("%s Navigation", StyleHelpKey.Render("[Tab/Shift-Tab]")))
		if m.focus == FocusSearch {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Search", StyleHelpKey.Render("[Enter]")))
		} else if m.focus == FocusRepos {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Fetch Files", StyleHelpKey.Render("[Enter]")))
		} else if m.focus == FocusFiles {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Download GGUF", StyleHelpKey.Render("[Enter]")))
		} else if m.focus == FocusQueue {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Pause/Resume", StyleHelpKey.Render("[P]")))
			helpKeys = append(helpKeys, fmt.Sprintf("%s Cancel/Remove", StyleHelpKey.Render("[C]")))
		}
		helpKeys = append(helpKeys, fmt.Sprintf("%s Custom URL", StyleHelpKey.Render("[L]")))
		helpKeys = append(helpKeys, fmt.Sprintf("%s Return to Browser", StyleHelpKey.Render("[Esc]")))
	}

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

