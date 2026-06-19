package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"llama-manager/config"
	"llama-manager/hardware"
	"llama-manager/runner"
)

type LifecycleState int

const (
	StateIdle LifecycleState = iota
	StateChecking
	StateNoUpdate
	StateUpdateAvailable
	StateDownloading
	StateExtracting
	StateVerifying
	StateUpdateSuccess
	StateRollingBack
	StateRollbackSuccess
	StateError
)

type updateMsg struct {
	state    LifecycleState
	progress float64
	msg      string
	err      error
	release  *runner.GithubRelease
	ch       chan updateMsg
}

type LifecycleModel struct {
	srvRunner                     *runner.ServerRunner
	config                        *config.Config
	specs                         *hardware.HardwareSpecs
	state                         LifecycleState
	localVersion                  string
	localCommit                   string
	localBuildInfo                string
	latestTagName                 string
	latestRelease                 *runner.GithubRelease
	matchedAsset                  *runner.ReleaseAsset
	downloadProgress              float64
	actionMsg                     string
	err                           error
	hasBackup                     bool
	width, height                 int
}

func NewLifecycleModel(cfg *config.Config, srv *runner.ServerRunner) *LifecycleModel {
	specs, _ := hardware.DetectHardware()
	if specs == nil {
		specs = &hardware.HardwareSpecs{OS: runtime.GOOS}
	}

	m := &LifecycleModel{
		srvRunner: srv,
		config:    cfg,
		specs:     specs,
		state:     StateIdle,
	}
	m.RefreshLocalVersion()
	m.RefreshBackupStatus()
	return m
}

func (m *LifecycleModel) RefreshLocalVersion() {
	version, commit, buildInfo, err := runner.QueryLocalVersion(m.config.Paths.LlamaCPP)
	if err == nil {
		m.localVersion = version
		m.localCommit = commit
		m.localBuildInfo = buildInfo
	} else {
		m.localVersion = "Not Installed"
		m.localCommit = "N/A"
		m.localBuildInfo = "N/A"
	}
}

func (m *LifecycleModel) RefreshBackupStatus() {
	backupDir := m.config.Paths.LlamaCPP + ".backup"
	if _, err := os.Stat(backupDir); err == nil {
		m.hasBackup = true
	} else {
		m.hasBackup = false
	}
}

func (m *LifecycleModel) StartCheckOnly() tea.Cmd {
	ch := make(chan updateMsg)

	go func() {
		ch <- updateMsg{state: StateChecking, msg: "Checking for updates...", ch: ch}
		release, err := runner.CheckLatestRelease()
		if err != nil {
			ch <- updateMsg{state: StateError, err: fmt.Errorf("failed to check for updates: %w", err), ch: ch}
			return
		}

		localV, _, _, _ := runner.QueryLocalVersion(m.config.Paths.LlamaCPP)
		state := StateUpdateAvailable
		cleanLocal := strings.TrimPrefix(strings.ToLower(localV), "b")
		cleanLatest := strings.TrimPrefix(strings.ToLower(release.TagName), "b")
		if cleanLocal == cleanLatest && cleanLocal != "unknown" && cleanLocal != "not installed" {
			state = StateNoUpdate
		}

		ch <- updateMsg{
			state:   state,
			msg:     fmt.Sprintf("Latest available release: %s", release.TagName),
			release: release,
			ch:      ch,
		}
	}()

	return m.ReadUpdateChan(ch)
}

