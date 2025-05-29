#!/bin/bash

# PR Analysis & CI/CD Automation System - Setup Script
# This script automates the complete setup process

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="/tmp/pr-analyzer-setup.log"

# Default values
DEFAULT_DB_PASSWORD="secure_password_123"
DEFAULT_WEBHOOK_SECRET="$(openssl rand -hex 32)"
DEFAULT_JWT_SECRET="$(openssl rand -hex 64)"

# Functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_FILE"
    exit 1
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$LOG_FILE"
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        error "$1 is not installed. Please install it first."
    fi
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if running on supported OS
    if [[ "$OSTYPE" != "linux-gnu"* ]] && [[ "$OSTYPE" != "darwin"* ]]; then
        error "This script only supports Linux and macOS"
    fi
    
    # Check required commands
    check_command "curl"
    check_command "git"
    check_command "openssl"
    
    # Check if running in WSL (for Windows users)
    if grep -qi microsoft /proc/version 2>/dev/null; then
        info "Detected WSL environment"
        export IS_WSL=true
    fi
    
    log "Prerequisites check completed"
}

install_system_dependencies() {
    log "Installing system dependencies..."
    
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Update package list
        sudo apt update
        
        # Install essential packages
        sudo apt install -y \
            curl \
            wget \
            git \
            build-essential \
            software-properties-common \
            apt-transport-https \
            ca-certificates \
            gnupg \
            lsb-release
            
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        # Check if Homebrew is installed
        if ! command -v brew &> /dev/null; then
            info "Installing Homebrew..."
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        fi
        
        # Install essential packages
        brew install curl wget git openssl
    fi
    
    log "System dependencies installed"
}

install_nodejs() {
    log "Installing Node.js and pnpm..."
    
    if ! command -v node &> /dev/null; then
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            # Install Node.js 18.x
            curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
            sudo apt install -y nodejs
        elif [[ "$OSTYPE" == "darwin"* ]]; then
            brew install node@18
        fi
    fi
    
    # Install pnpm
    if ! command -v pnpm &> /dev/null; then
        npm install -g pnpm
    fi
    
    log "Node.js $(node --version) and pnpm $(pnpm --version) installed"
}

install_python() {
    log "Installing Python and pip..."
    
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        sudo apt install -y python3 python3-pip python3-venv python3-dev
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        brew install python@3.11
    fi
    
    # Upgrade pip
    python3 -m pip install --upgrade pip
    
    log "Python $(python3 --version) installed"
}

install_go() {
    log "Installing Go..."
    
    if ! command -v go &> /dev/null; then
        GO_VERSION="1.21.0"
        
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            wget "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
            sudo tar -C /usr/local -xzf /tmp/go.tar.gz
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            export PATH=$PATH:/usr/local/go/bin
        elif [[ "$OSTYPE" == "darwin"* ]]; then
            brew install go
        fi
    fi
    
    log "Go $(go version | cut -d' ' -f3) installed"
}

install_postgresql() {
    log "Installing PostgreSQL..."
    
    if ! command -v psql &> /dev/null; then
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            # Install PostgreSQL 17
            sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
            wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
            sudo apt update
            sudo apt install -y postgresql-17 postgresql-contrib-17
        elif [[ "$OSTYPE" == "darwin"* ]]; then
            brew install postgresql@17
            brew services start postgresql@17
        fi
    fi
    
    log "PostgreSQL installed"
}

setup_database() {
    log "Setting up database..."
    
    # Start PostgreSQL service
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        sudo systemctl start postgresql
        sudo systemctl enable postgresql
    fi
    
    # Create database and user
    DB_PASSWORD="${DB_PASSWORD:-$DEFAULT_DB_PASSWORD}"
    
    sudo -u postgres psql << EOF
CREATE DATABASE IF NOT EXISTS pr_analysis;
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'pr_analyzer') THEN
        CREATE USER pr_analyzer WITH PASSWORD '$DB_PASSWORD';
    END IF;
END
\$\$;
GRANT ALL PRIVILEGES ON DATABASE pr_analysis TO pr_analyzer;
\c pr_analysis
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
EOF
    
    # Set database URL
    export DATABASE_URL="postgresql://pr_analyzer:$DB_PASSWORD@localhost:5432/pr_analysis"
    echo "export DATABASE_URL=\"$DATABASE_URL\"" >> ~/.bashrc
    
    # Test connection
    if psql "$DATABASE_URL" -c "SELECT version();" &> /dev/null; then
        log "Database setup completed successfully"
    else
        error "Database connection test failed"
    fi
}

