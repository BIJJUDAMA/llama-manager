package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/BIJJUDAMA/llama-manager/config"
	"github.com/BIJJUDAMA/llama-manager/model"
	"github.com/BIJJUDAMA/llama-manager/runner"
)

func TestBrowserModelInit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llama-ui-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	modelsDir := filepath.Join(tempDir, "models")
	cacheDir := filepath.Join(tempDir, "cache")
	_ = os.MkdirAll(modelsDir, 0755)
	_ = os.MkdirAll(cacheDir, 0755)

	cfg := config.DefaultConfig()
	cfg.Paths.Models = modelsDir
	cfg.Paths.Cache = cacheDir

	srv := runner.NewServerRunner(cacheDir)
	model := NewBrowserModel(cfg, srv)

	if model.loading != true {
		t.Errorf("expected model to start in loading state")
	}

	cmd := model.Init()
	if cmd == nil {
		t.Errorf("expected Init to return a batch command, got nil")
	}
}

func TestBrowserSidebarRebuild(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)

	// Add mock GGUF models
	bm.models = []*model.GGUFMetadata{
		{Name: "Qwen 2.5", FilePath: "models/qwen2.5.gguf"},
		{Name: "Gemma 2", FilePath: "models/gemma2.gguf"},
		{Name: "Llama 3", FilePath: "models/llama3.gguf"},
	}
	bm.filterModels()

	// 1. By default, there should only be "ALL MODELS" and the three models
	expectedInitialCount := 4
	if len(bm.sidebarItems) != expectedInitialCount {
		t.Errorf("expected %d sidebar items initially, got %d", expectedInitialCount, len(bm.sidebarItems))
	}
	if bm.sidebarItems[0].Type != ItemSectionHeader || bm.sidebarItems[0].Label != "ALL MODELS" {
		t.Errorf("expected first item to be ALL MODELS section header, got %+v", bm.sidebarItems[0])
	}

	// 2. Add Gemma 2 to Favorites
	cfg.ToggleFavorite("models/gemma2.gguf")
	bm.rebuildSidebar()
	expectedFavCount := 6
	if len(bm.sidebarItems) != expectedFavCount {
		t.Errorf("expected %d sidebar items after favoriting, got %d: %+v", expectedFavCount, len(bm.sidebarItems), bm.sidebarItems)
	}

	// 3. Test Navigation and selection adjustment
	bm.selected = 2
	bm.adjustSelection()
	if bm.sidebarItems[bm.selected].Type == ItemSectionHeader {
		t.Errorf("adjustSelection failed, selected is still on section header: %d", bm.selected)
	}
}

func TestBrowserBenchmarkTrigger(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)

	// Add mock GGUF models
	bm.models = []*model.GGUFMetadata{
		{Name: "Qwen 2.5", FilePath: "models/qwen2.5.gguf"},
	}
	bm.filterModels()

	// Initial screen mode is ScreenBrowser
	if bm.screenMode != ScreenBrowser {
		t.Errorf("expected screenMode to be ScreenBrowser, got %d", bm.screenMode)
	}

	// Press "b" to trigger benchmark
	var nextModel tea.Model
	var cmd tea.Cmd
	nextModel, cmd = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	updated := nextModel.(*BrowserModel)

	if updated.screenMode != ScreenBenchmarkProgress {
		t.Errorf("expected screenMode to transition to ScreenBenchmarkProgress, got %d", updated.screenMode)
	}
	if updated.benchmarkProgress == nil {
		t.Errorf("expected benchmarkProgress model to be initialized")
	}
	if cmd == nil {
		t.Errorf("expected benchmark launch command to be dispatched")
	}
}

func TestBrowserMonitorTrigger(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)

	// Initial screen mode is ScreenBrowser
	if bm.screenMode != ScreenBrowser {
		t.Errorf("expected screenMode to be ScreenBrowser, got %d", bm.screenMode)
	}

	// Press "m" to trigger monitor
	nextModel, _ := bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	updated := nextModel.(*BrowserModel)

	if updated.screenMode != ScreenServerMonitor {
		t.Errorf("expected screenMode to transition to ScreenServerMonitor, got %d", updated.screenMode)
	}
	if updated.monitorModel == nil {
		t.Errorf("expected monitorModel to be initialized")
	}
}

func TestBrowserSettingsTrigger(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)

	// Initial screen mode is ScreenBrowser
	if bm.screenMode != ScreenBrowser {
		t.Errorf("expected screenMode to be ScreenBrowser, got %d", bm.screenMode)
	}

	// Press "u" to trigger settings
	nextModel, _ := bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	updated := nextModel.(*BrowserModel)

	if updated.screenMode != ScreenSettings {
		t.Errorf("expected screenMode to transition to ScreenSettings, got %d", updated.screenMode)
	}
	if updated.lifecycleModel == nil {
		t.Errorf("expected lifecycleModel to be initialized")
	}
}

