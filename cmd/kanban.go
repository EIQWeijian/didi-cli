package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var kanbanCmd = &cobra.Command{
	Use:   "kanban",
	Short: "Display kanban board for active sprint",
	Long:  `Shows tickets assigned to you in the active sprint, grouped by status.`,
	RunE:  runKanban,
}

type JiraSearchResult struct {
	Issues []JiraTicket `json:"issues"`
}

type kanbanModel struct {
	tickets        []JiraTicket
	displayOrder   []int // indices into tickets array in display order
	cursor         int   // index into displayOrder
	baseURL        string
	email          string
	token          string
	err            error
	showDetails    bool
	detailsText    string
	statusMessage  string
}

func (m *kanbanModel) Init() tea.Cmd {
	return nil
}

func (m *kanbanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.showDetails {
				m.showDetails = false
				return m, nil
			}
			return m, tea.Quit

		case "up", "k":
			if !m.showDetails && m.cursor > 0 {
				m.cursor--
				m.statusMessage = "" // Clear status when navigating
			}

		case "down", "j":
			if !m.showDetails && m.cursor < len(m.displayOrder)-1 {
				m.cursor++
				m.statusMessage = "" // Clear status when navigating
			}

		case "enter":
			if !m.showDetails && len(m.displayOrder) > 0 {
				ticketIdx := m.displayOrder[m.cursor]
				ticket, err := FetchTicket(m.baseURL, m.email, m.token, m.tickets[ticketIdx].Key)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.detailsText = formatTicketDetails(ticket, m.baseURL)
				m.showDetails = true
			} else {
				m.showDetails = false
			}

		case "o":
			if len(m.displayOrder) > 0 {
				ticketIdx := m.displayOrder[m.cursor]
				openBrowser(fmt.Sprintf("%s/browse/%s", m.baseURL, m.tickets[ticketIdx].Key))
			}

		case "s":
			if !m.showDetails && len(m.displayOrder) > 0 {
				ticketIdx := m.displayOrder[m.cursor]
				ticketID := m.tickets[ticketIdx].Key
				workspaceDir, err := CreateWorkspace(m.baseURL, m.email, m.token, ticketID)
				if err != nil {
					m.statusMessage = fmt.Sprintf("✗ Failed to save: %v", err)
				} else {
					m.statusMessage = fmt.Sprintf("✓ Saved to %s", workspaceDir)
				}
			}

		case "esc":
			if m.showDetails {
				m.showDetails = false
			} else if m.statusMessage != "" {
				m.statusMessage = ""
			}
		}
	}

	return m, nil
}

