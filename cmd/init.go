package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize didi with Claude Code skill and environment check",
	Long:  `Installs the /didi skill for Claude Code in ~/.claude/skills/ directory and verifies Jira environment variables are set.`,
	RunE:  runInit,
}

const skillContent = `# Didi

Manage Jira tickets using the didi CLI tool with local workspace integration.

## Usage

` + "```bash" + `
/didi list
/didi desc DDI-123
/didi open DDI-456
/didi save
/didi save DDI-123
` + "```" + `

## Commands

### ` + "`/didi list`" + `

Lists all tickets assigned to you in the active sprint:
- Non-interactive output grouped by status
- Clean, minimal display with ticket IDs and summaries
- Ticket IDs are clickable hyperlinks in supported terminals

### ` + "`/didi desc TICKET-ID`" + `

Displays Jira ticket information in the terminal with formatted output.

### ` + "`/didi open TICKET-ID`" + `

Opens a Jira ticket and creates a local workspace with:
- Ticket information in markdown format
- Execution plan extracted from comments (if available)
- Saves ticket as the "last ticket" for convenience

### ` + "`/didi save [TICKET-ID]`" + `

Saves the local execution plan back to Jira as a comment:
- If no ticket ID provided, uses the last opened ticket
- Reads ` + "`.jira/{TICKET-ID}/plan.md`" + `
- Posts content as a comment to the Jira ticket

## Workflow

### Step 1: Parse Command

Parse the user's input to extract:
1. The subcommand: ` + "`list`" + `, ` + "`desc`" + `, ` + "`open`" + `, or ` + "`save`" + `
2. The ticket ID (e.g., ` + "`DDI-123`" + `, ` + "`LTD-456`" + `) - optional for ` + "`save`" + ` and ` + "`list`" + ` commands

### Step 2: Execute Command

Run the appropriate didi command using the Bash tool:

**For ` + "`list`" + ` command:**
` + "```bash" + `
didi list
` + "```" + `

This will display all tickets assigned to the user in the active sprint, grouped by status with clickable ticket IDs.

**For ` + "`desc`" + ` command:**
` + "```bash" + `
didi desc TICKET-ID
` + "```" + `

This will display ticket information directly in the terminal with formatted colors and clickable links.

**For ` + "`open`" + ` command:**
` + "```bash" + `
didi open TICKET-ID
` + "```" + `

This will:
1. Create workspace directory: ` + "`.jira/TICKET-ID/`" + `
2. Fetch ticket from Jira API
3. Save ticket details to ` + "`.jira/TICKET-ID/TICKET-ID.md`" + `
4. Extract execution plan from comments (if found) and save to ` + "`.jira/TICKET-ID/plan.md`" + `
5. Save ticket as the "last ticket"

**For ` + "`save`" + ` command:**
` + "```bash" + `
didi save           # Uses last opened ticket
didi save TICKET-ID # Specify ticket ID
` + "```" + `

This will:
1. Read ` + "`.jira/TICKET-ID/plan.md`" + `
2. Convert markdown to Atlassian Document Format (ADF)
3. Post as a comment to the Jira ticket

Use this after updating the execution plan to sync changes back to Jira.

### Step 3: Read and Display Workspace Files (for ` + "`open`" + ` command only)

After running ` + "`didi open`" + `, read and display the created files:

1. Read ` + "`.jira/TICKET-ID/TICKET-ID.md`" + ` and display a summary
2. If ` + "`.jira/TICKET-ID/plan.md`" + ` exists, read and display its contents

Example output format:
` + "```" + `
✓ Workspace created at .jira/DDI-123

Ticket Summary:
- DDI-123: [ticket title]
- Status: [status]
- Assignee: [assignee]

Execution Plan:
[plan contents if available]
` + "```" + `

### Step 4: Error Handling

Handle common errors gracefully:

- **Missing environment variables**: If the command fails with "environment variable not set", remind the user to set:
  - ` + "`JIRA_API_TOKEN`" + `
  - ` + "`JIRA_BASE_URL`" + `
  - ` + "`JIRA_EMAIL`" + `

- **Ticket not found**: Display the error message from didi
- **Network errors**: Display helpful message about checking Jira connection

## Environment Requirements

The didi CLI requires these environment variables to be set:

` + "```bash" + `
export JIRA_API_TOKEN="your-jira-api-token"
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_EMAIL="your-email@example.com"
` + "```" + `

## Working with Execution Plans

When the user asks to view, update, or modify an execution plan, follow this logic to find the correct plan file:

### Finding the Current Ticket

1. **Check conversation context**: Look for recent mentions of ticket IDs (e.g., "DDI-123", "LTD-456")
2. **Check .jira directory**: List directories in ` + "`.jira/`" + ` to see available tickets
3. **Ask if ambiguous**: If multiple tickets exist or none mentioned, ask which ticket they're referring to

### Locating the Plan File

Once you have the ticket ID, the plan file is located at:
` + "```" + `
.jira/{TICKET-ID}/plan.md
` + "```" + `

For example: ` + "`.jira/DDI-123/plan.md`" + `

### When User Asks to Update/Modify Plan

If the user says:
- "Update the execution plan"
- "Modify the plan"
- "Add a step to the plan"
- "Change step 2 in the plan"

**Do this:**
1. Determine the ticket ID from context
2. Read ` + "`.jira/{TICKET-ID}/plan.md`" + `
3. Make the requested changes using the Edit tool
4. Confirm the changes to the user

### When User Asks to View Plan

If the user says:
- "Show me the plan"
- "What's in the execution plan?"
- "Read the plan"

**Do this:**
1. Determine the ticket ID from context
2. Read ` + "`.jira/{TICKET-ID}/plan.md`" + `
3. Display the contents to the user

### Example Scenarios

**Scenario 1: User just opened a ticket**
` + "```" + `
User: /didi open DDI-123
[You run the command and read the files]
User: Update the execution plan to add a new step
[You edit .jira/DDI-123/plan.md]
` + "```" + `

**Scenario 2: Multiple tickets in workspace**
` + "```" + `
User: Update the execution plan
[You check .jira/ and find DDI-123, DDI-456, LTD-789]
You: Which ticket's plan would you like to update? I see:
- DDI-123
- DDI-456
- LTD-789
` + "```" + `

**Scenario 3: No ticket context**
` + "```" + `
User: Show me the execution plan
[You check .jira/ directory]
You: I found these tickets: DDI-123, DDI-456. Which plan would you like to see?
` + "```" + `

## Implementation Notes

- The ` + "`didi`" + ` binary must be installed and available in PATH (typically in ` + "`~/go/bin/didi`" + `)
- The tool uses Jira REST API v3 for fetching tickets
- All workspace files are created in ` + "`.jira/`" + ` directory (gitignored)
- Ticket descriptions use Atlassian Document Format (ADF) which is automatically converted to plain text
- Execution plans are detected by pattern matching in comments for keywords like "execution plan", "plan:", or "implementation plan"
- **Always check conversation context first** when the user references "the plan" or "the execution plan"
- If no context exists, check ` + "`.jira/`" + ` directory for available tickets

## Example Outputs

**Successful desc command:**
` + "```" + `
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  DDI-123: Implement new feature
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Type:     Story
Status:   In Progress
Priority: High
Reporter: John Doe
Assignee: Jane Smith

Description:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[description content]

https://your-domain.atlassian.net/browse/DDI-123
` + "```" + `

**Successful open command:**
` + "```" + `
✓ Workspace ready: .jira/DDI-123
` + "```" + `

**Error case:**
` + "```" + `
Error: JIRA_API_TOKEN environment variable not set
` + "```" + `
`

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Printf("%s%sInitializing didi...%s\n\n", colorBold, colorCyan, colorReset)

	// Check environment variables
	jiraToken := os.Getenv("JIRA_API_TOKEN")
	jiraBaseURL := os.Getenv("JIRA_BASE_URL")
	jiraEmail := os.Getenv("JIRA_EMAIL")

	hasAllEnvVars := true
	fmt.Printf("%s%sChecking environment variables:%s\n", colorBold, colorBlue, colorReset)

	if jiraToken != "" {
		fmt.Printf("  %s✓%s JIRA_API_TOKEN is set\n", colorGreen, colorReset)
	} else {
		fmt.Printf("  %s✗%s JIRA_API_TOKEN is not set\n", colorYellow, colorReset)
		hasAllEnvVars = false
	}

	if jiraBaseURL != "" {
		fmt.Printf("  %s✓%s JIRA_BASE_URL is set (%s)\n", colorGreen, colorReset, jiraBaseURL)
	} else {
		fmt.Printf("  %s✗%s JIRA_BASE_URL is not set\n", colorYellow, colorReset)
		hasAllEnvVars = false
	}

	if jiraEmail != "" {
		fmt.Printf("  %s✓%s JIRA_EMAIL is set (%s)\n", colorGreen, colorReset, jiraEmail)
	} else {
		fmt.Printf("  %s✗%s JIRA_EMAIL is not set\n", colorYellow, colorReset)
		hasAllEnvVars = false
	}

	if !hasAllEnvVars {
		fmt.Printf("\n%s%sWarning:%s Missing environment variables. Set them with:\n", colorBold, colorYellow, colorReset)
		fmt.Printf("  export JIRA_API_TOKEN=\"your-jira-api-token\"\n")
		fmt.Printf("  export JIRA_BASE_URL=\"https://your-domain.atlassian.net\"\n")
		fmt.Printf("  export JIRA_EMAIL=\"your-email@example.com\"\n")
		fmt.Printf("\nGet API token: https://id.atlassian.com/manage-profile/security/api-tokens\n")
	}

	fmt.Printf("\n%s%sInstalling Claude Code skill:%s\n", colorBold, colorBlue, colorReset)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create skill directory path
	skillDir := filepath.Join(homeDir, ".claude", "skills", "didi")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill.md file
	skillPath := filepath.Join(skillDir, "skill.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	fmt.Printf("  %s✓%s Skill installed to %s%s%s\n", colorGreen, colorReset, colorCyan, skillPath, colorReset)

	fmt.Printf("\n%s%sUsage in Claude Code:%s\n", colorBold, colorGreen, colorReset)
	fmt.Printf("  /didi desc DDI-123\n")
	fmt.Printf("  /didi open DDI-456\n")

	if hasAllEnvVars {
		fmt.Printf("\n%s✓ All set! You're ready to use didi.%s\n", colorGreen, colorReset)
	} else {
		fmt.Printf("\n%s⚠ Set the environment variables above to start using didi.%s\n", colorYellow, colorReset)
	}

	return nil
}