func TestBrowserTokenConfiguration(t *testing.T) {
	// Backup user config if exists
	hasUserConfig := false
	if _, err := os.Stat("config.json"); err == nil {
		hasUserConfig = true
		_ = os.Rename("config.json", "config.json.tmp")
	}
	defer func() {
		_ = os.Remove("config.json")
		if hasUserConfig {
			_ = os.Rename("config.json.tmp", "config.json")
		}
	}()

	cfg := config.DefaultConfig()
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)

	// Transition to settings
	m, _ := bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	bm = m.(*BrowserModel)

	// Trigger token editing mode
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	bm = m.(*BrowserModel)

	if !bm.lifecycleModel.tokenEditActive {
		t.Errorf("expected tokenEditActive to be true after pressing 't'")
	}

	// Simulate typing "hf_testtoken123"
	for _, char := range "hf_testtoken123" {
		m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		bm = m.(*BrowserModel)
	}

	// Press enter to save
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm = m.(*BrowserModel)

	if bm.lifecycleModel.tokenEditActive {
		t.Errorf("expected tokenEditActive to be false after pressing Enter")
	}

	if bm.config.HFToken != "hf_testtoken123" {
		t.Errorf("expected HFToken in config to be hf_testtoken123, got %q", bm.config.HFToken)
	}
}

func TestBrowserDownloaderTrigger(t *testing.T) {
	cfg := config.DefaultConfig()
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)

	// Initial screen mode is ScreenBrowser
	if bm.screenMode != ScreenBrowser {
		t.Errorf("expected screenMode to be ScreenBrowser, got %d", bm.screenMode)
	}

	// Press "d" to trigger downloader
	nextModel, _ := bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := nextModel.(*BrowserModel)

	if updated.screenMode != ScreenDownloader {
		t.Errorf("expected screenMode to transition to ScreenDownloader, got %d", updated.screenMode)
	}
	if updated.downloaderModel == nil {
		t.Errorf("expected downloaderModel to be initialized")
	}
}

func TestBrowserDownloaderDirectURL(t *testing.T) {
	// Backup user config if exists
	hasUserConfig := false
	if _, err := os.Stat("config.json"); err == nil {
		hasUserConfig = true
		_ = os.Rename("config.json", "config.json.tmp")
	}
	defer func() {
		_ = os.Remove("config.json")
		if hasUserConfig {
			_ = os.Rename("config.json.tmp", "config.json")
		}
	}()

	cfg := config.DefaultConfig()
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)

	// 1. Transition to Downloader screen
	m, _ := bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	bm = m.(*BrowserModel)

	if bm.screenMode != ScreenDownloader {
		t.Fatalf("expected screenMode to be ScreenDownloader, got %d", bm.screenMode)
	}

	// 2. By default, focus starts on FocusURL in our simplified downloader.
	if bm.downloaderModel.focus != FocusURL {
		t.Fatalf("expected initial focus to be FocusURL, got %d", bm.downloaderModel.focus)
	}

	// 3. Type URL: "http://example.com/models/test-model.gguf"
	for _, char := range "http://example.com/models/test-model.gguf" {
		m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		bm = m.(*BrowserModel)
	}

	// Tab to switch to filename field
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyTab})
	bm = m.(*BrowserModel)

	if bm.downloaderModel.focus != FocusFilename {
		t.Errorf("expected focus to switch to FocusFilename, got focus %d", bm.downloaderModel.focus)
	}

	// Type custom filename: "custom-name.gguf"
	for _, char := range "custom-name.gguf" {
		m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		bm = m.(*BrowserModel)
	}

	// Press enter to queue download
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm = m.(*BrowserModel)

	// Focus should return to FocusURL
	if bm.downloaderModel.focus != FocusURL {
		t.Errorf("expected focus to return to FocusURL after submission, got %d", bm.downloaderModel.focus)
	}

	tasks := bm.downloadQueue.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in queue, got %d", len(tasks))
	}

	task := tasks[0]
	if task.FileName != "custom-name.gguf" {
		t.Errorf("expected task filename to be 'custom-name.gguf', got %q", task.FileName)
	}
	if task.URL != "http://example.com/models/test-model.gguf" {
		t.Errorf("expected task URL to be correct, got %q", task.URL)
	}
}