func (m *kanbanModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nError: %v\n\n", m.err)
	}

	if m.showDetails {
		return m.detailsText + "\n\n" + dimText("Press Enter or Esc to return") + "\n\n"
	}

	if len(m.tickets) == 0 {
		return "\n" + dimText("No tickets found in active sprint") + "\n\n"
	}

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(boldText(cyanText("Active Sprint - My Tickets")))
	b.WriteString("\n")
	b.WriteString(dimText(cyanText("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
	b.WriteString("\n\n")

	// Group tickets by status
	ticketsByStatus := make(map[string][]int)
	for i, ticket := range m.tickets {
		status := ticket.Fields.Status.Name
		ticketsByStatus[status] = append(ticketsByStatus[status], i)
	}

	// Get sorted status names
	statuses := make([]string, 0, len(ticketsByStatus))
	for status := range ticketsByStatus {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	// Build display order as we render
	m.displayOrder = make([]int, 0, len(m.tickets))
	displayPosition := 0

	for _, status := range statuses {
		indices := ticketsByStatus[status]

		b.WriteString(boldText(greenText(fmt.Sprintf("%s (%d)", status, len(indices)))))
		b.WriteString("\n")
		b.WriteString(dimText(greenText("────────────────────────────────────────────────────")))
		b.WriteString("\n")

		for _, ticketIdx := range indices {
			m.displayOrder = append(m.displayOrder, ticketIdx)

			ticket := m.tickets[ticketIdx]
			cursor := "  "
			isSelected := displayPosition == m.cursor

			if isSelected {
				cursor = "> "
			}

			priorityIcon := getPriorityIcon(ticket.Fields.Priority.Name)
			typeIcon := getTypeIcon(ticket.Fields.IssueType.Name)

			line := fmt.Sprintf("%s%s %s %s %s",
				cursor,
				priorityIcon,
				typeIcon,
				cyanText(ticket.Key),
				ticket.Fields.Summary)

			if isSelected {
				b.WriteString(boldText(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")

			displayPosition++
		}
		b.WriteString("\n")
	}

	b.WriteString(dimText(yellowText(fmt.Sprintf("Total: %d ticket(s)", len(m.tickets)))))
	b.WriteString("\n\n")

	if m.statusMessage != "" {
		b.WriteString(greenText(m.statusMessage))
		b.WriteString("\n\n")
	}

	b.WriteString(dimText("↑/↓ or j/k: Navigate | Enter: Details | s: Save | o: Open in Browser | q: Quit"))
	b.WriteString("\n\n")

	return b.String()
}

func runKanban(cmd *cobra.Command, args []string) error {
	// Get Jira configuration from environment
	jiraToken := os.Getenv("JIRA_API_TOKEN")
	jiraBaseURL := os.Getenv("JIRA_BASE_URL")
	jiraEmail := os.Getenv("JIRA_EMAIL")

	if jiraToken == "" {
		return fmt.Errorf("JIRA_API_TOKEN environment variable not set")
	}
	if jiraBaseURL == "" {
		return fmt.Errorf("JIRA_BASE_URL environment variable not set (e.g., https://your-domain.atlassian.net)")
	}
	if jiraEmail == "" {
		return fmt.Errorf("JIRA_EMAIL environment variable not set")
	}

	// Fetch tickets for active sprint assigned to current user
	tickets, err := fetchActiveSprintTickets(jiraBaseURL, jiraEmail, jiraToken)
	if err != nil {
		return fmt.Errorf("failed to fetch tickets: %w", err)
	}

	// Start interactive mode
	p := tea.NewProgram(&kanbanModel{
		tickets: tickets,
		baseURL: jiraBaseURL,
		email:   jiraEmail,
		token:   jiraToken,
	})

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func fetchActiveSprintTickets(baseURL, email, token string) ([]JiraTicket, error) {
	// JQL query: assignee = currentUser() AND sprint in openSprints()
	jql := "assignee = currentUser() AND sprint in openSprints() ORDER BY status ASC, priority DESC"

	encodedJQL := url.QueryEscape(jql)
	apiURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&fields=key,summary,status,priority,issuetype", baseURL, encodedJQL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Jira API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result JiraSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Issues, nil
}

func groupTicketsByStatus(tickets []JiraTicket) map[string][]JiraTicket {
	grouped := make(map[string][]JiraTicket)

	for _, ticket := range tickets {
		status := ticket.Fields.Status.Name
		grouped[status] = append(grouped[status], ticket)
	}

	return grouped
}

func formatTicketDetails(ticket *JiraTicket, baseURL string) string {
	description := ExtractText(ticket.Fields.Description)
	ticketURL := fmt.Sprintf("%s/browse/%s", baseURL, ticket.Key)

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(dimText(cyanText("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
	b.WriteString("\n")
	b.WriteString(boldText(cyanText(fmt.Sprintf("  %s: %s", ticket.Key, ticket.Fields.Summary))))
	b.WriteString("\n")
	b.WriteString(dimText(cyanText("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
	b.WriteString("\n\n")

	b.WriteString(yellowText(fmt.Sprintf("%-9s", "Type:")))
	b.WriteString(fmt.Sprintf(" %s\n", ticket.Fields.IssueType.Name))

	b.WriteString(yellowText(fmt.Sprintf("%-9s", "Status:")))
	b.WriteString(fmt.Sprintf(" %s\n", greenText(ticket.Fields.Status.Name)))

	b.WriteString(yellowText(fmt.Sprintf("%-9s", "Priority:")))
	b.WriteString(fmt.Sprintf(" %s\n", ticket.Fields.Priority.Name))

	b.WriteString(yellowText(fmt.Sprintf("%-9s", "Reporter:")))
	b.WriteString(fmt.Sprintf(" %s\n", ticket.Fields.Reporter.DisplayName))

	if ticket.Fields.Assignee.DisplayName != "" {
		b.WriteString(yellowText(fmt.Sprintf("%-9s", "Assignee:")))
		b.WriteString(fmt.Sprintf(" %s\n", ticket.Fields.Assignee.DisplayName))
	}

	b.WriteString("\n")
	b.WriteString(boldText(blueText("Description:")))
	b.WriteString("\n")
	b.WriteString(dimText(blueText("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")))
	b.WriteString("\n")

	if description != "" {
		b.WriteString(description)
	} else {
		b.WriteString(dimText("No description provided"))
	}

	b.WriteString("\n\n")
	b.WriteString(dimText(cyanText(ticketURL)))
	b.WriteString("\n")

	return b.String()
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// Helper functions for text styling
func boldText(s string) string {
	return colorBold + s + colorReset
}

func dimText(s string) string {
	return colorDim + s + colorReset
}

func cyanText(s string) string {
	return colorCyan + s + colorReset
}

func yellowText(s string) string {
	return colorYellow + s + colorReset
}

func greenText(s string) string {
	return colorGreen + s + colorReset
}

func blueText(s string) string {
	return colorBlue + s + colorReset
}

func getPriorityIcon(priority string) string {
	switch priority {
	case "Highest", "Critical":
		return "🔴"
	case "High":
		return "🟠"
	case "Medium":
		return "🟡"
	case "Low":
		return "🟢"
	case "Lowest":
		return "⚪"
	default:
		return "⚫"
	}
}

func getTypeIcon(issueType string) string {
	switch issueType {
	case "Bug":
		return "🐛"
	case "Task":
		return "📋"
	case "Story":
		return "📖"
	case "Epic":
		return "🎯"
	case "Subtask", "Sub-task":
		return "📌"
	default:
		return "📝"
	}
}
