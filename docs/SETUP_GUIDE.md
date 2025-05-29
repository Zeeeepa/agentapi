# 🚀 Complete Setup Guide

This guide provides detailed step-by-step instructions for setting up the PR Analysis & CI/CD Automation System.

## 📋 Prerequisites Checklist

Before starting, ensure you have:

- [ ] Windows 10/11 with WSL2 enabled OR Linux Ubuntu 20.04+
- [ ] 8GB+ RAM available
- [ ] 20GB+ free disk space
- [ ] Internet connection for downloading dependencies
- [ ] GitHub account with repository access
- [ ] Anthropic API key for Claude Code

## 🔧 Detailed Installation Steps

### Step 1: WSL2 Setup (Windows Only)

#### 1.1 Enable WSL2
```powershell
# Run PowerShell as Administrator
dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart
dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart

# Restart your computer
shutdown /r /t 0
```

#### 1.2 Install WSL2 Kernel Update
1. Download the [WSL2 Linux kernel update package](https://wslstorestorage.blob.core.windows.net/wslblob/wsl_update_x64.msi)
2. Run the installer
3. Set WSL2 as default:
```powershell
wsl --set-default-version 2
```

#### 1.3 Install Ubuntu 22.04
```powershell
# Install from Microsoft Store or command line
wsl --install -d Ubuntu-22.04

# Launch Ubuntu and create user account
ubuntu2204.exe
```

#### 1.4 Configure WSL2 Resources
Create `.wslconfig` in your Windows user directory:
```ini
# C:\Users\YourUsername\.wslconfig
[wsl2]
memory=8GB
processors=4
swap=2GB
localhostForwarding=true
```

Restart WSL2:
```powershell
wsl --shutdown
wsl
```

### Step 2: System Dependencies

#### 2.1 Update System
```bash
sudo apt update && sudo apt upgrade -y
```

#### 2.2 Install Essential Tools
```bash
# Development tools
sudo apt install -y curl wget git build-essential

# Node.js and pnpm
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs
npm install -g pnpm

# Python and pip
sudo apt install -y python3 python3-pip python3-venv

# Go (for AgentAPI)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### Step 3: PostgreSQL Database Setup

#### 3.1 Install PostgreSQL
```bash
# Install PostgreSQL 17
sudo apt install -y postgresql-17 postgresql-contrib-17

# Start and enable PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### 3.2 Configure Database
```bash
# Switch to postgres user
sudo -u postgres psql

# Create database and user
CREATE DATABASE pr_analysis;
CREATE USER pr_analyzer WITH PASSWORD 'secure_password_123';
GRANT ALL PRIVILEGES ON DATABASE pr_analysis TO pr_analyzer;

# Enable necessary extensions
\c pr_analysis
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

\q
```

#### 3.3 Configure Connection
```bash
# Add to ~/.bashrc
echo 'export DATABASE_URL="postgresql://pr_analyzer:secure_password_123@localhost:5432/pr_analysis"' >> ~/.bashrc
source ~/.bashrc

# Test connection
psql $DATABASE_URL -c "SELECT version();"
```

### Step 4: AgentAPI Installation

#### 4.1 Download AgentAPI
```bash
# Create directory for AgentAPI
mkdir -p ~/tools/agentapi
cd ~/tools/agentapi

# Download latest release
LATEST_VERSION=$(curl -s https://api.github.com/repos/Zeeeepa/agentapi/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/Zeeeepa/agentapi/releases/download/${LATEST_VERSION}/agentapi-linux-amd64" -o agentapi

# Make executable and add to PATH
chmod +x agentapi
sudo ln -s $(pwd)/agentapi /usr/local/bin/agentapi
```

#### 4.2 Verify Installation
```bash
agentapi --help
agentapi version
```

### Step 5: Claude Code Setup

#### 5.1 Install Claude Code
```bash
# Method 1: Using pip
pip3 install claude-code

# Method 2: Download binary (if available)
# curl -L https://github.com/anthropics/claude-code/releases/latest/download/claude-linux -o claude
# chmod +x claude
# sudo mv claude /usr/local/bin/
```

#### 5.2 Configure API Key
```bash
# Get your API key from https://console.anthropic.com/
read -s -p "Enter your Anthropic API key: " ANTHROPIC_API_KEY
echo

# Add to environment
echo "export ANTHROPIC_API_KEY=\"$ANTHROPIC_API_KEY\"" >> ~/.bashrc
source ~/.bashrc

# Verify configuration
claude --version
```

### Step 6: Project Setup

#### 6.1 Clone Repository
```bash
# Clone the main repository
git clone https://github.com/Zeeeepa/agentapi.git
cd agentapi

# Install dependencies
go mod download
```

#### 6.2 Build Project
```bash
# Build AgentAPI
make build

# Run tests
make test
```

#### 6.3 Configuration Files
```bash
# Create configuration directory
mkdir -p config

# Copy example configurations
cp config/example.yml config/production.yml
cp config/database.example.yml config/database.yml
```

Edit configuration files:

**config/production.yml**:
```yaml
server:
  host: "0.0.0.0"
  port: 3284
  
agent:
  type: "claude"
  timeout: 300
  
analysis:
  parallel_jobs: 4
  timeout: 600
  
webhook:
  port: 8080
  secret: "your-webhook-secret-here"
```

**config/database.yml**:
```yaml
database:
  url: "postgresql://pr_analyzer:secure_password_123@localhost:5432/pr_analysis"
  max_connections: 10
  ssl_mode: "disable"
```

### Step 7: Analysis Modules Setup

#### 7.1 Install Analysis Dependencies
```bash
# TypeScript analysis tools
npm install -g typescript @typescript-eslint/parser

# Python analysis tools
pip3 install pylint mypy bandit

# Security analysis tools
pip3 install semgrep safety

# Performance analysis tools
npm install -g clinic autocannon
```

#### 7.2 Configure Analysis Modules
```bash
# Create analyzers directory
mkdir -p analyzers/{static,dynamic,security,performance}

# Copy analyzer configurations
cp -r examples/analyzers/* analyzers/
```

### Step 8: Linear Integration Setup

#### 8.1 Get Linear API Key
1. Go to [Linear Settings](https://linear.app/settings/api)
2. Create a new API key
3. Copy the key

#### 8.2 Configure Linear Integration
```bash
# Add Linear API key to environment
read -s -p "Enter your Linear API key: " LINEAR_API_KEY
echo "export LINEAR_API_KEY=\"$LINEAR_API_KEY\"" >> ~/.bashrc
source ~/.bashrc
```

**config/linear.yml**:
```yaml
linear:
  api_key: "${LINEAR_API_KEY}"
  team_id: "your-team-id"
  project_id: "your-project-id"
  
issue_templates:
  static_analysis: "templates/static_analysis_issue.md"
  security_issue: "templates/security_issue.md"
  performance_issue: "templates/performance_issue.md"
```

### Step 9: GitHub Webhook Setup

#### 9.1 Configure Webhook Endpoint
```bash
# Install ngrok for local development (optional)
curl -s https://ngrok-agent.s3.amazonaws.com/ngrok.asc | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null
echo "deb https://ngrok-agent.s3.amazonaws.com buster main" | sudo tee /etc/apt/sources.list.d/ngrok.list
sudo apt update && sudo apt install ngrok

# Authenticate ngrok
ngrok config add-authtoken YOUR_NGROK_TOKEN

# Expose webhook endpoint
ngrok http 8080
```

#### 9.2 Add Webhook to GitHub Repository
```bash
# Using GitHub CLI (install if needed)
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update && sudo apt install gh

# Authenticate with GitHub
gh auth login

# Add webhook
gh api repos/OWNER/REPO/hooks \
  --method POST \
  --field name='web' \
  --field active=true \
  --field events='["pull_request"]' \
  --field config='{"url":"https://your-ngrok-url.ngrok.io/webhook","content_type":"json","secret":"your-webhook-secret"}'
```

### Step 10: Service Configuration

#### 10.1 Create Systemd Services
**AgentAPI Service** (`/etc/systemd/system/agentapi.service`):
```ini
[Unit]
Description=AgentAPI Service
After=network.target

[Service]
Type=simple
User=your-username
WorkingDirectory=/home/your-username/agentapi
ExecStart=/usr/local/bin/agentapi server -- claude
Restart=always
RestartSec=10
Environment=ANTHROPIC_API_KEY=your-api-key

[Install]
WantedBy=multi-user.target
```

**Webhook Service** (`/etc/systemd/system/pr-webhook.service`):
```ini
[Unit]
Description=PR Analysis Webhook Service
After=network.target postgresql.service

[Service]
Type=simple
User=your-username
WorkingDirectory=/home/your-username/agentapi
ExecStart=/home/your-username/agentapi/bin/webhook-server
Restart=always
RestartSec=10
Environment=DATABASE_URL=postgresql://pr_analyzer:secure_password_123@localhost:5432/pr_analysis
Environment=LINEAR_API_KEY=your-linear-api-key

[Install]
WantedBy=multi-user.target
```

#### 10.2 Enable and Start Services
```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable services
sudo systemctl enable agentapi
sudo systemctl enable pr-webhook

# Start services
sudo systemctl start agentapi
sudo systemctl start pr-webhook

# Check status
sudo systemctl status agentapi
sudo systemctl status pr-webhook
```

### Step 11: Verification and Testing

#### 11.1 Health Checks
```bash
# Check AgentAPI
curl http://localhost:3284/status

# Check webhook endpoint
curl http://localhost:8080/health

# Check database connection
psql $DATABASE_URL -c "SELECT 1;"
```

#### 11.2 Test Analysis Pipeline
```bash
# Create test PR webhook payload
cat > test_webhook.json << EOF
{
  "action": "opened",
  "pull_request": {
    "number": 1,
    "head": {"sha": "test123"},
    "base": {"sha": "main456"},
    "html_url": "https://github.com/test/repo/pull/1"
  },
  "repository": {
    "full_name": "test/repo",
    "clone_url": "https://github.com/test/repo.git"
  }
}
EOF

# Send test webhook
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=$(echo -n "$(cat test_webhook.json)" | openssl dgst -sha256 -hmac "your-webhook-secret" | cut -d' ' -f2)" \
  -d @test_webhook.json
```

#### 11.3 Monitor Logs
```bash
# View AgentAPI logs
sudo journalctl -u agentapi -f

# View webhook logs
sudo journalctl -u pr-webhook -f

# View application logs
tail -f logs/pr-analyzer.log
```

## 🔧 Troubleshooting Setup Issues

### Common Installation Problems

#### WSL2 Issues
```bash
# If WSL2 won't start
wsl --shutdown
wsl --unregister Ubuntu-22.04
wsl --install -d Ubuntu-22.04

# If memory issues occur
# Edit .wslconfig to reduce memory allocation
```

#### PostgreSQL Connection Issues
```bash
# Check if PostgreSQL is running
sudo systemctl status postgresql

# Reset PostgreSQL password
sudo -u postgres psql -c "ALTER USER pr_analyzer PASSWORD 'new_password';"

# Check connection settings
sudo nano /etc/postgresql/17/main/postgresql.conf
sudo nano /etc/postgresql/17/main/pg_hba.conf
```

#### AgentAPI Build Issues
```bash
# Update Go version
sudo rm -rf /usr/local/go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Clear Go module cache
go clean -modcache
go mod download
```

#### Claude Code Authentication Issues
```bash
# Verify API key format
echo $ANTHROPIC_API_KEY | wc -c  # Should be around 108 characters

# Test API key directly
curl -H "Authorization: Bearer $ANTHROPIC_API_KEY" \
  -H "Content-Type: application/json" \
  https://api.anthropic.com/v1/messages \
  -d '{"model":"claude-3-sonnet-20240229","max_tokens":10,"messages":[{"role":"user","content":"Hi"}]}'
```

### Performance Optimization

#### System Tuning
```bash
# Increase file descriptor limits
echo "* soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "* hard nofile 65536" | sudo tee -a /etc/security/limits.conf

# Optimize PostgreSQL
sudo nano /etc/postgresql/17/main/postgresql.conf
# Add:
# shared_buffers = 256MB
# effective_cache_size = 1GB
# work_mem = 4MB
```

#### Application Tuning
```bash
# Set Node.js memory limits
echo "export NODE_OPTIONS='--max-old-space-size=4096'" >> ~/.bashrc

# Configure Go garbage collector
echo "export GOGC=100" >> ~/.bashrc
```

## 🎯 Next Steps

After successful setup:

1. **Configure your first repository** for analysis
2. **Create test pull requests** to verify the pipeline
3. **Set up monitoring dashboards** for system health
4. **Configure backup strategies** for the database
5. **Set up log rotation** for application logs

## 📞 Getting Help

If you encounter issues during setup:

1. Check the [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Review system logs for error messages
3. Join our [Discord community](https://discord.gg/pr-analyzer)
4. Create an issue on [GitHub](https://github.com/Zeeeepa/agentapi/issues)

---

**Setup completed successfully? 🎉**

Continue to the [Usage Guide](USAGE_GUIDE.md) to start analyzing pull requests!

