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
	Type    string                 `json:"type"`
	Content []interface{}          `json:"content,omitempty"`
	Text    string                 `json:"text,omitempty"`
	Marks   []interface{}          `json:"marks,omitempty"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
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
func convertMarkdownToADF(markdown string) ADFDocument {
	doc := ADFDocument{
		Type:    "doc",
		Version: 1,
		Content: []interface{}{},
	}

	lines := strings.Split(markdown, "\n")
	var currentParagraphLines []string
	var inCodeBlock bool
	var codeBlockLines []string
	var codeBlockLanguage string

	flushParagraph := func() {
		if len(currentParagraphLines) > 0 {
			text := strings.Join(currentParagraphLines, " ")
			if text != "" {
				content := parseInlineMarkdown(text)
				paragraph := ADFNode{
					Type:    "paragraph",
					Content: content,
				}
				doc.Content = append(doc.Content, paragraph)
			}
			currentParagraphLines = []string{}
		}
	}

	flushCodeBlock := func() {
		if len(codeBlockLines) > 0 {
			codeBlock := ADFNode{
				Type: "codeBlock",
				Attrs: map[string]interface{}{
					"language": codeBlockLanguage,
				},
				Content: []interface{}{
					ADFNode{
						Type: "text",
						Text: strings.Join(codeBlockLines, "\n"),
					},
				},
			}
			doc.Content = append(doc.Content, codeBlock)
			codeBlockLines = []string{}
			codeBlockLanguage = ""
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Handle code blocks
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				flushCodeBlock()
				inCodeBlock = false
			} else {
				flushParagraph()
				inCodeBlock = true
				codeBlockLanguage = strings.TrimPrefix(trimmed, "```")
				if codeBlockLanguage == "" {
					codeBlockLanguage = "text"
				}
			}
			continue
		}

		if inCodeBlock {
			codeBlockLines = append(codeBlockLines, line)
			continue
		}

		// Handle headings
		if strings.HasPrefix(trimmed, "#") {
			flushParagraph()

			level := 0
			for j := 0; j < len(trimmed) && trimmed[j] == '#'; j++ {
				level++
			}
			if level > 6 {
				level = 6
			}

			headingText := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			heading := ADFNode{
				Type: "heading",
				Attrs: map[string]interface{}{
					"level": level,
				},
				Content: parseInlineMarkdown(headingText),
			}
			doc.Content = append(doc.Content, heading)
			continue
		}

		// Handle bullet lists (-, *, +)
		if len(trimmed) > 2 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') && trimmed[1] == ' ' {
			flushParagraph()

			// Collect all consecutive list items
			listItems := []interface{}{}
			for i < len(lines) {
				currentLine := strings.TrimSpace(lines[i])
				if len(currentLine) > 2 && (currentLine[0] == '-' || currentLine[0] == '*' || currentLine[0] == '+') && currentLine[1] == ' ' {
					itemText := strings.TrimSpace(currentLine[2:])
					listItem := ADFNode{
						Type: "listItem",
						Content: []interface{}{
							ADFNode{
								Type:    "paragraph",
								Content: parseInlineMarkdown(itemText),
							},
						},
					}
					listItems = append(listItems, listItem)
					i++
				} else {
					break
				}
			}
			i-- // Adjust for outer loop increment

			bulletList := ADFNode{
				Type:    "bulletList",
				Content: listItems,
			}
			doc.Content = append(doc.Content, bulletList)
			continue
		}

		// Handle numbered lists
		if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			dotIndex := strings.Index(trimmed, ".")
			if dotIndex > 0 && dotIndex < len(trimmed)-1 && trimmed[dotIndex+1] == ' ' {
				flushParagraph()

				// Collect all consecutive numbered list items
				listItems := []interface{}{}
				for i < len(lines) {
					currentLine := strings.TrimSpace(lines[i])
					dotIdx := strings.Index(currentLine, ".")
					if len(currentLine) > 2 && currentLine[0] >= '0' && currentLine[0] <= '9' && dotIdx > 0 && dotIdx < len(currentLine)-1 && currentLine[dotIdx+1] == ' ' {
						itemText := strings.TrimSpace(currentLine[dotIdx+2:])
						listItem := ADFNode{
							Type: "listItem",
							Content: []interface{}{
								ADFNode{
									Type:    "paragraph",
									Content: parseInlineMarkdown(itemText),
								},
							},
						}
						listItems = append(listItems, listItem)
						i++
					} else {
						break
					}
				}
				i-- // Adjust for outer loop increment

				orderedList := ADFNode{
					Type:    "orderedList",
					Content: listItems,
				}
				doc.Content = append(doc.Content, orderedList)
				continue
			}
		}

		// Handle horizontal rules
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			flushParagraph()
			rule := ADFNode{
				Type: "rule",
			}
			doc.Content = append(doc.Content, rule)
			continue
		}

		// Empty lines trigger paragraph break
		if trimmed == "" {
			flushParagraph()
			continue
		}

		// Accumulate lines for paragraph
		currentParagraphLines = append(currentParagraphLines, trimmed)
	}

	// Flush any remaining content
	flushParagraph()
	flushCodeBlock()

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

