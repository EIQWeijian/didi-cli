package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [TICKET_ID]",
	Short: "Open a Jira ticket and create local workspace",
	Long:  `Fetches a Jira ticket and creates a local workspace with ticket information and execution plan.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runOpen,
}

type JiraTicket struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string          `json:"summary"`
		Description json.RawMessage `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee struct {
			DisplayName string `json:"displayName"`
		} `json:"assignee"`
		Reporter struct {
			DisplayName string `json:"displayName"`
		} `json:"reporter"`
		IssueType struct {
			Name string `json:"name"`
		} `json:"issuetype"`
		Priority struct {
			Name string `json:"name"`
		} `json:"priority"`
	} `json:"fields"`
}

type JiraComment struct {
	Comments []struct {
		Body   json.RawMessage `json:"body"`
		Author struct {
			DisplayName string `json:"displayName"`
		} `json:"author"`
	} `json:"comments"`
}

// ADFContent represents Atlassian Document Format structure
type ADFContent struct {
	Type    string       `json:"type"`
	Content []ADFContent `json:"content,omitempty"`
	Text    string       `json:"text,omitempty"`
}

// ExtractText converts Jira field content (string or ADF) to plain text
func ExtractText(rawContent json.RawMessage) string {
	if len(rawContent) == 0 {
		return ""
	}

	// Try to unmarshal as a string first (older Jira format)
	var simpleString string
	if err := json.Unmarshal(rawContent, &simpleString); err == nil {
		return simpleString
	}

	// Try to unmarshal as ADF (newer Jira format)
	var adf ADFContent
	if err := json.Unmarshal(rawContent, &adf); err != nil {
		return ""
	}

	return extractADFText(&adf)
}

// extractADFText recursively extracts text from ADF structure
func extractADFText(node *ADFContent) string {
	var result strings.Builder

	if node.Text != "" {
		result.WriteString(node.Text)
	}

	for i, child := range node.Content {
		childText := extractADFText(&child)
		result.WriteString(childText)

		// Add spacing based on node type
		if child.Type == "paragraph" && i < len(node.Content)-1 {
			result.WriteString("\n\n")
		} else if child.Type == "hardBreak" {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// CreateWorkspace creates a local workspace for a ticket with markdown file and execution plan
func CreateWorkspace(baseURL, email, token, ticketID string) (string, error) {
	// Create workspace directory
	workspaceDir := filepath.Join(".jira", ticketID)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Fetch ticket information
	ticket, err := FetchTicket(baseURL, email, token, ticketID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch ticket: %w", err)
	}

	// Save ticket information to markdown
	ticketMDPath := filepath.Join(workspaceDir, fmt.Sprintf("%s.md", ticketID))
	if err := saveTicketMarkdown(ticket, ticketMDPath); err != nil {
		return "", fmt.Errorf("failed to save ticket markdown: %w", err)
	}

	// Fetch comments
	comments, err := fetchComments(baseURL, email, token, ticketID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch comments: %w", err)
	}

	// Extract and save execution plan if found
	executionPlan := extractExecutionPlan(comments)
	if executionPlan != "" {
		planPath := filepath.Join(workspaceDir, "plan.md")
		if err := os.WriteFile(planPath, []byte(executionPlan), 0644); err != nil {
			return "", fmt.Errorf("failed to save execution plan: %w", err)
		}
	}

	return workspaceDir, nil
}

func runOpen(cmd *cobra.Command, args []string) error {
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

	workspaceDir, err := CreateWorkspace(jiraBaseURL, jiraEmail, jiraToken, ticketID)
	if err != nil {
		return err
	}

	// Save last ticket ID
	if err := SaveLastTicket(ticketID); err != nil {
		// Don't fail the command, just warn
		fmt.Printf("Warning: failed to save last ticket: %v\n", err)
	}

	fmt.Printf("✓ Workspace ready: %s\n", workspaceDir)
	return nil
}

// SaveLastTicket saves the ticket ID to .jira/.last-ticket
func SaveLastTicket(ticketID string) error {
	jiraDir := ".jira"
	if err := os.MkdirAll(jiraDir, 0755); err != nil {
		return err
	}

	lastTicketPath := filepath.Join(jiraDir, ".last-ticket")
	return os.WriteFile(lastTicketPath, []byte(ticketID), 0644)
}

// GetLastTicket retrieves the last ticket ID from .jira/.last-ticket
func GetLastTicket() (string, error) {
	lastTicketPath := filepath.Join(".jira", ".last-ticket")
	data, err := os.ReadFile(lastTicketPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func FetchTicket(baseURL, email, token, ticketID string) (*JiraTicket, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", baseURL, ticketID)

	req, err := http.NewRequest("GET", url, nil)
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

	var ticket JiraTicket
	if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
		return nil, err
	}

	return &ticket, nil
}

func fetchComments(baseURL, email, token, ticketID string) (*JiraComment, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/comment", baseURL, ticketID)

	req, err := http.NewRequest("GET", url, nil)
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
		return &JiraComment{}, nil // Return empty comments if not found
	}

	var comments JiraComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, err
	}

	return &comments, nil
}

func saveTicketMarkdown(ticket *JiraTicket, path string) error {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# %s: %s\n\n", ticket.Key, ticket.Fields.Summary))
	md.WriteString(fmt.Sprintf("**Type:** %s\n", ticket.Fields.IssueType.Name))
	md.WriteString(fmt.Sprintf("**Status:** %s\n", ticket.Fields.Status.Name))
	md.WriteString(fmt.Sprintf("**Priority:** %s\n", ticket.Fields.Priority.Name))
	md.WriteString(fmt.Sprintf("**Reporter:** %s\n", ticket.Fields.Reporter.DisplayName))

	if ticket.Fields.Assignee.DisplayName != "" {
		md.WriteString(fmt.Sprintf("**Assignee:** %s\n", ticket.Fields.Assignee.DisplayName))
	}

	md.WriteString("\n## Description\n\n")
	description := ExtractText(ticket.Fields.Description)
	if description != "" {
		md.WriteString(description)
	} else {
		md.WriteString("_No description provided_")
	}
	md.WriteString("\n")

	return os.WriteFile(path, []byte(md.String()), 0644)
}

func extractExecutionPlan(comments *JiraComment) string {
	// Look for comments that contain execution plan markers
	planPattern := regexp.MustCompile(`(?i)(execution\s+plan|plan:|implementation\s+plan)`)

	for _, comment := range comments.Comments {
		commentText := ExtractText(comment.Body)
		if planPattern.MatchString(commentText) {
			// Found a comment with execution plan
			return fmt.Sprintf("# Execution Plan\n\n**Author:** %s\n\n%s\n",
				comment.Author.DisplayName,
				commentText)
		}
	}

	return ""
}
