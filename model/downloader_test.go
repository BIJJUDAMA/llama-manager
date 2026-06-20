package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadQueueFlow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llama-manager-download-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	q := NewDownloadQueue(tempDir, "hf_mock_token")

	// Initially queue should be empty
	if len(q.GetTasks()) != 0 {
		t.Errorf("expected empty task list initially")
	}

	// Add a task
	task := q.AddTask("org/repo", "model.gguf", 1000, "http://example.com/model.gguf")

	// Verify task details
	tasks := q.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in queue, got %d", len(tasks))
	}

	t0 := tasks[0]
	if t0.ModelID != "org/repo" {
		t.Errorf("expected ModelID org/repo, got %s", t0.ModelID)
	}
	if t0.FileName != "model.gguf" {
		t.Errorf("expected FileName model.gguf, got %s", t0.FileName)
	}
	if t0.TotalSize != 1000 {
		t.Errorf("expected TotalSize 1000, got %d", t0.TotalSize)
	}
	expectedPath := filepath.Join(tempDir, "org_repo", "model.gguf")
	if t0.DestPath != expectedPath {
		t.Errorf("expected DestPath %q, got %q", expectedPath, t0.DestPath)
	}

	// Test notify channel contains task update
	select {
	case notifiedTask := <-q.GetChan():
		if notifiedTask != task {
			t.Errorf("expected to be notified with added task")
		}
	default:
		t.Errorf("expected to receive task notification on channel")
	}

	// Test Pause Task
	q.PauseTask(task)
	if task.Status != StatusPaused {
		t.Errorf("expected task status to be StatusPaused, got %d", task.Status)
	}

	// Test Resume Task
	q.ResumeTask(task)
	if task.Status != StatusQueued {
		t.Errorf("expected task status to be StatusQueued, got %d", task.Status)
	}

	// Test Cancel Task
	q.CancelTask(task)
	if len(q.GetTasks()) != 0 {
		t.Errorf("expected task to be removed from queue after Cancel")
	}
}

func TestClearAndRemoveTasks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "llama-manager-clear-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	q := NewDownloadQueue(tempDir, "hf_mock_token")

	// Add several tasks
	t1 := q.AddTask("org/repo1", "model1.gguf", 1000, "http://example.com/model1.gguf")
	t2 := q.AddTask("org/repo2", "model2.gguf", 2000, "http://example.com/model2.gguf")
	t3 := q.AddTask("org/repo3", "model3.gguf", 3000, "http://example.com/model3.gguf")

	// Set their statuses manually for testing
	t1.Status = StatusCompleted
	t2.Status = StatusDownloading
	t3.Status = StatusFailed

	// Clear finished should remove t1 and t3, but keep t2
	q.ClearFinishedTasks()

	tasks := q.GetTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task remaining, got %d", len(tasks))
	}
	if tasks[0] != t2 {
		t.Errorf("expected remaining task to be t2")
	}

	// Remove task t2 using RemoveTask
	q.RemoveTask(t2)
	if len(q.GetTasks()) != 0 {
		t.Errorf("expected 0 tasks remaining after RemoveTask")
	}
}

