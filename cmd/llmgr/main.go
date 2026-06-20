package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/BIJJUDAMA/llama-manager/config"
	"github.com/BIJJUDAMA/llama-manager/runner"
	"github.com/BIJJUDAMA/llama-manager/ui"
)

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		v := info.Main.Version
		if v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

func main() {
	resetOnboarding := flag.Bool("reset-onboarding", false, "Reset and run the onboarding tour")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("llmgr", buildVersion())
		os.Exit(0)
	}

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

	srv := runner.NewServerRunner(cfg.Paths.Cache)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		_ = srv.Stop()
		os.Exit(0)
	}()

	defer func() {
		_ = srv.Stop()
	}()

	m := ui.NewBrowserModel(cfg, srv)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
