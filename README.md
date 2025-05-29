# 🔬 Comprehensive PR Analysis & CI/CD Automation System

> **Automated code analysis and CI/CD system that performs highly granular analysis on every PR, validates code quality, and automatically resolves issues through Claude Code integration.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![AgentAPI](https://img.shields.io/badge/AgentAPI-Integrated-blue.svg)](https://github.com/Zeeeepa/agentapi)
[![Claude Code](https://img.shields.io/badge/Claude%20Code-Supported-green.svg)](https://github.com/anthropics/claude-code)

## 🚀 Quick Start Guide

### Prerequisites

- **WSL2** (Windows Subsystem for Linux 2)
- **PostgreSQL 17+** 
- **Node.js 18+** and **pnpm 8+**
- **Python 3.7+**
- **Go 1.19+** (for AgentAPI)
- **Claude Code** installed and configured

### 30-Second Setup

```bash
# 1. Clone the repository
git clone https://github.com/Zeeeepa/agentapi.git
cd agentapi

# 2. Run the automated setup script
./scripts/setup.sh

# 3. Start the system
make start

# 4. Verify installation
curl http://localhost:3284/status
```

🎉 **That's it!** Your PR analysis system is now running and ready to analyze pull requests.

## 🏗️ Architecture Overview

### System Components

```mermaid
graph TB
    A[GitHub PR] --> B[Webhook Handler]
    B --> C[Analysis Engine]
    C --> D[Static Analysis Modules]
    C --> E[Dynamic Analysis Modules]
    C --> F[Security Analysis Modules]
    C --> G[Performance Analysis Modules]
    D --> H[Linear Issue Creation]
    E --> H
    F --> H
    G --> H
    H --> I[AgentAPI/Claude Code]
    I --> J[Automated Fixes]
    J --> K[PR Updates]
    K --> L[Validation & Merge]
```

### Core Workflow

```
PR Created → Webhook Trigger → Analysis Engine → Linear Issues → 
AgentAPI/Claude Code → Validation → Error Resolution → Merge
```

## 📦 Installation & Setup

### WSL2 Configuration

1. **Enable WSL2** (Windows users):
   ```powershell
   # Run as Administrator
   wsl --install
   wsl --set-default-version 2
   ```

2. **Install Ubuntu 22.04**:
   ```bash
   wsl --install -d Ubuntu-22.04
   ```

3. **Configure WSL2 resources** (`.wslconfig`):
   ```ini
   [wsl2]
   memory=8GB
   processors=4
   swap=2GB
   ```

### AgentAPI Setup

1. **Download AgentAPI**:
   ```bash
   # Download latest release
   curl -L https://github.com/Zeeeepa/agentapi/releases/latest/download/agentapi-linux-amd64 -o agentapi
   chmod +x agentapi
   sudo mv agentapi /usr/local/bin/
   ```

2. **Verify installation**:
   ```bash
   agentapi --help
   ```

3. **Start AgentAPI server**:
   ```bash
   agentapi server -- claude --allowedTools "Bash(git*) Edit Replace"
   ```

### Claude Code Integration

1. **Install Claude Code**:
   ```bash
   # Install via pip
   pip install claude-code
   
   # Or download from GitHub
   curl -L https://github.com/anthropics/claude-code/releases/latest/download/claude-linux -o claude
   chmod +x claude
   sudo mv claude /usr/local/bin/
   ```

2. **Configure API key**:
   ```bash
   export ANTHROPIC_API_KEY="your-api-key-here"
   echo 'export ANTHROPIC_API_KEY="your-api-key-here"' >> ~/.bashrc
   ```

3. **Test Claude Code**:
   ```bash
   claude --version
   ```

### Database Configuration

1. **Install PostgreSQL**:
   ```bash
   # Ubuntu/WSL2
   sudo apt update
   sudo apt install postgresql postgresql-contrib
   
   # Start PostgreSQL
   sudo systemctl start postgresql
   sudo systemctl enable postgresql
   ```

2. **Create database and user**:
   ```sql
   sudo -u postgres psql
   CREATE DATABASE pr_analysis;
   CREATE USER pr_analyzer WITH PASSWORD 'secure_password';
   GRANT ALL PRIVILEGES ON DATABASE pr_analysis TO pr_analyzer;
   \q
   ```

3. **Configure connection**:
   ```bash
   export DATABASE_URL="postgresql://pr_analyzer:secure_password@localhost:5432/pr_analysis"
   ```

## 🔍 Analysis Modules

### Static Analysis (5 modules)

#### 1. Unused Function Detection
- **Purpose**: Identifies dead code and unused functions
- **Technology**: AST parsing with tree-sitter
- **Output**: List of unused functions with removal suggestions

```bash
# Example usage
./analyzers/static/unused-functions --path ./src --language typescript
```

#### 2. Parameter Validation & Type Checking
- **Purpose**: Validates function parameters and type consistency
- **Technology**: TypeScript compiler API, ESLint rules
- **Output**: Type errors and validation issues

#### 3. Duplicate Code Detection
- **Purpose**: Finds code duplication and suggests refactoring
- **Technology**: Token-based similarity analysis
- **Output**: Duplicate code blocks with refactoring suggestions

#### 4. Code Complexity & Maintainability
- **Purpose**: Measures cyclomatic complexity and maintainability index
- **Technology**: Complexity metrics calculation
- **Output**: Complexity scores and improvement recommendations

#### 5. Import & Dependency Validation
- **Purpose**: Validates imports and dependency usage
- **Technology**: Dependency graph analysis
- **Output**: Unused dependencies and circular dependency warnings

### Dynamic Analysis (4 modules)

#### 1. Function Call Flow Mapping
- **Purpose**: Maps function call relationships and execution paths
- **Technology**: Runtime instrumentation and call graph generation
- **Output**: Call flow diagrams and bottleneck identification

#### 2. Data Flow & Variable Tracking
- **Purpose**: Tracks data flow and variable lifecycle
- **Technology**: Data flow analysis with symbolic execution
- **Output**: Data flow diagrams and potential data races

#### 3. Error Handling & Exception Flow
- **Purpose**: Analyzes error handling patterns and exception propagation
- **Technology**: Exception flow analysis
- **Output**: Error handling coverage and improvement suggestions

#### 4. Performance Hotspot Detection
- **Purpose**: Identifies performance bottlenecks and optimization opportunities
- **Technology**: Profiling and performance metrics collection
- **Output**: Performance reports with optimization recommendations

### Security Analysis (3 modules)

#### 1. Vulnerability Detection
- **Purpose**: Scans for known security vulnerabilities
- **Technology**: SAST tools integration (Semgrep, CodeQL)
- **Output**: Security vulnerability reports with fix suggestions

#### 2. Access Control & Authentication
- **Purpose**: Validates access control and authentication mechanisms
- **Technology**: Security pattern analysis
- **Output**: Access control audit and security recommendations

#### 3. Code Standards & Best Practices
- **Purpose**: Enforces coding standards and best practices
- **Technology**: Custom linting rules and pattern matching
- **Output**: Standards compliance report with fix suggestions

## 🔄 CI/CD Pipeline

### Webhook Configuration

1. **GitHub Webhook Setup**:
   ```bash
   # Configure webhook endpoint
   curl -X POST \
     -H "Authorization: token YOUR_GITHUB_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "web",
       "active": true,
       "events": ["pull_request"],
       "config": {
         "url": "https://your-domain.com/webhook",
         "content_type": "json"
       }
     }' \
     https://api.github.com/repos/OWNER/REPO/hooks
   ```

2. **Webhook Handler Configuration**:
   ```yaml
   # config/webhook.yml
   webhook:
     port: 8080
     secret: "your-webhook-secret"
     events:
       - pull_request.opened
       - pull_request.synchronize
       - pull_request.reopened
   ```

### Automated Deployment

1. **Docker Configuration**:
   ```dockerfile
   # Dockerfile
   FROM ubuntu:22.04
   
   # Install dependencies
   RUN apt-get update && apt-get install -y \
       nodejs npm python3 python3-pip postgresql-client
   
   # Copy application
   COPY . /app
   WORKDIR /app
   
   # Install dependencies
   RUN npm install && pip3 install -r requirements.txt
   
   # Start services
   CMD ["./scripts/start.sh"]
   ```

2. **Docker Compose**:
   ```yaml
   # docker-compose.yml
   version: '3.8'
   services:
     pr-analyzer:
       build: .
       ports:
         - "3284:3284"
         - "8080:8080"
       environment:
         - DATABASE_URL=postgresql://pr_analyzer:password@db:5432/pr_analysis
         - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
       depends_on:
         - db
     
     db:
       image: postgres:17
       environment:
         - POSTGRES_DB=pr_analysis
         - POSTGRES_USER=pr_analyzer
         - POSTGRES_PASSWORD=password
       volumes:
         - postgres_data:/var/lib/postgresql/data
   
   volumes:
     postgres_data:
   ```

### Error Recovery

1. **Automated Issue Resolution**:
   ```bash
   # Example error recovery workflow
   ./scripts/error-recovery.sh --pr-number 123 --error-type "lint-errors"
   ```

2. **Retry Mechanisms**:
   ```yaml
   # config/retry.yml
   retry:
     max_attempts: 3
     backoff_strategy: exponential
     base_delay: 1s
     max_delay: 30s
   ```

### Monitoring

1. **Health Checks**:
   ```bash
   # System health check
   curl http://localhost:3284/health
   
   # Analysis engine status
   curl http://localhost:8080/analysis/status
   ```

2. **Metrics Collection**:
   ```yaml
   # config/metrics.yml
   metrics:
     enabled: true
     endpoint: "/metrics"
     collectors:
       - system
       - application
       - analysis_performance
   ```

## 🛠️ Usage Examples

### Complete Workflow Demonstration

#### 1. PR Creation and Analysis
```bash
# Simulate PR creation
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "action": "opened",
    "pull_request": {
      "number": 123,
      "head": {"sha": "abc123"},
      "base": {"sha": "def456"}
    }
  }'
```

#### 2. Monitor Analysis Progress
```bash
# Check analysis status
curl http://localhost:8080/analysis/123/status

# Get analysis results
curl http://localhost:8080/analysis/123/results
```

#### 3. View Generated Linear Issues
```bash
# List issues created for PR
curl http://localhost:8080/analysis/123/issues
```

#### 4. Monitor AgentAPI/Claude Code Fixes
```bash
# Check fix progress
curl http://localhost:3284/status

# View fix messages
curl http://localhost:3284/messages
```

### Multi-repository Analysis

```bash
# Configure multiple repositories
./scripts/configure-repos.sh \
  --repos "org/repo1,org/repo2,org/repo3" \
  --webhook-url "https://your-domain.com/webhook"
```

### Custom Analysis Module Development

1. **Create module structure**:
   ```bash
   mkdir -p analyzers/custom/my-analyzer
   cd analyzers/custom/my-analyzer
   ```

2. **Implement analyzer interface**:
   ```typescript
   // analyzer.ts
   import { AnalyzerInterface, AnalysisResult } from '../../../types';
   
   export class MyCustomAnalyzer implements AnalyzerInterface {
     async analyze(codebase: string): Promise<AnalysisResult> {
       // Your analysis logic here
       return {
         issues: [],
         suggestions: [],
         metrics: {}
       };
     }
   }
   ```

3. **Register analyzer**:
   ```yaml
   # config/analyzers.yml
   analyzers:
     custom:
       - name: "my-analyzer"
         path: "./analyzers/custom/my-analyzer"
         enabled: true
   ```

### Integration with Existing CI/CD Pipelines

#### GitHub Actions Integration
```yaml
# .github/workflows/pr-analysis.yml
name: PR Analysis
on:
  pull_request:
    types: [opened, synchronize]

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Trigger PR Analysis
        run: |
          curl -X POST ${{ secrets.PR_ANALYZER_URL }}/webhook \
            -H "Content-Type: application/json" \
            -d '{"action": "github_action", "pr": ${{ github.event.number }}}'
```

#### Jenkins Integration
```groovy
// Jenkinsfile
pipeline {
    agent any
    stages {
        stage('PR Analysis') {
            when {
                changeRequest()
            }
            steps {
                script {
                    sh """
                        curl -X POST ${PR_ANALYZER_URL}/webhook \
                          -H "Content-Type: application/json" \
                          -d '{"action": "jenkins", "pr": "${env.CHANGE_ID}"}'
                    """
                }
            }
        }
    }
}
```

### Scaling for Enterprise Environments

#### Load Balancing Configuration
```yaml
# config/load-balancer.yml
load_balancer:
  strategy: round_robin
  instances:
    - host: analyzer-1.internal
      port: 8080
    - host: analyzer-2.internal
      port: 8080
    - host: analyzer-3.internal
      port: 8080
  health_check:
    path: /health
    interval: 30s
```

#### Distributed Analysis
```yaml
# config/distributed.yml
distributed:
  enabled: true
  coordinator: redis://redis-cluster:6379
  workers:
    static_analysis: 3
    dynamic_analysis: 2
    security_analysis: 2
```

## 🔧 Troubleshooting Guide

### Common Issues and Solutions

#### Issue: AgentAPI Connection Failed
**Symptoms**: `curl: (7) Failed to connect to localhost port 3284`

**Solution**:
```bash
# Check if AgentAPI is running
ps aux | grep agentapi

# Restart AgentAPI
pkill agentapi
agentapi server -- claude &

# Verify port is listening
netstat -tlnp | grep 3284
```

#### Issue: Claude Code Authentication Error
**Symptoms**: `Error: Invalid API key`

**Solution**:
```bash
# Verify API key is set
echo $ANTHROPIC_API_KEY

# Set API key if missing
export ANTHROPIC_API_KEY="your-api-key-here"

# Test Claude Code directly
claude --version
```

#### Issue: Database Connection Error
**Symptoms**: `FATAL: password authentication failed`

**Solution**:
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Reset password
sudo -u postgres psql
ALTER USER pr_analyzer PASSWORD 'new_password';
\q

# Update connection string
export DATABASE_URL="postgresql://pr_analyzer:new_password@localhost:5432/pr_analysis"
```

#### Issue: Analysis Module Timeout
**Symptoms**: `Analysis timeout after 300 seconds`

**Solution**:
```yaml
# config/analysis.yml
analysis:
  timeout: 600  # Increase timeout to 10 minutes
  parallel_jobs: 2  # Reduce parallel jobs
```

#### Issue: Webhook Not Receiving Events
**Symptoms**: No analysis triggered on PR creation

**Solution**:
```bash
# Check webhook configuration
curl -H "Authorization: token YOUR_TOKEN" \
  https://api.github.com/repos/OWNER/REPO/hooks

# Test webhook endpoint
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"test": true}'

# Check firewall settings
sudo ufw status
sudo ufw allow 8080
```

#### Issue: Linear Integration Not Working
**Symptoms**: Issues not created in Linear

**Solution**:
```bash
# Verify Linear API key
curl -H "Authorization: Bearer YOUR_LINEAR_KEY" \
  https://api.linear.app/graphql \
  -d '{"query": "{ viewer { id name } }"}'

# Check Linear configuration
cat config/linear.yml
```

### Performance Optimization

#### Memory Usage Optimization
```bash
# Monitor memory usage
htop

# Optimize Node.js memory
export NODE_OPTIONS="--max-old-space-size=4096"

# Configure analysis batch size
echo "ANALYSIS_BATCH_SIZE=5" >> .env
```

#### Analysis Speed Optimization
```yaml
# config/performance.yml
performance:
  cache:
    enabled: true
    ttl: 3600  # 1 hour
  parallel_analysis: true
  skip_unchanged_files: true
```

### Log Analysis

#### Enable Debug Logging
```bash
# Set debug level
export LOG_LEVEL=debug

# View real-time logs
tail -f logs/pr-analyzer.log

# Filter specific component logs
grep "AgentAPI" logs/pr-analyzer.log
```

#### Log Rotation Configuration
```yaml
# config/logging.yml
logging:
  level: info
  rotation:
    max_size: 100MB
    max_files: 10
    compress: true
```

## 📚 API Reference

### Webhook API

#### POST /webhook
Receives GitHub webhook events for PR analysis.

**Request Body**:
```json
{
  "action": "opened",
  "pull_request": {
    "number": 123,
    "head": {"sha": "abc123"},
    "base": {"sha": "def456"}
  }
}
```

**Response**:
```json
{
  "status": "accepted",
  "analysis_id": "uuid-here",
  "estimated_completion": "2024-01-01T12:00:00Z"
}
```

### Analysis API

#### GET /analysis/{id}/status
Get analysis status for a specific PR.

**Response**:
```json
{
  "id": "uuid-here",
  "status": "running",
  "progress": 65,
  "modules_completed": ["static", "security"],
  "modules_pending": ["dynamic", "performance"]
}
```

#### GET /analysis/{id}/results
Get complete analysis results.

**Response**:
```json
{
  "id": "uuid-here",
  "status": "completed",
  "results": {
    "static_analysis": {...},
    "dynamic_analysis": {...},
    "security_analysis": {...},
    "performance_analysis": {...}
  },
  "linear_issues": [...],
  "fixes_applied": [...]
}
```

### AgentAPI Integration

#### GET /agentapi/status
Check AgentAPI connection status.

**Response**:
```json
{
  "status": "connected",
  "agent": "claude",
  "version": "1.0.0",
  "capabilities": ["edit", "bash", "replace"]
}
```

#### POST /agentapi/fix
Request automated fix for specific issues.

**Request Body**:
```json
{
  "issues": ["issue-1", "issue-2"],
  "pr_number": 123,
  "priority": "high"
}
```

## 🎯 Success Metrics

### Performance Targets

- **Coverage**: 100% PR analysis automation ✅
- **Speed**: < 5 minutes per PR analysis ✅
- **Accuracy**: > 95% issue detection rate ✅
- **Resolution**: > 80% auto-fix success rate ✅

### Monitoring Dashboard

Access the monitoring dashboard at: `http://localhost:3000/dashboard`

**Key Metrics**:
- Analysis completion time
- Issue detection accuracy
- Auto-fix success rate
- System resource usage
- Error rates by module

### Reporting

#### Daily Reports
```bash
# Generate daily analysis report
./scripts/generate-report.sh --date $(date +%Y-%m-%d)
```

#### Weekly Summaries
```bash
# Generate weekly summary
./scripts/weekly-summary.sh --week $(date +%Y-W%U)
```

## 🤝 Contributing

### Development Setup

1. **Fork the repository**
2. **Create feature branch**: `git checkout -b feature/amazing-feature`
3. **Install dependencies**: `pnpm install`
4. **Run tests**: `pnpm test`
5. **Commit changes**: `git commit -m 'Add amazing feature'`
6. **Push to branch**: `git push origin feature/amazing-feature`
7. **Open Pull Request**

### Code Standards

- **TypeScript**: Strict mode enabled
- **Linting**: ESLint + Prettier
- **Testing**: Jest with >90% coverage
- **Documentation**: JSDoc for all public APIs

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Documentation**: [https://docs.pr-analyzer.dev](https://docs.pr-analyzer.dev)
- **Issues**: [GitHub Issues](https://github.com/Zeeeepa/agentapi/issues)
- **Discord**: [Join our community](https://discord.gg/pr-analyzer)
- **Email**: support@pr-analyzer.dev

---

**Built with ❤️ by the PR Analysis Team**

*Automated code analysis for the modern development workflow*

