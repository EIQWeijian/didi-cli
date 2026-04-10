package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save [TICKET_ID]",
	Short: "Save execution plan to Jira as a comment",
	Long:  `Reads the local execution plan from .jira/{TICKET-ID}/plan.md and posts it as a comment to the Jira ticket. If no ticket ID is provided, uses the last opened ticket.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSave,
}

type JiraCommentRequest struct {
	Body interface{} `json:"body"`
}

type ADFDocument struct {
	Type    string        `json:"type"`
	Version int           `json:"version"`
	Content []interface{} `json:"content"`
}

type ADFNode struct {
	Type    string        `json:"type"`
	Content []interface{} `json:"content,omitempty"`
	Text    string        `json:"text,omitempty"`
	Marks   []interface{} `json:"marks,omitempty"`
}

func runSave(cmd *cobra.Command, args []string) error {
	var ticketID string

	// Get ticket ID from args or from last ticket
	if len(args) > 0 {
		ticketID = args[0]
	} else {
		var err error
		ticketID, err = GetLastTicket()
		if err != nil {
			return fmt.Errorf("no ticket ID provided and no last ticket found. Use: didi save TICKET-ID")
		}
	}

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

	// Read plan.md file
	planPath := filepath.Join(".jira", ticketID, "plan.md")
	planContent, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("failed to read plan file at %s: %w", planPath, err)
	}

	// Post plan to Jira as a comment
	if err := postPlanToJira(jiraBaseURL, jiraEmail, jiraToken, ticketID, string(planContent)); err != nil {
		return fmt.Errorf("failed to post plan to Jira: %w", err)
	}

	fmt.Printf("%s✓%s Plan synced to Jira ticket %s%s%s\n", colorGreen, colorReset, colorCyan, ticketID, colorReset)
	fmt.Printf("View at: %s%s/browse/%s%s\n", colorDim, jiraBaseURL, ticketID, colorReset)

	return nil
}

func postPlanToJira(baseURL, email, token, ticketID, planContent string) error {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/comment", baseURL, ticketID)

	// Convert markdown to ADF format
	adfBody := convertMarkdownToADF(planContent)

	commentReq := JiraCommentRequest{
		Body: adfBody,
	}

	jsonData, err := json.Marshal(commentReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Jira API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// convertMarkdownToADF converts markdown text to Atlassian Document Format (ADF)
// This is a simplified conversion - for production use, consider a proper markdown parser
func convertMarkdownToADF(markdown string) ADFDocument {
	doc := ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []interface{}{},
	}

	lines := strings.Split(markdown, "\n")
	var currentParagraphLines []string

	flushParagraph := func() {
		if len(currentParagraphLines) > 0 {
			text := strings.Join(currentParagraphLines, "\n")
			if text != "" {
				paragraph := ADFNode{
					Type: "paragraph",
					Content: []interface{}{
						ADFNode{
							Type: "text",
							Text: text,
						},
					},
				}
				doc.Content = append(doc.Content, paragraph)
			}
			currentParagraphLines = []string{}
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle headings
		if strings.HasPrefix(trimmed, "#") {
			flushParagraph()

			level := 1
			for i := 0; i < len(trimmed) && trimmed[i] == '#'; i++ {
				level++
			}
			if level > 6 {
				level = 6
			}

			headingText := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			heading := ADFNode{
				Type: "heading",
				Content: []interface{}{
					ADFNode{
						Type: "text",
						Text: headingText,
					},
				},
			}
			// Add level attribute (ADF uses attrs, but we'll keep it simple for now)
			doc.Content = append(doc.Content, heading)
			continue
		}

		// Handle code blocks
		if strings.HasPrefix(trimmed, "```") {
			flushParagraph()
			// Skip code block markers for now - just treat as text
			continue
		}

		// Empty lines trigger paragraph break
		if trimmed == "" {
			flushParagraph()
			continue
		}

		// Accumulate lines for paragraph
		currentParagraphLines = append(currentParagraphLines, line)
	}

	// Flush any remaining paragraph
	flushParagraph()

	// If no content was added, add a simple paragraph with the whole text
	if len(doc.Content) == 0 {
		doc.Content = append(doc.Content, ADFNode{
			Type: "paragraph",
			Content: []interface{}{
				ADFNode{
					Type: "text",
					Text: markdown,
				},
			},
		})
	}

	return doc
}
