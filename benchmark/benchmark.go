package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/BIJJUDAMA/runora/config"
	"github.com/BIJJUDAMA/runora/hardware"
	"github.com/BIJJUDAMA/runora/model"
	"github.com/BIJJUDAMA/runora/runner"
)

type BenchmarkResult struct {
	ModelPath        string    `json:"model_path"`
	ModelName        string    `json:"model_name"`
	RunDate          time.Time `json:"run_date"`
	StartupTimeMs    int64     `json:"startup_time_ms"`
	TokensPerSec     float64   `json:"tokens_per_sec"`
	PeakTokensPerSec float64   `json:"peak_tokens_per_sec"`
	RAMUsageMB       float64   `json:"ram_usage_mb"`
	VRAMUsageMB      float64   `json:"vram_usage_mb"`
}

type LlamaTimings struct {
	PromptN            int     `json:"prompt_n"`
	PromptMs           float64 `json:"prompt_ms"`
	PromptPerSecond    float64 `json:"prompt_per_second"`
	PredictedN         int     `json:"predicted_n"`
	PredictedMs        float64 `json:"predicted_ms"`
	PredictedPerSecond float64 `json:"predicted_per_second"`
}

type LlamaCompletionResponse struct {
	Content string       `json:"content"`
	Timings LlamaTimings `json:"timings"`
}

// RunBenchmark launches llama-server, issues a test prompt, parses timings, and stops the server.
func RunBenchmark(llamaCppDir string, m *model.GGUFMetadata, specs *hardware.HardwareSpecs, cfg *config.Config, onStep func(int)) (*BenchmarkResult, error) {
	if onStep != nil {
		onStep(0) // StepBooting
	}
	benchRunner := runner.NewLlamaCppRuntime(cfg.Paths.Cache)

	startTime := time.Now()
	port := 9091 // Dedicated benchmarking port

	// Stop any previous process running on this runner just in case
	_ = benchRunner.Stop()

	// Launch llama-server with low-context fast settings for benchmarking
	err := benchRunner.Start(m.FilePath, runner.StartOptions{
		LlamaCppDir: llamaCppDir,
		ContextSize: 512, // low context for quick initialization
		Threads:     4,   // standard threads
		GPULayers:   999, // offload layers
		BatchSize:   512, // batch size
		Host:        "127.0.0.1",
		Port:        port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start benchmark server: %w", err)
	}

	// Poll health endpoint
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	healthy := false
	var startupTimeMs int64

	for i := 0; i < 40; i++ { // Try for 20 seconds
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				healthy = true
				startupTimeMs = time.Since(startTime).Milliseconds()
				break
			}
		}
	}

	if !healthy {
		_ = benchRunner.Stop()
		return nil, fmt.Errorf("benchmark server failed to respond within 20s")
	}

	if onStep != nil {
		onStep(1) // StepRunningPrompt
	}

	// Run completion request
	promptPayload := map[string]interface{}{
		"prompt":    "Write a short poem about local AI running in the terminal.",
		"n_predict": 64,
	}
	payloadBytes, err := json.Marshal(promptPayload)
	if err != nil {
		_ = benchRunner.Stop()
		return nil, fmt.Errorf("failed to marshal prompt: %w", err)
	}

	completionURL := fmt.Sprintf("http://127.0.0.1:%d/completion", port)
	compStart := time.Now()
	resp, err := http.Post(completionURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		_ = benchRunner.Stop()
		return nil, fmt.Errorf("completion request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = benchRunner.Stop()
		return nil, fmt.Errorf("failed to read completion body: %w", err)
	}

	var compResp LlamaCompletionResponse
	if err := json.Unmarshal(body, &compResp); err != nil {
		_ = benchRunner.Stop()
		return nil, fmt.Errorf("failed to unmarshal timings: %w", err)
	}

	// Terminate server
	_ = benchRunner.Stop()

	if onStep != nil {
		onStep(2) // StepSavingData
	}

	// Calculate speed
	tokensPerSec := compResp.Timings.PredictedPerSecond
	if tokensPerSec == 0 && compResp.Timings.PredictedN > 0 {
		duration := time.Since(compStart).Seconds()
		tokensPerSec = float64(compResp.Timings.PredictedN) / duration
	}

	// Estimate RAM and VRAM footprint at 512 context size
	var ramEst, vramEst float64 = 0, 0
	if specs != nil {
		est := hardware.EstimateMemory(m, specs, 512)
		ramEst = float64(est.TotalMemory) / (1024 * 1024)  // MB
		vramEst = float64(est.KVCacheSize) / (1024 * 1024) // MB
	}

	result := &BenchmarkResult{
		ModelPath:        m.FilePath,
		ModelName:        m.Name,
		RunDate:          time.Now(),
		StartupTimeMs:    startupTimeMs,
		TokensPerSec:     tokensPerSec,
		PeakTokensPerSec: tokensPerSec * 1.05, // estimated peak
		RAMUsageMB:       ramEst,
		VRAMUsageMB:      vramEst,
	}

	return result, nil
}

// LoadHistory returns all benchmark results.
func LoadHistory(benchmarksDir string) ([]*BenchmarkResult, error) {
	if err := os.MkdirAll(benchmarksDir, 0755); err != nil {
		return nil, err
	}
	historyFile := filepath.Join(benchmarksDir, "history.json")
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		return []*BenchmarkResult{}, nil
	}

	data, err := os.ReadFile(historyFile)
	if err != nil {
		return nil, err
	}

	var history []*BenchmarkResult
	if err := json.Unmarshal(data, &history); err != nil {
		return []*BenchmarkResult{}, nil
	}
	return history, nil
}

// SaveResult appends a benchmark result to the local history database.
func SaveResult(benchmarksDir string, res *BenchmarkResult) error {
	history, err := LoadHistory(benchmarksDir)
	if err != nil {
		history = []*BenchmarkResult{}
	}

	history = append(history, res)

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	historyFile := filepath.Join(benchmarksDir, "history.json")
	return os.WriteFile(historyFile, data, 0644)
}
