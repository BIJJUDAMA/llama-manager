package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/BIJJUDAMA/llama-manager/config"
	"github.com/BIJJUDAMA/llama-manager/hardware"
	"github.com/BIJJUDAMA/llama-manager/runner"
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

type appCheckMsg struct {
	latestTag string
	err       error
}

type LifecycleModel struct {
	srvRunner        runner.ModelRuntime
	config           *config.Config
	specs            *hardware.HardwareSpecs
	state            LifecycleState
	localVersion     string
	localCommit      string
	localBuildInfo   string
	latestTagName    string
	latestRelease    *runner.GithubRelease
	matchedAsset     *runner.ReleaseAsset
	matchedCudart    *runner.ReleaseAsset
	downloadProgress float64
	actionMsg        string
	err              error
	hasBackup        bool
	width, height    int
	tokenInput       textinput.Model
	tokenEditActive  bool
	// App self-update fields
	appVersion       string
	appLatestTag     string
	appCheckErr      error
	appChecking      bool
	appUpdating      bool
	appUpdateErr     error
	appUpdateSuccess bool
	appUpdateMsg     string
}

func resolveAppVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		v := info.Main.Version
		if v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

func NewLifecycleModel(cfg *config.Config, srv runner.ModelRuntime) *LifecycleModel {
	specs, _ := hardware.DetectHardware()
	if specs == nil {
		specs = &hardware.HardwareSpecs{OS: runtime.GOOS}
	}

	ti := textinput.New()
	ti.Placeholder = "Enter HF_TOKEN (hf_...)"
	ti.CharLimit = 100
	ti.Width = 40
	ti.EchoMode = textinput.EchoPassword

	m := &LifecycleModel{
		srvRunner:       srv,
		config:          cfg,
		specs:           specs,
		state:           StateIdle,
		tokenInput:      ti,
		tokenEditActive: false,
		appVersion:      resolveAppVersion(),
	}
	m.RefreshLocalVersion()
	m.RefreshBackupStatus()
	return m
}

// StartAppCheck queries GitHub for the latest llama-manager release tag.
func (m *LifecycleModel) StartAppCheck() tea.Cmd {
	m.appChecking = true
	m.appCheckErr = nil
	return func() tea.Msg {
		rel, err := runner.CheckAppRelease()
		if err != nil {
			return appCheckMsg{err: err}
		}
		return appCheckMsg{latestTag: rel.TagName}
	}
}

type appUpdateMsg struct {
	err error
	msg string
}

