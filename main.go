package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"llama-manager/config"
	"llama-manager/runner"
	"llama-manager/ui"
)

func main() {
	// Parse command line flags
	resetOnboarding := flag.Bool("reset-onboarding", false, "Reset and run the onboarding tour")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if *resetOnboarding {
		cfg.OnboardingCompleted = false
		if err := cfg.Save(); err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			os.Exit(1)
		}
	}

	// Create server runner using cache folder for logging
	srv := runner.NewServerRunner(cfg.Paths.Cache)

	// Setup clean cleanup of server process on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		_ = srv.Stop()
		os.Exit(0)
	}()

	// Ensure running processes are stopped on normal exit
	defer func() {
		_ = srv.Stop()
	}()

	// Initialize UI Model
	m := ui.NewBrowserModel(cfg, srv)

	// Run Bubble Tea program with AltScreen (full screen mode)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