install_agentapi() {
    log "Installing AgentAPI..."
    
    # Create directory
    mkdir -p ~/tools/agentapi
    cd ~/tools/agentapi
    
    # Download latest release
    LATEST_VERSION=$(curl -s https://api.github.com/repos/Zeeeepa/agentapi/releases/latest | grep tag_name | cut -d '"' -f 4)
    
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        BINARY_NAME="agentapi-linux-amd64"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        BINARY_NAME="agentapi-darwin-amd64"
    fi
    
    curl -L "https://github.com/Zeeeepa/agentapi/releases/download/${LATEST_VERSION}/${BINARY_NAME}" -o agentapi
    chmod +x agentapi
    
    # Add to PATH
    sudo ln -sf "$(pwd)/agentapi" /usr/local/bin/agentapi
    
    # Verify installation
    if agentapi --help &> /dev/null; then
        log "AgentAPI installed successfully"
    else
        error "AgentAPI installation failed"
    fi
}

install_claude_code() {
    log "Installing Claude Code..."
    
    # Install via pip
    python3 -m pip install claude-code
    
    # Verify installation
    if claude --version &> /dev/null; then
        log "Claude Code installed successfully"
    else
        warn "Claude Code installation may have failed. Please check manually."
    fi
}

setup_analysis_tools() {
    log "Setting up analysis tools..."
    
    # TypeScript analysis tools
    npm install -g typescript @typescript-eslint/parser eslint
    
    # Python analysis tools
    python3 -m pip install pylint mypy bandit safety
    
    # Security analysis tools
    python3 -m pip install semgrep
    
    # Performance analysis tools
    npm install -g clinic autocannon
    
    log "Analysis tools installed"
}

create_configuration() {
    log "Creating configuration files..."
    
    cd "$PROJECT_ROOT"
    
    # Create config directory if it doesn't exist
    mkdir -p config
    
    # Generate secrets
    WEBHOOK_SECRET="${WEBHOOK_SECRET:-$DEFAULT_WEBHOOK_SECRET}"
    JWT_SECRET="${JWT_SECRET:-$DEFAULT_JWT_SECRET}"
    
    # Create environment file
    cat > .env << EOF
# Database Configuration
DATABASE_URL=$DATABASE_URL

# Webhook Configuration
WEBHOOK_SECRET=$WEBHOOK_SECRET

# JWT Configuration
JWT_SECRET=$JWT_SECRET

# AgentAPI Configuration
AGENTAPI_URL=http://localhost:3284

# API Keys (to be filled by user)
ANTHROPIC_API_KEY=
LINEAR_API_KEY=
GITHUB_TOKEN=

# Feature Flags
FEATURE_AUTO_FIX=true
FEATURE_LINEAR_INTEGRATION=true
FEATURE_GITHUB_INTEGRATION=true

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
EOF
    
    # Copy example configurations
    if [[ -f "config/production.yml" ]]; then
        cp config/production.yml config/local.yml
    fi
    
    log "Configuration files created"
}

setup_systemd_services() {
    if [[ "$OSTYPE" != "linux-gnu"* ]]; then
        warn "Systemd services are only available on Linux"
        return
    fi
    
    log "Setting up systemd services..."
    
    # AgentAPI service
    sudo tee /etc/systemd/system/agentapi.service > /dev/null << EOF
[Unit]
Description=AgentAPI Service
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$PROJECT_ROOT
ExecStart=/usr/local/bin/agentapi server -- claude
Restart=always
RestartSec=10
Environment=ANTHROPIC_API_KEY=\${ANTHROPIC_API_KEY}

[Install]
WantedBy=multi-user.target
EOF
    
    # Webhook service
    sudo tee /etc/systemd/system/pr-webhook.service > /dev/null << EOF
[Unit]
Description=PR Analysis Webhook Service
After=network.target postgresql.service

[Service]
Type=simple
User=$USER
WorkingDirectory=$PROJECT_ROOT
ExecStart=$PROJECT_ROOT/bin/webhook-server
Restart=always
RestartSec=10
EnvironmentFile=$PROJECT_ROOT/.env

[Install]
WantedBy=multi-user.target
EOF
    
    # Reload systemd
    sudo systemctl daemon-reload
    
    log "Systemd services created (not enabled yet)"
}

build_project() {
    log "Building project..."
    
    cd "$PROJECT_ROOT"
    
    # Install Go dependencies
    if [[ -f "go.mod" ]]; then
        go mod download
        go build -o bin/webhook-server ./cmd/webhook
        go build -o bin/analysis-engine ./cmd/analysis
    fi
    
    # Install Node.js dependencies (if package.json exists)
    if [[ -f "package.json" ]]; then
        pnpm install
        pnpm build
    fi
    
    log "Project built successfully"
}

run_tests() {
    log "Running tests..."
    
    cd "$PROJECT_ROOT"
    
    # Run Go tests
    if [[ -f "go.mod" ]]; then
        go test ./... -v
    fi
    
    # Run Node.js tests
    if [[ -f "package.json" ]]; then
        pnpm test
    fi
    
    log "Tests completed"
}

setup_monitoring() {
    log "Setting up monitoring..."
    
    # Create log directory
    sudo mkdir -p /var/log/pr-analyzer
    sudo chown "$USER:$USER" /var/log/pr-analyzer
    
    # Setup log rotation
    sudo tee /etc/logrotate.d/pr-analyzer > /dev/null << EOF
/var/log/pr-analyzer/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 $USER $USER
    postrotate
        systemctl reload pr-webhook agentapi 2>/dev/null || true
    endscript
}
EOF
    
    log "Monitoring setup completed"
}

