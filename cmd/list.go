package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets from active sprint",
	Long:  `Non-interactive list of tickets assigned to you in the active sprint, grouped by status with clickable links.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
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

	if len(tickets) == 0 {
		fmt.Printf("\n%sNo tickets found in active sprint%s\n\n", colorDim, colorReset)
		return nil
	}

	// Display tickets grouped by status
	displayTicketList(tickets, jiraBaseURL)

	return nil
}

func displayTicketList(tickets []JiraTicket, baseURL string) {
	fmt.Printf("\n")
	fmt.Printf("%s%sActive Sprint Tickets (%d)%s\n", colorBold, colorCyan, len(tickets), colorReset)
	fmt.Printf("%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorDim, colorCyan, colorReset)
	fmt.Printf("\n")

	// Group tickets by status
	ticketsByStatus := make(map[string][]JiraTicket)
	for _, ticket := range tickets {
		status := ticket.Fields.Status.Name
		ticketsByStatus[status] = append(ticketsByStatus[status], ticket)
	}

	// Get sorted status names
	statuses := make([]string, 0, len(ticketsByStatus))
	for status := range ticketsByStatus {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	// Display tickets by status
	for _, status := range statuses {
		statusTickets := ticketsByStatus[status]

		fmt.Printf("%s%s%s (%d)%s\n", colorBold, colorGreen, status, len(statusTickets), colorReset)
		fmt.Printf("%s%s────────────────────────────────────────────────────%s\n", colorDim, colorGreen, colorReset)

		for _, ticket := range statusTickets {
			displayTicketLine(ticket, baseURL)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("%sTip: Click on ticket IDs to open in browser (if supported by your terminal)%s\n", colorDim, colorReset)
	fmt.Printf("\n")
}

func displayTicketLine(ticket JiraTicket, baseURL string) {
	ticketURL := fmt.Sprintf("%s/browse/%s", baseURL, ticket.Key)

	// Create clickable hyperlink using OSC 8 escape sequence
	clickableTicketID := makeClickableLink(ticketURL, ticket.Key)

	// Format: TICKET-ID: Summary
	fmt.Printf("  %s: %s\n",
		clickableTicketID,
		truncateString(ticket.Fields.Summary, 80))
}

// makeClickableLink creates a terminal hyperlink
// This uses OSC 8 which is supported by modern terminals (iTerm2, Terminal.app, VS Code, etc.)
func makeClickableLink(url, text string) string {
	// OSC 8 format: ESC ] 8 ; ; URL ST text ESC ] 8 ; ; ST
	// where ST can be either ESC \ or BEL
	osc8Start := fmt.Sprintf("\033]8;;%s\007", url)  // Using BEL (\007) as terminator
	osc8End := "\033]8;;\007"                          // Clear hyperlink

	return fmt.Sprintf("%s%s%s%s%s", colorCyan, osc8Start, text, osc8End, colorReset)
}

// truncateString truncates a string to maxLen and adds ... if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Handle multi-byte characters properly
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen-3]) + "..."
}
