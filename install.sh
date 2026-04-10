#!/bin/bash
set -e

# Didi CLI installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/EIQWeijian/didi-cli/main/install.sh | bash

INSTALL_DIR="${INSTALL_DIR:-$HOME/go/bin}"
VERSION="${VERSION:-latest}"
REPO="EIQWeijian/didi-cli"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}Installing didi CLI...${NC}"

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    *)
        echo -e "${RED}Unsupported operating system: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Construct download URL
if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/didi-${OS}-${ARCH}"
else
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/didi-${OS}-${ARCH}"
fi

echo -e "${CYAN}Detected: $OS $ARCH${NC}"
echo -e "${CYAN}Installing to: $INSTALL_DIR${NC}"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download binary
echo -e "${CYAN}Downloading didi...${NC}"
if command -v curl > /dev/null 2>&1; then
    curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/didi"
elif command -v wget > /dev/null 2>&1; then
    wget -q "$DOWNLOAD_URL" -O "$INSTALL_DIR/didi"
else
    echo -e "${RED}Error: curl or wget is required${NC}"
    exit 1
fi

# Make executable
chmod +x "$INSTALL_DIR/didi"

# Check if install directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo -e "Add this to your ~/.zshrc or ~/.bashrc:"
    echo ""
    echo -e "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
fi

# Verify installation
if "$INSTALL_DIR/didi" --help > /dev/null 2>&1; then
    echo ""
    echo -e "${GREEN}✓ didi installed successfully!${NC}"
    echo ""
    echo -e "${CYAN}Next steps:${NC}"
    echo ""
    echo "1. Configure Jira credentials:"
    echo ""
    echo "   export JIRA_API_TOKEN=\"your-api-token\""
    echo "   export JIRA_BASE_URL=\"https://your-domain.atlassian.net\""
    echo "   export JIRA_EMAIL=\"your-email@example.com\""
    echo ""
    echo "2. Initialize Claude Code skill (optional):"
    echo ""
    echo "   didi init"
    echo ""
    echo "3. Start using didi:"
    echo ""
    echo "   didi list           # List active sprint tickets"
    echo "   didi open DDI-123   # Open a ticket"
    echo ""
    echo "Get API token: https://id.atlassian.com/manage-profile/security/api-tokens"
    echo ""
else
    echo -e "${RED}Installation failed - binary not working${NC}"
    exit 1
fi