func TestBrowserCreateCustomProfile(t *testing.T) {
	tempProfilesDir, err := os.MkdirTemp("", "llama-manager-profiles-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempProfilesDir)

	cfg := config.DefaultConfig()
	cfg.Paths.Profiles = tempProfilesDir
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)

	// Set some mock models so we can enter Dashboard
	bm.models = []*model.GGUFMetadata{
		{
			Name:     "Test Model",
			FilePath: "models/test.gguf",
		},
	}
	bm.rebuildSidebar()

	// Select the model entry (index 1 is the model since index 0 is Section Header)
	bm.selected = 1

	// 1. Enter Dashboard by pressing Enter
	m, _ := bm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm = m.(*BrowserModel)

	if bm.screenMode != ScreenDashboard {
		t.Fatalf("expected screenMode to be ScreenDashboard, got %d", bm.screenMode)
	}

	// 2. Press 'P' to open Profile Creator
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("P")})
	bm = m.(*BrowserModel)

	if bm.screenMode != ScreenProfileCreator {
		t.Fatalf("expected screenMode to be ScreenProfileCreator, got %d", bm.screenMode)
	}

	// 3. Type Name: "Custom-Test-Profile"
	for _, char := range "Custom-Test-Profile" {
		m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		bm = m.(*BrowserModel)
	}

	// Tab to Context size
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyTab})
	bm = m.(*BrowserModel)

	// Type context size: "8192"
	for _, char := range "8192" {
		m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		bm = m.(*BrowserModel)
	}

	// Tab to GPU layers
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyTab})
	bm = m.(*BrowserModel)

	// Type GPU layers: "99"
	for _, char := range "99" {
		m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		bm = m.(*BrowserModel)
	}

	// Tab to Port
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyTab})
	bm = m.(*BrowserModel)

	// Type Port: "8085"
	for _, char := range "8085" {
		m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		bm = m.(*BrowserModel)
	}

	// Press Enter to save
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm = m.(*BrowserModel)

	// Should return to Dashboard
	if bm.screenMode != ScreenDashboard {
		t.Errorf("expected to return to ScreenDashboard, got %d", bm.screenMode)
	}

	// Verify profile is created and loaded
	found := false
	for _, p := range bm.profiles {
		if p.Name == "Custom-Test-Profile" {
			found = true
			if p.Context != 8192 {
				t.Errorf("expected context size 8192, got %d", p.Context)
			}
			if p.GPULayers != 99 {
				t.Errorf("expected GPU layers 99, got %d", p.GPULayers)
			}
			if p.Port != 8085 {
				t.Errorf("expected port 8085, got %d", p.Port)
			}
		}
	}
	if !found {
		t.Errorf("created custom profile 'Custom-Test-Profile' was not found in loaded profiles")
	}
}

func TestBrowserOnboardingTour(t *testing.T) {
	// Backup user config if exists
	hasUserConfig := false
	if _, err := os.Stat("config.json"); err == nil {
		hasUserConfig = true
		_ = os.Rename("config.json", "config.json.tmp")
	}
	defer func() {
		_ = os.Remove("config.json")
		if hasUserConfig {
			_ = os.Rename("config.json.tmp", "config.json")
		}
	}()

	cfg := config.DefaultConfig()
	cfg.OnboardingCompleted = false // force onboarding
	srv := runner.NewServerRunner("")
	bm := NewBrowserModel(cfg, srv)
	bm.onboardingActive = true // force onboarding in test environment

	if !bm.onboardingActive {
		t.Errorf("expected onboarding to be active initially")
	}
	if bm.onboardingStep != StepWelcome {
		t.Errorf("expected onboarding to start at StepWelcome, got %d", bm.onboardingStep)
	}

	// Press Enter to advance to next step
	m, _ := bm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm = m.(*BrowserModel)
	if bm.onboardingStep != StepModelSidebar {
		t.Errorf("expected onboarding to advance to StepModelSidebar, got %d", bm.onboardingStep)
	}

	// Press 'b' to go back
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	bm = m.(*BrowserModel)
	if bm.onboardingStep != StepWelcome {
		t.Errorf("expected onboarding to go back to StepWelcome, got %d", bm.onboardingStep)
	}

	// Test that background messages like discoverMsg fall through during onboarding
	bm.onboardingActive = true
	bm.loading = true
	m, _ = bm.Update(discoverMsg{models: []*model.GGUFMetadata{}})
	bm = m.(*BrowserModel)
	if bm.loading {
		t.Errorf("expected discoverMsg to not be swallowed and loading to be false during onboarding")
	}

	// Skip tour
	m, _ = bm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	bm = m.(*BrowserModel)
	if bm.onboardingActive {
		t.Errorf("expected onboarding to be deactivated after pressing Esc")
	}
	if !bm.config.OnboardingCompleted {
		t.Errorf("expected OnboardingCompleted to be set to true in config")
	}
}





