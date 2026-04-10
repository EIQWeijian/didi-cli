package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorBlue   = "\033[34m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

var descCmd = &cobra.Command{
	Use:   "desc [TICKET_ID]",
	Short: "Display Jira ticket information",
	Long:  `Fetches and displays a Jira ticket's information in the terminal.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDesc,
}

func runDesc(cmd *cobra.Command, args []string) error {
	ticketID := args[0]

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

	// Fetch ticket information
	ticket, err := FetchTicket(jiraBaseURL, jiraEmail, jiraToken, ticketID)
	if err != nil {
		return fmt.Errorf("failed to fetch ticket: %w", err)
	}

	// Save last ticket ID
	if err := SaveLastTicket(ticketID); err != nil {
		// Don't fail the command, just warn
		fmt.Printf("Warning: failed to save last ticket: %v\n", err)
	}

	// Display ticket information
	displayTicket(ticket, jiraBaseURL)

	return nil
}

func displayTicket(ticket *JiraTicket, baseURL string) {
	description := ExtractText(ticket.Fields.Description)
	ticketURL := fmt.Sprintf("%s/browse/%s", baseURL, ticket.Key)

	// Create clickable hyperlink for the ticket ID
	clickableTicketID := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", ticketURL, ticket.Key)

	fmt.Printf("\n")
	fmt.Printf("%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorDim, colorCyan, colorReset)
	fmt.Printf("%s%s  %s%s%s: %s%s\n", colorBold, colorCyan, clickableTicketID, colorReset, colorBold, ticket.Fields.Summary, colorReset)
	fmt.Printf("%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorDim, colorCyan, colorReset)
	fmt.Printf("\n")
	fmt.Printf("%s%-9s%s %s\n", colorYellow, "Type:", colorReset, ticket.Fields.IssueType.Name)
	fmt.Printf("%s%-9s%s %s%s%s\n", colorYellow, "Status:", colorReset, colorGreen, ticket.Fields.Status.Name, colorReset)
	fmt.Printf("%s%-9s%s %s\n", colorYellow, "Priority:", colorReset, ticket.Fields.Priority.Name)
	fmt.Printf("%s%-9s%s %s\n", colorYellow, "Reporter:", colorReset, ticket.Fields.Reporter.DisplayName)

	if ticket.Fields.Assignee.DisplayName != "" {
		fmt.Printf("%s%-9s%s %s\n", colorYellow, "Assignee:", colorReset, ticket.Fields.Assignee.DisplayName)
	}

	fmt.Printf("\n")
	fmt.Printf("%s%sDescription:%s\n", colorBold, colorBlue, colorReset)
	fmt.Printf("%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorDim, colorBlue, colorReset)
	if description != "" {
		fmt.Printf("%s\n", description)
	} else {
		fmt.Printf("%sNo description provided%s\n", colorDim, colorReset)
	}
	fmt.Printf("\n")
	fmt.Printf("%s%s%s%s\n", colorDim, colorCyan, ticketURL, colorReset)
	fmt.Printf("\n")
}
