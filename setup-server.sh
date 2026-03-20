#!/bin/bash
# go-ocs Linux Server Setup Script
# Run this directly on the server: bash setup-server.sh
# Installs: Go, gh CLI, Goose, Claude Code, GitHub Actions self-hosted runner

set -e

REPO_URL="https://github.com/eddiecarpenter/go-ocs"
RUNNER_VERSION="2.322.0"
WORKSPACE="/workspace"
RUNNER_DIR="/home/eddiecarpenter/actions-runner"

echo "======================================"
echo " go-ocs Server Setup"
echo "======================================"

# --- gh CLI ---
echo ""
echo ">>> Installing gh CLI..."
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt-get update -q
sudo apt-get install -y gh
echo "gh $(gh --version | head -1) installed"


# --- Go ---
echo ""
echo ">>> Installing Go..."
GO_VERSION="1.24.1"
curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz

# Add Go to PATH permanently
if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    echo 'export GOPATH=$HOME/go' >> ~/.bashrc
    echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
fi
export PATH=$PATH:/usr/local/go/bin
echo "Go $(go version) installed"

# --- Goose ---
echo ""
echo ">>> Installing Goose..."
curl -fsSL https://github.com/block/goose/releases/latest/download/goose_linux_amd64 -o /tmp/goose
chmod +x /tmp/goose
sudo mv /tmp/goose /usr/local/bin/goose
echo "Goose $(goose --version) installed"

# --- Claude Code ---
echo ""
echo ">>> Installing Claude Code..."
npm install -g @anthropic-ai/claude-code
echo "Claude Code $(claude --version) installed"


# --- Workspace directory ---
echo ""
echo ">>> Creating workspace directory..."
sudo mkdir -p $WORKSPACE
sudo chown eddiecarpenter:eddiecarpenter $WORKSPACE
echo "Workspace: $WORKSPACE"

# --- GitHub Actions self-hosted runner ---
echo ""
echo ">>> Installing GitHub Actions self-hosted runner..."
mkdir -p $RUNNER_DIR
cd $RUNNER_DIR

curl -fsSL "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz" -o actions-runner.tar.gz
tar xzf actions-runner.tar.gz
rm actions-runner.tar.gz

echo ""
echo "======================================"
echo " Setup complete!"
echo "======================================"
echo ""
echo "Next steps:"
echo ""
echo "1. Authenticate gh CLI:"
echo "   gh auth login"
echo ""
echo "2. Configure the GitHub Actions runner:"
echo "   cd $RUNNER_DIR"
echo "   ./config.sh --url $REPO_URL --token <RUNNER_TOKEN>"
echo ""
echo "   Get your runner token from:"
echo "   https://github.com/eddiecarpenter/go-ocs/settings/actions/runners/new"
echo ""
echo "3. Install runner as a service (starts on boot):"
echo "   sudo ./svc.sh install"
echo "   sudo ./svc.sh start"
echo ""
echo "4. Verify runner is online:"
echo "   https://github.com/eddiecarpenter/go-ocs/settings/actions/runners"
echo ""
echo "5. Configure Goose:"
echo "   mkdir -p ~/.config/goose"
echo "   cat > ~/.config/goose/config.yaml << 'EOF'"
echo "   GOOSE_PROVIDER: claude-code"
echo "   GOOSE_MODEL: default"
echo "   EOF"
echo ""
echo "6. Copy recipes from repo:"
echo "   cp ~/go-ocs/.ai/recipes/*.yaml ~/.config/goose/recipes/"
