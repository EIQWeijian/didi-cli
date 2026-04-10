# Didi CLI

A CLI tool for managing Jira tickets with local workspace. 
- Includes Claude Skill integration
- syncs implementation plans `plan.md` back to jira

<img width="1616" height="959" alt="Screenshot 2026-04-10 at 3 19 38 PM" src="https://github.com/user-attachments/assets/2642c1cd-7d62-4cc0-ba1d-12fc098530a9" />

## Quick Start

```bash
# 1. Install
go build -o ~/go/bin/didi

# 2. Configure environment variables
export JIRA_API_TOKEN="your-api-token"
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_EMAIL="your-email@example.com"

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
- Automatically extract execution plans from ticket comments
- Store ticket details in markdown format
- Sync local execution plan changes back to Jira as comments
- Track last opened ticket for quick operations
- Interactive kanban board for sprint management
- Claude Code integration with `/didi` skill for AI-powered workflow

## Installation

### Prerequisites

- Go 1.19 or later
- `~/go/bin` in your PATH (for global installation)

Add this to your `~/.zshrc` or `~/.bashrc` if not already present:
```bash
export PATH="$HOME/go/bin:$PATH"
```

Then reload your shell:
```bash
source ~/.zshrc  # or source ~/.bashrc
```

### Build and Install Globally

Clone the repository and build:

```bash
# Clone the repository
git clone git@github.com:EIQWeijian/didi-cli.git
cd didi-cli

# Build and install globally
go build -o ~/go/bin/didi

# Verify installation
which didi
# Should output: /Users/yourname/go/bin/didi
```

### Initialize didi

After building the binary, initialize didi to install the Claude Code skill and verify your environment:

```bash
didi init
```

This will:
- Check that required Jira environment variables are set
- Install the `/didi` skill to `~/.claude/skills/didi/`, making it available in all Claude Code sessions

### Alternative: Download Pre-built Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/EIQWeijian/didi-cli/releases):

**macOS (Apple Silicon):**
```bash
curl -L -o ~/go/bin/didi https://github.com/EIQWeijian/didi-cli/releases/latest/download/didi-darwin-arm64
chmod +x ~/go/bin/didi
```

**macOS (Intel):**
```bash
curl -L -o ~/go/bin/didi https://github.com/EIQWeijian/didi-cli/releases/latest/download/didi-darwin-amd64
chmod +x ~/go/bin/didi
```

**Linux:**
```bash
curl -L -o ~/go/bin/didi https://github.com/EIQWeijian/didi-cli/releases/latest/download/didi-linux-amd64
chmod +x ~/go/bin/didi
```

**Windows:**
Download [didi-windows-amd64.exe](https://github.com/EIQWeijian/didi-cli/releases/latest/download/didi-windows-amd64.exe) and add to your PATH.

## Configuration

Set the following environment variables:

```bash
export JIRA_API_TOKEN="your-jira-api-token"
export JIRA_BASE_URL="https://your-domain.atlassian.net"
export JIRA_EMAIL="your-email@example.com"
```

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
