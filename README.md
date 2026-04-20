# Didi CLI

A CLI tool for managing Jira tickets with local workspace. 
- Includes Claude Skill integration
- syncs implementation plans `plan.md` back to jira

<img width="1616" height="959" alt="Screenshot 2026-04-10 at 3 19 38 PM" src="https://github.com/user-attachments/assets/2642c1cd-7d62-4cc0-ba1d-12fc098530a9" />

## Quick Start

```bash
# 1. Install (downloads pre-built binary, no Go required)
curl -fsSL https://raw.githubusercontent.com/EIQWeijian/didi-cli/main/install.sh | bash

# 2. Add to your ~/.bashrc or ~/.zshrc to persist across sessions
export JIRA_API_TOKEN="your-api-token"
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_EMAIL="your-email@example.com"
export PATH="$HOME/go/bin:$PATH"

# 3. Initialize Claude Code skill (optional)
didi init

# 4. Start using!
didi list                # List your active sprint tickets
didi open DDI-123        # Open a ticket and create workspace
didi save                # Save plan back to Jira
```

## Features

- List active sprint tickets with clickable hyperlinks
- Fetch Jira ticket information and create local workspaces
- Fetch Confluence wiki pages and display in terminal
- Automatically extract execution plans from ticket comments
- Store ticket details in markdown format
- Sync local execution plan changes back to Jira as comments
- Track last opened ticket for quick operations
- Interactive kanban board for sprint management
- Claude Code integration with `/didi` skill for AI-powered workflow

## Installation

**macOS & Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/EIQWeijian/didi-cli/main/install.sh | bash
```

**Custom install location:**
```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/EIQWeijian/didi-cli/main/install.sh | bash
```

### Initialize

After installation, initialize didi to install the Claude Code skill and verify your environment:

```bash
didi init
```

This will:
- Check that required Jira environment variables are set
- Install the `/didi` slash command to `~/.claude/commands/didi.md`, making it available in all Claude Code sessions
- Configure JIRA env vars in `.claude/settings.local.json` so Claude Code's Bash tool can access them (Claude Code runs a non-interactive shell that doesn't load `~/.bashrc`)

## Configuration

Add the following to your `~/.bashrc` or `~/.zshrc` so they persist across sessions:

```bash
export JIRA_API_TOKEN="your-jira-api-token"
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_EMAIL="your-email@example.com"
export PATH="$HOME/go/bin:$PATH"
```

Then reload your shell: `source ~/.bashrc` (or `source ~/.zshrc`).

### Getting a Jira API Token

1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Give it a name and copy the token
4. Set it as the `JIRA_API_TOKEN` environment variable

## Usage

### CLI Usage

#### List active sprint tickets

```bash
didi list
```

Non-interactive list of all tickets assigned to you in the active sprint, grouped by status. Ticket IDs are clickable hyperlinks in supported terminals (iTerm2, VS Code terminal, Windows Terminal, modern Terminal.app).

#### View ticket details

```bash
didi desc DDI-123
```

Displays formatted ticket information in the terminal with colors and clickable links.

#### Open a Jira ticket

```bash
didi open DDI-435
```

This command will:
1. Create a folder structure: `.jira/DDI-435/`
2. Fetch the ticket information from Jira
3. Save ticket details to `.jira/DDI-435/DDI-435.md`
4. Extract any execution plan from comments and save to `.jira/DDI-435/plan.md`
5. Save as the "last ticket" for convenience

#### Fetch wiki page content

```bash
didi get https://evolutioniq.atlassian.net/wiki/spaces/ENG/pages/562692754/Product+Repositories
```

Fetches Confluence wiki pages and displays the content in your terminal. Uses the same Jira API credentials.

#### Save execution plan to Jira

```bash
didi save          # Uses last opened ticket
didi save DDI-123  # Specify ticket ID
```

Reads `.jira/{TICKET-ID}/plan.md` and posts it as a comment to the Jira ticket. This syncs your local plan changes back to Jira.

#### Interactive Kanban Board

```bash
didi kanban
```

Opens an interactive kanban board showing all tickets in your active sprint with keyboard navigation.

### Example Output

**Open command:**
```
✓ Workspace ready: .jira/DDI-435
```

**Save command:**
```
✓ Plan synced to Jira ticket DDI-123
View at: https://your-domain.atlassian.net/browse/DDI-123
```

## Claude Code Integration

After running `didi init`, you can use the `/didi` command in Claude Code:

### View ticket details
```
/didi desc DDI-123
```

### Open ticket workspace
```
/didi open DDI-456
```

### Generate implementation plan
```
/didi plan              # Uses context to determine ticket
/didi plan DDI-456      # Explicit ticket ID
```

Claude will analyze the codebase and generate a detailed technical implementation plan focused on code changes.

**Ticket resolution:** If no ticket ID is provided, Claude will check:
1. `.jira/.last-ticket` for the most recently opened ticket
2. Recent conversation for mentioned ticket IDs
3. Available workspaces in `.jira/` directory

The plan includes:
- Technical implementation steps with code examples
- Testing strategy (unit, integration, e2e tests)
- Test files to create/modify
- Data flow diagrams
- Files changed summary

**Focus:** Plans are optimized for creating quality PRs, not project management. They exclude PR checklists, rollout plans, risks & mitigations, and success criteria.

### Working with execution plans

Once you've opened a ticket, Claude Code will automatically know to reference execution plans at `.jira/{TICKET-ID}/plan.md` when you ask to:
- "Update the execution plan"
- "Show me the plan"
- "Add a step to the plan"

Claude will use conversation context to determine which ticket you're working on.

## Project Structure

```
didi-cli/
├── main.go           # Entry point
├── cmd/
│   ├── root.go       # Root command
│   ├── list.go       # List command - non-interactive ticket list
│   ├── open.go       # Open command implementation
│   ├── desc.go       # Desc command implementation
│   ├── save.go       # Save command - sync plan to Jira
│   ├── kanban.go     # Kanban command - interactive board
│   └── init.go       # Init command - install skill & check env
└── .jira/            # Created workspaces (gitignored)
    ├── .last-ticket  # Tracks last opened ticket
    └── DDI-XXX/
        ├── DDI-XXX.md
        └── plan.md
```