func (m *LifecycleModel) StartUpdate() tea.Cmd {
	ch := make(chan updateMsg)

	go func() {
		instances := m.srvRunner.GetAllInstances()
		if len(instances) > 0 {
			ch <- updateMsg{state: StateError, err: fmt.Errorf("cannot update: active server instances are running. Please stop all servers first."), ch: ch}
			return
		}

		var release *runner.GithubRelease
		var err error
		if m.latestRelease != nil {
			release = m.latestRelease
		} else {
			ch <- updateMsg{state: StateChecking, msg: "Checking latest release on GitHub...", ch: ch}
			release, err = runner.CheckLatestRelease()
			if err != nil {
				ch <- updateMsg{state: StateError, err: fmt.Errorf("failed to check release: %w", err), ch: ch}
				return
			}
		}

		asset, err := runner.MatchAsset(release, m.specs)
		if err != nil {
			ch <- updateMsg{state: StateError, err: fmt.Errorf("failed to match release asset: %w", err), ch: ch}
			return
		}

		ch <- updateMsg{state: StateDownloading, progress: 0.0, msg: fmt.Sprintf("Downloading %s...", asset.Name), ch: ch}

		destFile := filepath.Join(m.config.Paths.Downloads, asset.Name)
		progressChan := make(chan float64, 5)

		downloadErrChan := make(chan error, 1)
		go func() {
			downloadErrChan <- runner.DownloadRelease(asset.BrowserDownloadURL, destFile, progressChan)
		}()

		for p := range progressChan {
			ch <- updateMsg{state: StateDownloading, progress: p, msg: fmt.Sprintf("Downloading %s (%.1f%%)...", asset.Name, p*100.0), ch: ch}
		}

		if derr := <-downloadErrChan; derr != nil {
			ch <- updateMsg{state: StateError, err: fmt.Errorf("failed to download release: %w", derr), ch: ch}
			return
		}

		ch <- updateMsg{state: StateExtracting, msg: "Creating backup of existing llama.cpp...", ch: ch}
		backupDir := m.config.Paths.LlamaCPP + ".backup"

		if _, err := os.Stat(m.config.Paths.LlamaCPP); err == nil {
			err = runner.CreateBackup(m.config.Paths.LlamaCPP, backupDir)
			if err != nil {
				ch <- updateMsg{state: StateError, err: fmt.Errorf("failed to create backup: %w", err), ch: ch}
				return
			}
		}

		ch <- updateMsg{state: StateExtracting, msg: "Extracting updated binaries...", ch: ch}
		_ = os.MkdirAll(m.config.Paths.LlamaCPP, 0755)
		err = runner.ExtractArchive(destFile, m.config.Paths.LlamaCPP)
		if err != nil {
			_ = runner.RollbackBackup(backupDir, m.config.Paths.LlamaCPP)
			ch <- updateMsg{state: StateError, err: fmt.Errorf("extraction failed (rolled back): %w", err), ch: ch}
			return
		}

		_ = os.Remove(destFile)

		ch <- updateMsg{state: StateVerifying, msg: "Verifying installation...", ch: ch}
		version, commit, buildInfo, err := runner.QueryLocalVersion(m.config.Paths.LlamaCPP)
		if err != nil {
			_ = runner.RollbackBackup(backupDir, m.config.Paths.LlamaCPP)
			ch <- updateMsg{state: StateError, err: fmt.Errorf("verification failed (rolled back): %w", err), ch: ch}
			return
		}

		ch <- updateMsg{
			state: StateUpdateSuccess,
			msg:   fmt.Sprintf("Update successful! Version: %s, Commit: %s (%s)", version, commit, buildInfo),
			ch:    ch,
		}
	}()

	return m.ReadUpdateChan(ch)
}

func (m *LifecycleModel) StartRollback() tea.Cmd {
	ch := make(chan updateMsg)

	go func() {
		instances := m.srvRunner.GetAllInstances()
		if len(instances) > 0 {
			ch <- updateMsg{state: StateError, err: fmt.Errorf("cannot rollback: active server instances are running. Please stop all servers first."), ch: ch}
			return
		}

		ch <- updateMsg{state: StateRollingBack, msg: "Restoring backup version...", ch: ch}
		backupDir := m.config.Paths.LlamaCPP + ".backup"

		err := runner.RollbackBackup(backupDir, m.config.Paths.LlamaCPP)
		if err != nil {
			ch <- updateMsg{state: StateError, err: fmt.Errorf("rollback failed: %w", err), ch: ch}
			return
		}

		ch <- updateMsg{state: StateRollbackSuccess, msg: "Rollback completed successfully!", ch: ch}
	}()

	return m.ReadUpdateChan(ch)
}

func (m *LifecycleModel) ReadUpdateChan(ch chan updateMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *LifecycleModel) Update(msg tea.Msg) (*LifecycleModel, tea.Cmd) {
	switch msg := msg.(type) {
	case updateMsg:
		m.state = msg.state
		if msg.err != nil {
			m.err = msg.err
			m.actionMsg = ""
		} else {
			m.err = nil
			m.actionMsg = msg.msg
		}

		if msg.progress > 0 {
			m.downloadProgress = msg.progress
		}

		if msg.release != nil {
			m.latestRelease = msg.release
			m.latestTagName = msg.release.TagName
		}

		// Check if we are done or failed to refresh info
		if m.state == StateUpdateSuccess || m.state == StateRollbackSuccess || m.state == StateError {
			m.RefreshLocalVersion()
			m.RefreshBackupStatus()
		}

		if m.state != StateUpdateSuccess && m.state != StateRollbackSuccess && m.state != StateError && m.state != StateNoUpdate && m.state != StateUpdateAvailable {
			return m, m.ReadUpdateChan(msg.ch)
		}
	}
	return m, nil
}