// parseInlineMarkdown converts inline markdown (bold, italic, code, links) to ADF
func parseInlineMarkdown(text string) []interface{} {
	content := []interface{}{}
	current := ""
	i := 0

	for i < len(text) {
		// Bold with **
		if i+1 < len(text) && text[i:i+2] == "**" {
			if current != "" {
				content = append(content, ADFNode{Type: "text", Text: current})
				current = ""
			}
			// Find closing **
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				boldText := text[i+2 : i+2+end]
				content = append(content, ADFNode{
					Type: "text",
					Text: boldText,
					Marks: []interface{}{
						map[string]interface{}{"type": "strong"},
					},
				})
				i += 4 + end
				continue
			}
		}

		// Italic with *
		if text[i] == '*' && (i == 0 || text[i-1] != '*') && (i+1 >= len(text) || text[i+1] != '*') {
			if current != "" {
				content = append(content, ADFNode{Type: "text", Text: current})
				current = ""
			}
			// Find closing *
			end := strings.Index(text[i+1:], "*")
			if end != -1 && (i+1+end+1 >= len(text) || text[i+1+end+1] != '*') {
				italicText := text[i+1 : i+1+end]
				content = append(content, ADFNode{
					Type: "text",
					Text: italicText,
					Marks: []interface{}{
						map[string]interface{}{"type": "em"},
					},
				})
				i += 2 + end
				continue
			}
		}

		// Inline code with `
		if text[i] == '`' {
			if current != "" {
				content = append(content, ADFNode{Type: "text", Text: current})
				current = ""
			}
			// Find closing `
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				codeText := text[i+1 : i+1+end]
				content = append(content, ADFNode{
					Type: "text",
					Text: codeText,
					Marks: []interface{}{
						map[string]interface{}{"type": "code"},
					},
				})
				i += 2 + end
				continue
			}
		}

		// Links [text](url)
		if text[i] == '[' {
			closeBracket := strings.Index(text[i+1:], "]")
			if closeBracket != -1 && i+closeBracket+2 < len(text) && text[i+closeBracket+2] == '(' {
				closeParen := strings.Index(text[i+closeBracket+3:], ")")
				if closeParen != -1 {
					if current != "" {
						content = append(content, ADFNode{Type: "text", Text: current})
						current = ""
					}
					linkText := text[i+1 : i+1+closeBracket]
					linkURL := text[i+closeBracket+3 : i+closeBracket+3+closeParen]
					content = append(content, ADFNode{
						Type: "text",
						Text: linkText,
						Marks: []interface{}{
							map[string]interface{}{
								"type": "link",
								"attrs": map[string]interface{}{
									"href": linkURL,
								},
							},
						},
					})
					i += closeBracket + closeParen + 4
					continue
				}
			}
		}

		current += string(text[i])
		i++
	}

	if current != "" {
		content = append(content, ADFNode{Type: "text", Text: current})
	}

	if len(content) == 0 {
		content = append(content, ADFNode{Type: "text", Text: ""})
	}

	return content
}
