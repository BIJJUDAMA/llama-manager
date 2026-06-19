package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"llama-manager/config"
	"llama-manager/model"
	"llama-manager/runner"
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

	// 3. Add Qwen 2.5 to "Coding" collection
	cfg.AddToCollection("Coding", "models/qwen2.5.gguf")
	bm.rebuildSidebar()
	expectedCollCollapsedCount := 8
	if len(bm.sidebarItems) != expectedCollCollapsedCount {
		t.Errorf("expected %d sidebar items with collapsed collection, got %d", expectedCollCollapsedCount, len(bm.sidebarItems))
	}
	folderItem := bm.sidebarItems[3]
	if folderItem.Type != ItemFolderHeader || folderItem.CollectionName != "Coding" || folderItem.Expanded {
		t.Errorf("expected index 3 to be collapsed Coding folder header, got %+v", folderItem)
	}

	// 4. Expand "Coding" collection
	bm.expandedCollections["Coding"] = true
	bm.rebuildSidebar()
	expectedCollExpandedCount := 9
	if len(bm.sidebarItems) != expectedCollExpandedCount {
		t.Errorf("expected %d sidebar items with expanded collection, got %d", expectedCollExpandedCount, len(bm.sidebarItems))
	}
	nestedItem := bm.sidebarItems[4]
	if nestedItem.Type != ItemModelEntry || nestedItem.ModelPath != "models/qwen2.5.gguf" || nestedItem.CollectionName != "Coding" {
		t.Errorf("expected index 4 to be nested Qwen 2.5 model, got %+v", nestedItem)
	}

	// 5. Test Navigation and selection adjustment
	bm.selected = 5
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



