package runner

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BIJJUDAMA/llama-manager/hardware"
)

func TestMatchAsset(t *testing.T) {
	release := &GithubRelease{
		TagName: "b3310",
		Name:    "llama.cpp b3310",
		Assets: []ReleaseAsset{
			{Name: "llama-b3310-bin-win-cu12.2.0-x64.zip", BrowserDownloadURL: "http://win-cuda-12.zip", Size: 100},
			{Name: "cudart-llama-bin-win-cuda-12.4-x64.zip", BrowserDownloadURL: "http://win-cudart-12.zip", Size: 50},
			{Name: "llama-b3310-bin-win-llvm-x64.zip", BrowserDownloadURL: "http://win-llvm.zip", Size: 80},
			{Name: "llama-b3310-bin-ubuntu-x64.zip", BrowserDownloadURL: "http://linux.zip", Size: 70},
			{Name: "llama-b3310-bin-macos-arm64.zip", BrowserDownloadURL: "http://mac-arm64.zip", Size: 60},
			{Name: "unrelated.txt", BrowserDownloadURL: "http://unrelated.txt", Size: 10},
		},
	}

	// 1. Windows with CUDA GPU
	specsWinCUDA := &hardware.HardwareSpecs{
		OS: "Windows",
		GPU: hardware.GPUSpecs{
			Type: "CUDA",
		},
	}
	asset, cudart, err := MatchAsset(release, specsWinCUDA)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(asset.Name, "win-cu12.2.0") {
		t.Errorf("expected win-cu12.2.0 asset, got %s", asset.Name)
	}
	if cudart == nil || !strings.Contains(cudart.Name, "cudart-llama-bin-win-cuda-12.4-x64") {
		t.Errorf("expected cudart asset matching CUDA DLLs, got %+v", cudart)
	}

	// 2. Windows with CPU only
	specsWinCPU := &hardware.HardwareSpecs{
		OS: "Windows",
		GPU: hardware.GPUSpecs{
			Type: "CPU",
		},
	}
	asset, cudart, err = MatchAsset(release, specsWinCPU)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(asset.Name, "win-llvm") {
		t.Errorf("expected win-llvm asset, got %s", asset.Name)
	}
	if cudart != nil {
		t.Errorf("expected no cudart asset for CPU, got %s", cudart.Name)
	}

	// 3. macOS
	specsMac := &hardware.HardwareSpecs{
		OS: "darwin",
		GPU: hardware.GPUSpecs{
			Type: "Metal",
		},
	}
	asset, cudart, err = MatchAsset(release, specsMac)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(asset.Name, "macos-arm64") {
		t.Errorf("expected macos-arm64 asset, got %s", asset.Name)
	}
	if cudart != nil {
		t.Errorf("expected no cudart asset for Metal, got %s", cudart.Name)
	}
}

func TestExtractAndFlattenZip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llama-manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	zw := zip.NewWriter(zipFile)

	// Create files inside a single root directory in zip
	subFolder := "llama-b3310-bin-win-llvm-x64/"
	filesToCreate := map[string]string{
		subFolder + "llama-server.exe": "server-binary-content",
		subFolder + "llama-cli.exe":    "cli-binary-content",
		subFolder + "README.md":        "readme-content",
	}

	for name, content := range filesToCreate {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry: %v", err)
		}
		_, _ = w.Write([]byte(content))
	}
	_ = zw.Close()
	_ = zipFile.Close()

	// Extract
	destDir := filepath.Join(tempDir, "extracted")
	err = ExtractArchive(zipPath, destDir)
	if err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	// Verify that files are flattened directly under destDir
	expectedFiles := []string{"llama-server.exe", "llama-cli.exe", "README.md"}
	for _, f := range expectedFiles {
		path := filepath.Join(destDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to be extracted and flattened, but not found", f)
		}
	}

	// Check content of one of the files
	contentBytes, err := os.ReadFile(filepath.Join(destDir, "llama-server.exe"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(contentBytes) != "server-binary-content" {
		t.Errorf("incorrect file content: got %q, expected %q", string(contentBytes), "server-binary-content")
	}
}

func TestBackupAndRollback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llama-manager-backup-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "llama.cpp")
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("original-content"), 0644)

	backupDir := filepath.Join(tempDir, "llama.cpp.backup")

	// Test Backup
	err = CreateBackup(srcDir, backupDir)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup files exist
	if _, err := os.Stat(filepath.Join(backupDir, "test.txt")); os.IsNotExist(err) {
		t.Errorf("backup did not copy test.txt")
	}

	// Test Rollback
	// Modify src first
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("modified-content"), 0644)

	err = RollbackBackup(backupDir, srcDir)
	if err != nil {
		t.Fatalf("RollbackBackup failed: %v", err)
	}

	// Verify rollback content is original
	content, err := os.ReadFile(filepath.Join(srcDir, "test.txt"))
	if err != nil {
		t.Fatalf("failed to read text file: %v", err)
	}
	if string(content) != "original-content" {
		t.Errorf("expected rolled back content %q, got %q", "original-content", string(content))
	}
}

func TestQueryLocalVersionMissing(t *testing.T) {
	_, _, _, err := QueryLocalVersion("missing-directory-path")
	if err == nil {
		t.Errorf("expected error for missing directory path, got nil")
	}
}

// Helper functions for copying folders under tests
// (Already defined at package level in updater.go)