// StartAppUpdate runs go install to update the app.
func (m *LifecycleModel) StartAppUpdate() tea.Cmd {
	m.appUpdating = true
	m.appUpdateErr = nil
	m.appUpdateSuccess = false
	return func() tea.Msg {
		cmd := exec.Command("go", "install", "github.com/BIJJUDAMA/llama-manager/cmd/llmgr@latest")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return appUpdateMsg{err: fmt.Errorf("failed to run go install: %w (output: %s)", err, string(output))}
		}
		return appUpdateMsg{msg: "Update successful! Please restart the application."}
	}
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

		mainAsset, cudartAsset, err := runner.MatchAsset(release, m.specs)
		if err != nil {
			ch <- updateMsg{state: StateError, err: fmt.Errorf("failed to match release asset: %w", err), ch: ch}
			return
		}

		var mainScale float64 = 1.0
		if cudartAsset != nil {
			mainScale = 0.7
		}

		ch <- updateMsg{state: StateDownloading, progress: 0.0, msg: fmt.Sprintf("Downloading %s...", mainAsset.Name), ch: ch}

		destFile := filepath.Join(m.config.Paths.Downloads, mainAsset.Name)
		progressChan := make(chan float64, 5)

		downloadErrChan := make(chan error, 1)
		go func() {
			downloadErrChan <- runner.DownloadRelease(mainAsset.BrowserDownloadURL, destFile, progressChan)
		}()

		for p := range progressChan {
			ch <- updateMsg{state: StateDownloading, progress: p * mainScale, msg: fmt.Sprintf("Downloading %s (%.1f%%)...", mainAsset.Name, p*100.0), ch: ch}
		}

		if derr := <-downloadErrChan; derr != nil {
			ch <- updateMsg{state: StateError, err: fmt.Errorf("failed to download release: %w", derr), ch: ch}
			return
		}

		var destCudartFile string
		if cudartAsset != nil {
			ch <- updateMsg{state: StateDownloading, progress: 0.7, msg: fmt.Sprintf("Downloading %s...", cudartAsset.Name), ch: ch}
			destCudartFile = filepath.Join(m.config.Paths.Downloads, cudartAsset.Name)
			cudartProgressChan := make(chan float64, 5)

			cudartDownloadErrChan := make(chan error, 1)
			go func() {
				cudartDownloadErrChan <- runner.DownloadRelease(cudartAsset.BrowserDownloadURL, destCudartFile, cudartProgressChan)
			}()

			for p := range cudartProgressChan {
				combinedProgress := 0.7 + (p * 0.3)
				ch <- updateMsg{state: StateDownloading, progress: combinedProgress, msg: fmt.Sprintf("Downloading %s (%.1f%%)...", cudartAsset.Name, p*100.0), ch: ch}
			}

			if derr := <-cudartDownloadErrChan; derr != nil {
				ch <- updateMsg{state: StateError, err: fmt.Errorf("failed to download cudart release: %w", derr), ch: ch}
				return
			}
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

		if destCudartFile != "" {
			ch <- updateMsg{state: StateExtracting, msg: "Extracting CUDA runtime DLLs...", ch: ch}
			err = runner.ExtractArchive(destCudartFile, m.config.Paths.LlamaCPP)
			if err != nil {
				_ = runner.RollbackBackup(backupDir, m.config.Paths.LlamaCPP)
				ch <- updateMsg{state: StateError, err: fmt.Errorf("CUDA DLLs extraction failed (rolled back): %w", err), ch: ch}
				return
			}
			_ = os.Remove(destCudartFile)
		}

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
	if m.tokenEditActive {
		// Handle ctrl+v before delegating to the textinput Update
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "ctrl+v" {
			pasteFromClipboard(&m.tokenInput)
			return m, nil
		}

		var cmd tea.Cmd
		m.tokenInput, cmd = m.tokenInput.Update(msg)

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.config.HFToken = strings.TrimSpace(m.tokenInput.Value())
				_ = m.config.Save()
				m.tokenInput.Blur()
				m.tokenEditActive = false
			case "esc":
				m.tokenInput.Blur()
				m.tokenEditActive = false
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "t", "T":
			m.tokenEditActive = true
			m.tokenInput.Focus()
			m.tokenInput.SetValue(m.config.HFToken)
			return m, nil
		}

	case appCheckMsg:
		m.appChecking = false
		if msg.err != nil {
			m.appCheckErr = msg.err
		} else {
			m.appCheckErr = nil
			m.appLatestTag = msg.latestTag
		}

	case appUpdateMsg:
		m.appUpdating = false
		if msg.err != nil {
			m.appUpdateErr = msg.err
			m.appUpdateSuccess = false
		} else {
			m.appUpdateErr = nil
			m.appUpdateSuccess = true
			m.appUpdateMsg = msg.msg
			m.appVersion = m.appLatestTag
		}

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

func maskToken(token string) string {
	if token == "" {
		return lipgloss.NewStyle().Foreground(ColorMuted).Render("Not Configured")
	}
	if len(token) <= 10 {
		return "********"
	}
	return token[:5] + "..." + token[len(token)-5:]
}

func (m *LifecycleModel) View(width int, height int) string {
	m.width = width
	m.height = height

	if m.tokenEditActive {
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("CONFIGURE HUGGING FACE TOKEN")))
		sb.WriteString("  Please enter or paste your Hugging Face API token (HF_TOKEN).\n")
		sb.WriteString("  This token is used for downloading gated/private models and avoiding API limits.\n\n")
		sb.WriteString("  " + m.tokenInput.View() + "\n\n")
		sb.WriteString("  " + StyleHelpKey.Render("[Enter]") + " Save Token  " + StyleHelpKey.Render("[Esc]") + " Cancel / Exit\n")

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

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("SETTINGS")))

	// ── App Version ──────────────────────────────────────────────────────────
	sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Llama Manager:") + "\n")
	appVerStr := lipgloss.NewStyle().Foreground(ColorWhite).Render(m.appVersion)
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Installed Version:", appVerStr))
	if m.appChecking {
		sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Release:", lipgloss.NewStyle().Foreground(ColorMuted).Render("Checking...")))
	} else if m.appCheckErr != nil {
		sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Release:", StyleDanger.Render("Check failed")))
	} else if m.appLatestTag != "" {
		if m.appUpdateSuccess {
			sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Release:", StyleSuccess.Render(m.appLatestTag+" (up-to-date)")))
			sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Status:", StyleSuccess.Render(m.appUpdateMsg)))
		} else if m.appLatestTag == m.appVersion {
			sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Release:", StyleSuccess.Render(m.appLatestTag+" (up-to-date)")))
		} else {
			sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Release:", lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(m.appLatestTag+" — update available")))
			if m.appUpdating {
				sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Status:", lipgloss.NewStyle().Foreground(ColorAccent).Render("Installing update...")))
			} else if m.appUpdateErr != nil {
				sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Status:", StyleDanger.Render(fmt.Sprintf("Update failed: %v", m.appUpdateErr))))
				sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Press:", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("[A] to retry update")))
			} else {
				sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Press:", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("[A] to install update")))
			}
		}
	} else {
		sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Release:", lipgloss.NewStyle().Foreground(ColorMuted).Render("Not checked  [V] to check")))
	}
	sb.WriteString("\n")

	// ── llama.cpp Installation ───────────────────────────────────────────────
	sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Inference Runtime (llama.cpp):") + "\n")
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
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Local Backup:", backupStr))

	if m.latestTagName != "" {
		sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Release:", lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(m.latestTagName)))
	} else {
		sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Latest Release:", lipgloss.NewStyle().Foreground(ColorMuted).Render("Not checked  [C] to check")))
	}
	sb.WriteString("\n")

	// ── Preferences ──────────────────────────────────────────────────────────
	sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Preferences:") + "\n")
	themeStr := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(strings.Title(m.config.Theme))
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Color Theme:", themeStr))
	tokenStr := maskToken(m.config.HFToken)
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "HF Token:", tokenStr))
	onboardStr := lipgloss.NewStyle().Foreground(ColorMuted).Render("Completed")
	if !m.config.OnboardingCompleted {
		onboardStr = lipgloss.NewStyle().Foreground(ColorAccent).Render("Not completed")
	}
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Onboarding Tour:", onboardStr))
	sb.WriteString("\n")

	// ── Hardware ─────────────────────────────────────────────────────────────
	sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Platform:") + "\n")
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "Operating System:", m.specs.OS))
	sb.WriteString(fmt.Sprintf("    %-20s %s\n", "GPU Accelerator:", m.specs.GPU.Type))
	sb.WriteString("\n")

	// ── Runtime status ───────────────────────────────────────────────────────
	if m.state != StateIdle {
		sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Status:") + "\n")
		statusText := ""
		switch m.state {
		case StateChecking:
			statusText = "Checking latest llama.cpp release..."
		case StateNoUpdate:
			statusText = StyleSuccess.Render("Inference runtime is up-to-date.")
		case StateUpdateAvailable:
			statusText = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render("llama.cpp update available — press [U] to apply.")
		case StateDownloading:
			statusText = fmt.Sprintf("Downloading: %s", m.actionMsg)
		case StateExtracting:
			statusText = fmt.Sprintf("Extracting: %s", m.actionMsg)
		case StateVerifying:
			statusText = "Verifying installation integrity..."
		case StateUpdateSuccess:
			statusText = StyleSuccess.Render("llama.cpp update completed successfully.")
		case StateRollingBack:
			statusText = "Rolling back to previous backup..."
		case StateRollbackSuccess:
			statusText = StyleSuccess.Render("Rollback completed successfully.")
		case StateError:
			statusText = StyleDanger.Render(fmt.Sprintf("Error: %v", m.err))
		}
		if statusText != "" {
			sb.WriteString("    " + statusText + "\n")
		}
		if m.state == StateDownloading {
			sb.WriteString("    " + renderProgressBar(width-8, m.downloadProgress) + "\n")
		}
		sb.WriteString("\n")
	}

	// ── Help footer ───────────────────────────────────────────────────────────
	var helpKeys []string
	if m.state != StateDownloading && m.state != StateExtracting && m.state != StateVerifying && m.state != StateRollingBack {
		helpKeys = append(helpKeys, fmt.Sprintf("%s Check llama.cpp", StyleHelpKey.Render("[C]")))
		helpKeys = append(helpKeys, fmt.Sprintf("%s Check App", StyleHelpKey.Render("[V]")))
		if m.appLatestTag != "" && m.appLatestTag != m.appVersion && !m.appUpdating {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Update App", StyleHelpKey.Render("[A]")))
		}
		helpKeys = append(helpKeys, fmt.Sprintf("%s Cycle Theme", StyleHelpKey.Render("[O]")))
		if m.latestTagName != "" {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Apply llama.cpp Update", StyleHelpKey.Render("[U]")))
		}
		if m.hasBackup {
			helpKeys = append(helpKeys, fmt.Sprintf("%s Rollback", StyleHelpKey.Render("[R]")))
		}
		helpKeys = append(helpKeys, fmt.Sprintf("%s HF Token", StyleHelpKey.Render("[T]")))
		helpKeys = append(helpKeys, fmt.Sprintf("%s Reset Tour", StyleHelpKey.Render("[N]")))
	}
	helpKeys = append(helpKeys, fmt.Sprintf("%s Back", StyleHelpKey.Render("[Esc]")))

	sb.WriteString("  " + strings.Join(helpKeys, " │ ") + "\n")

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