func (m *LifecycleModel) View(width int, height int) string {
	m.width = width
	m.height = height

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("LLAMA.CPP LIFECYCLE MANAGEMENT")))

	// Current installation specs
	sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Local Installation:") + "\n")
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Folder Path:", m.config.Paths.LlamaCPP))

	localVerStr := m.localVersion
	if localVerStr != "Not Installed" && localVerStr != "Unknown" {
		localVerStr = StyleSuccess.Render(localVerStr)
	} else if localVerStr == "Not Installed" {
		localVerStr = StyleDanger.Render(localVerStr)
	}
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Version Tag:", localVerStr))
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Commit Hash:", m.localCommit))
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Compiler/Build:", m.localBuildInfo))

	backupStr := lipgloss.NewStyle().Foreground(ColorMuted).Render("Not Available")
	if m.hasBackup {
		backupStr = StyleSuccess.Render("Available (llama.cpp.backup/)")
	}
	sb.WriteString(fmt.Sprintf("    %-20s %s\n\n", "Local Backup:", backupStr))

	// Hardware Spec Match Info
	sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Target Platform Match:") + "\n")
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Operating System:", m.specs.OS))
	sb.WriteString(fmt.Sprintf("    %-20s %s\n\n", "GPU Accelerator:", m.specs.GPU.Type))

	// Update checker info
	sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("GitHub Release Status:") + "\n")
	if m.latestTagName != "" {
		sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Available:", lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(m.latestTagName)))
	} else {
		sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Available:", "Not checked yet"))
	}

	// Dynamic status display
	sb.WriteString("\n  " + lipgloss.NewStyle().Bold(true).Render("Status:") + "\n")
	statusText := "Idle"
	switch m.state {
	case StateChecking:
		statusText = "Checking latest release..."
	case StateNoUpdate:
		statusText = StyleSuccess.Render("Inference runtime is up-to-date.")
	case StateUpdateAvailable:
		statusText = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render("Update available!")
	case StateDownloading:
		statusText = fmt.Sprintf("Downloading release: %s", m.actionMsg)
	case StateExtracting:
		statusText = fmt.Sprintf("Extracting files: %s", m.actionMsg)
	case StateVerifying:
		statusText = "Verifying installation integrity..."
	case StateUpdateSuccess:
		statusText = StyleSuccess.Render("Update completed successfully!")
	case StateRollingBack:
		statusText = "Rolling back to previous backup..."
	case StateRollbackSuccess:
		statusText = StyleSuccess.Render("Rollback completed successfully!")
	case StateError:
		statusText = StyleDanger.Render(fmt.Sprintf("Error: %v", m.err))
	}
	sb.WriteString("    " + statusText + "\n\n")

	// Render Progress Bar if downloading
	if m.state == StateDownloading {
		sb.WriteString("    " + renderProgressBar(width-8, m.downloadProgress) + "\n\n")
	}

	// Instructions help footer
	var helpKeys []string
	if m.state != StateDownloading && m.state != StateExtracting && m.state != StateVerifying && m.state != StateRollingBack {
		helpKeys = append(helpKeys, fmt.Sprintf("%s Check Updates", StyleHelpKey.Render("[C]")))
		if m.latestTagName != "" {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Apply Update", StyleHelpKey.Render("[U]")))
		}
		if m.hasBackup {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Rollback to Backup", StyleHelpKey.Render("[R]")))
		}
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

func renderProgressBar(width int, fraction float64) string {
	barWidth := width - 10
	if barWidth < 10 {
		barWidth = 10
	}
	filledWidth := int(float64(barWidth) * fraction)
	if filledWidth > barWidth {
		filledWidth = barWidth
	}
	emptyWidth := barWidth - filledWidth

	filled := lipgloss.NewStyle().Foreground(ColorSecondary).Render(strings.Repeat("█", filledWidth))
	empty := lipgloss.NewStyle().Foreground(ColorMuted).Render(strings.Repeat("░", emptyWidth))
	percent := fmt.Sprintf(" %3.0f%%", fraction*100.0)

	return filled + empty + percent
}