print_next_steps() {
    log "Setup completed successfully!"
    
    echo ""
    echo -e "${GREEN}🎉 PR Analysis & CI/CD Automation System Setup Complete!${NC}"
    echo ""
    echo -e "${BLUE}Next Steps:${NC}"
    echo "1. Set your API keys in the .env file:"
    echo "   - ANTHROPIC_API_KEY (get from https://console.anthropic.com/)"
    echo "   - LINEAR_API_KEY (get from https://linear.app/settings/api)"
    echo "   - GITHUB_TOKEN (get from https://github.com/settings/tokens)"
    echo ""
    echo "2. Start the services:"
    echo "   sudo systemctl enable agentapi pr-webhook"
    echo "   sudo systemctl start agentapi pr-webhook"
    echo ""
    echo "3. Verify the installation:"
    echo "   curl http://localhost:3284/status"
    echo "   curl http://localhost:8080/health"
    echo ""
    echo "4. Configure GitHub webhooks:"
    echo "   Use the webhook URL: http://your-domain.com:8080/webhook"
    echo "   Secret: $WEBHOOK_SECRET"
    echo ""
    echo -e "${YELLOW}Configuration files:${NC}"
    echo "   - Main config: $PROJECT_ROOT/config/local.yml"
    echo "   - Environment: $PROJECT_ROOT/.env"
    echo "   - Logs: /var/log/pr-analyzer/"
    echo ""
    echo -e "${GREEN}For detailed documentation, see:${NC}"
    echo "   - Setup Guide: $PROJECT_ROOT/docs/SETUP_GUIDE.md"
    echo "   - API Reference: $PROJECT_ROOT/docs/API_REFERENCE.md"
    echo "   - Architecture: $PROJECT_ROOT/docs/ARCHITECTURE.md"
    echo ""
}

# Main execution
main() {
    log "Starting PR Analysis & CI/CD Automation System setup..."
    
    # Create log file
    touch "$LOG_FILE"
    
    # Run setup steps
    check_prerequisites
    install_system_dependencies
    install_nodejs
    install_python
    install_go
    install_postgresql
    setup_database
    install_agentapi
    install_claude_code
    setup_analysis_tools
    create_configuration
    setup_systemd_services
    build_project
    run_tests
    setup_monitoring
    
    print_next_steps
}

# Handle script interruption
trap 'error "Setup interrupted by user"' INT TERM

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --db-password)
            DB_PASSWORD="$2"
            shift 2
            ;;
        --webhook-secret)
            WEBHOOK_SECRET="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-tests        Skip running tests"
            echo "  --db-password       Set database password (default: $DEFAULT_DB_PASSWORD)"
            echo "  --webhook-secret    Set webhook secret (default: auto-generated)"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Run main function
main "$@"

