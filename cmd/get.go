package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get [URL]",
	Short: "Fetch wiki page content",
	Long:  `Fetches content from Confluence wiki pages using Jira API credentials.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

type ConfluencePage struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Body  struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
}

func runGet(cmd *cobra.Command, args []string) error {
	pageURL := args[0]

	// Parse URL
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check if it's an Atlassian wiki URL
	if !strings.Contains(parsedURL.Host, "atlassian.net") || !strings.Contains(parsedURL.Path, "/wiki/") {
		return fmt.Errorf("URL must be an Atlassian wiki page (e.g., https://domain.atlassian.net/wiki/...)")
	}

	// Extract page ID from URL
	// URL format: https://evolutioniq.atlassian.net/wiki/spaces/ENG/pages/562692754/Product+Repositories
	pageID, err := extractPageID(parsedURL.Path)
	if err != nil {
		return err
	}

	// Get Jira configuration from environment
	jiraToken := os.Getenv("JIRA_API_TOKEN")
	jiraEmail := os.Getenv("JIRA_EMAIL")

	if jiraToken == "" {
		return fmt.Errorf("JIRA_API_TOKEN environment variable not set")
	}
	if jiraEmail == "" {
		return fmt.Errorf("JIRA_EMAIL environment variable not set")
	}

	// Construct base URL
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Fetch page content
	page, err := fetchConfluencePage(baseURL, jiraEmail, jiraToken, pageID)
	if err != nil {
		return fmt.Errorf("failed to fetch wiki page: %w", err)
	}

	// Display page content
	displayWikiPage(page)

	return nil
}

func extractPageID(path string) (string, error) {
	// Match pattern: /wiki/spaces/{SPACE}/pages/{PAGE_ID}/{TITLE}
	re := regexp.MustCompile(`/wiki/spaces/[^/]+/pages/(\d+)`)
	matches := re.FindStringSubmatch(path)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract page ID from URL")
	}
	return matches[1], nil
}

func fetchConfluencePage(baseURL, email, token, pageID string) (*ConfluencePage, error) {
	// Confluence REST API endpoint
	apiURL := fmt.Sprintf("%s/wiki/rest/api/content/%s?expand=body.storage", baseURL, pageID)

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
		return nil, fmt.Errorf("Confluence API returned status %d: %s", resp.StatusCode, string(body))
	}

	var page ConfluencePage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	return &page, nil
}

func displayWikiPage(page *ConfluencePage) {
	fmt.Printf("\n")
	fmt.Printf("%s%s%s: %s\n", colorBold, colorCyan, page.Type, page.Title)
	fmt.Printf("%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorDim, colorCyan, colorReset)
	fmt.Printf("\n")

	// Convert Confluence storage format (XHTML) to plain text
	content := convertStorageToText(page.Body.Storage.Value)

	fmt.Printf("%s\n", content)
	fmt.Printf("\n")
}

func convertStorageToText(storage string) string {
	// Basic conversion from Confluence storage format (XHTML) to readable text
	// This is a simplified conversion - for better formatting, consider using an HTML parser

	text := storage

	// Remove XML declaration
	text = regexp.MustCompile(`<\?xml[^>]+\?>`).ReplaceAllString(text, "")

	// Convert headings
	text = regexp.MustCompile(`<h1>([^<]+)</h1>`).ReplaceAllString(text, "\n# $1\n")
	text = regexp.MustCompile(`<h2>([^<]+)</h2>`).ReplaceAllString(text, "\n## $1\n")
	text = regexp.MustCompile(`<h3>([^<]+)</h3>`).ReplaceAllString(text, "\n### $1\n")
	text = regexp.MustCompile(`<h4>([^<]+)</h4>`).ReplaceAllString(text, "\n#### $1\n")

	// Convert paragraphs
	text = regexp.MustCompile(`<p>([^<]*)</p>`).ReplaceAllString(text, "$1\n\n")
	text = regexp.MustCompile(`<p>(.*?)</p>`).ReplaceAllString(text, "$1\n\n")

	// Convert line breaks
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")

	// Convert lists
	text = regexp.MustCompile(`<li>([^<]+)</li>`).ReplaceAllString(text, "• $1\n")
	text = strings.ReplaceAll(text, "<ul>", "\n")
	text = strings.ReplaceAll(text, "</ul>", "\n")
	text = strings.ReplaceAll(text, "<ol>", "\n")
	text = strings.ReplaceAll(text, "</ol>", "\n")

	// Convert strong/bold
	text = regexp.MustCompile(`<strong>([^<]+)</strong>`).ReplaceAllString(text, "**$1**")
	text = regexp.MustCompile(`<b>([^<]+)</b>`).ReplaceAllString(text, "**$1**")

	// Convert emphasis/italic
	text = regexp.MustCompile(`<em>([^<]+)</em>`).ReplaceAllString(text, "*$1*")
	text = regexp.MustCompile(`<i>([^<]+)</i>`).ReplaceAllString(text, "*$1*")

	// Convert code
	text = regexp.MustCompile(`<code>([^<]+)</code>`).ReplaceAllString(text, "`$1`")

	// Convert links
	text = regexp.MustCompile(`<a[^>]+href="([^"]+)"[^>]*>([^<]+)</a>`).ReplaceAllString(text, "$2 ($1)")

	// Remove remaining HTML tags
	text = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(text, "")

	// Clean up excessive newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	// Decode HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return strings.TrimSpace(text)
}
