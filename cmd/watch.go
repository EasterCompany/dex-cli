package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/health"
	"github.com/EasterCompany/dex-cli/ui"
)

type tickMsg time.Time

type watchModel struct {
	serviceMap *config.ServiceMapConfig
	rows       []ui.TableRow
	quitting   bool
}

func (m watchModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.fetchStatuses,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m watchModel) fetchStatuses() tea.Msg {
	var rows []ui.TableRow

	for _, services := range m.serviceMap.Services {
		for _, service := range services {
			if service.Addr == "" {
				rows = append(rows, ui.FormatTableRow(
					service.ID,
					"N/A",
					"N/A",
					"SKIPPED",
					"N/A",
					"N/A",
				))
				continue
			}

			statusURL := strings.TrimSuffix(service.Addr, "/") + "/status"
			client := http.Client{Timeout: 1 * time.Second}
			resp, err := client.Get(statusURL)
			if err != nil {
				rows = append(rows, ui.FormatTableRow(
					service.ID,
					service.Addr,
					"N/A",
					"DOWN",
					"N/A",
					time.Now().Format("15:04:05"),
				))
				continue
			}

			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				rows = append(rows, ui.FormatTableRow(
					service.ID,
					service.Addr,
					"N/A",
					"ERROR",
					"N/A",
					time.Now().Format("15:04:05"),
				))
				continue
			}

			var statusResp health.StatusResponse
			if err := json.Unmarshal(body, &statusResp); err != nil {
				rows = append(rows, ui.FormatTableRow(
					service.ID,
					service.Addr,
					"N/A",
					"INVALID RESP",
					"N/A",
					time.Now().Format("15:04:05"),
				))
				continue
			}

			uptime := formatUptime(time.Duration(statusResp.Uptime) * time.Second)
			rows = append(rows, ui.FormatTableRow(
				statusResp.Service,
				service.Addr,
				statusResp.Version,
				statusResp.Status,
				uptime,
				time.Unix(statusResp.Timestamp, 0).Format("15:04:05"),
			))
		}
	}

	return statusUpdateMsg{rows: rows}
}

type statusUpdateMsg struct {
	rows []ui.TableRow
}

func (m watchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}

	case tickMsg:
		return m, tea.Batch(tickCmd(), m.fetchStatuses)

	case statusUpdateMsg:
		m.rows = msg.rows
		return m, nil
	}

	return m, nil
}

func (m watchModel) View() string {
	if m.quitting {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("Stopped watching services. Goodbye!\n")
	}

	var s strings.Builder

	// Info line
	s.WriteString(ui.RenderSubtitle("Press 'q' or Ctrl+C to exit • Refreshing every 2 seconds"))
	s.WriteString("\n\n")

	// Table
	if len(m.rows) == 0 {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render("Loading service statuses..."))
	} else {
		table := ui.CreateServiceTable(m.rows)
		s.WriteString(ui.RenderTable(table))
	}

	return s.String()
}

// Watch provides a live dashboard of all service statuses
func Watch() error {
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	m := watchModel{
		serviceMap: serviceMap,
		rows:       []ui.TableRow{},
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running watch: %w", err)
	}

	return nil
}
