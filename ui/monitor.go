package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/BIJJUDAMA/llama-manager/runner"
)

type MonitorModel struct {
	srvRunner     *runner.ServerRunner
	instances     []runner.InstanceInfo
	selected      int
	width, height int
}

func NewMonitorModel(srv *runner.ServerRunner) *MonitorModel {
	return &MonitorModel{
		srvRunner: srv,
		instances: []runner.InstanceInfo{},
		selected:  0,
	}
}

func (m *MonitorModel) Refresh() {
	m.instances = m.srvRunner.GetAllInstances()
	if m.selected >= len(m.instances) {
		m.selected = len(m.instances) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m *MonitorModel) Update(msg tea.Msg) tea.Cmd {
	m.Refresh()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.instances)-1 {
				m.selected++
			}
		case "s", "K":
			if len(m.instances) > 0 && m.selected >= 0 && m.selected < len(m.instances) {
				port := m.instances[m.selected].Port
				_ = m.srvRunner.StopInstance(port)
				m.Refresh()
			}
		}
	}
	return nil
}

func (m *MonitorModel) View(width int, height int) string {
	m.width = width
	m.height = height
	m.Refresh()

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("RUNTIME SERVER MONITOR")))

	if len(m.instances) == 0 {
		sb.WriteString("  No active server instances are currently running.\n\n")
		sb.WriteString("  " + StyleHelpKey.Render("[Esc]") + " Back to Browser\n")
	} else {
		sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Active Server Instances:") + "\n")
		sb.WriteString(fmt.Sprintf("  %-6s %-20s %-10s %-10s %-6s\n", "Port", "Model", "PID", "Uptime", "Status"))
		sb.WriteString("  " + strings.Repeat("─", width-8) + "\n")

		for idx, inst := range m.instances {
			modelName := filepath.Base(inst.ModelPath)
			if len(modelName) > 20 {
				modelName = modelName[:17] + "..."
			}

			uptimeSec := int(inst.Uptime.Seconds())
			uptimeStr := fmt.Sprintf("%dh %dm %ds", uptimeSec/3600, (uptimeSec%3600)/60, uptimeSec%60)

			statusStr := lipgloss.NewStyle().Foreground(ColorSecondary).Render("Serving")

			row := fmt.Sprintf("  %-6d %-20s %-10d %-10s %-6s",
				inst.Port, modelName, inst.PID, uptimeStr, statusStr,
			)

			if idx == m.selected {
				sb.WriteString(StyleSelectedListItem.Width(width - 4).Render(row) + "\n")
			} else {
				sb.WriteString(row + "\n")
			}
		}

		sb.WriteString("\n")

		selectedInst := m.instances[m.selected]
		sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Selected Instance Performance Metrics:") + "\n")
		sb.WriteString("  " + strings.Repeat("─", width-8) + "\n")

		memStr := "Gathering..."
		mem, err := runner.GetMemoryUsage(selectedInst.PID)
		if err == nil {
			memStr = fmt.Sprintf("%.2f MB", mem)
		} else {
			memStr = "N/A"
		}

		reqStr := "Gathering..."
		reqs, err := runner.QueryServerRequests(selectedInst.Port)
		if err == nil {
			reqStr = fmt.Sprintf("%d requests", reqs)
		} else {
			reqStr = "0 requests"
		}

		sb.WriteString(fmt.Sprintf("  %-20s %d\n", "Process PID:", selectedInst.PID))
		sb.WriteString(fmt.Sprintf("  %-20s %d\n", "Server Port:", selectedInst.Port))
		sb.WriteString(fmt.Sprintf("  %-20s http://127.0.0.1:%d\n", "Endpoint:", selectedInst.Port))
		sb.WriteString(fmt.Sprintf("  %-20s %s\n", "Active Memory (RSS):", memStr))
		sb.WriteString(fmt.Sprintf("  %-20s %s\n", "Requests Handled:", reqStr))
		sb.WriteString(fmt.Sprintf("  %-20s %s\n\n", "Log File Path:", selectedInst.LogFile))

		helpStr := fmt.Sprintf("%s Stop Selected Server  %s Back to Browser",
			StyleHelpKey.Render("[S/K]"),
			StyleHelpKey.Render("[Esc]"),
		)
		sb.WriteString("  " + helpStr + "\n")
	}

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
