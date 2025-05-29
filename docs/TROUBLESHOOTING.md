# 🔧 Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the PR Analysis & CI/CD Automation System.

## 🚨 Quick Diagnostics

### System Health Check

Run this comprehensive health check script:

```bash
#!/bin/bash
# health-check.sh

echo "🔍 PR Analyzer System Health Check"
echo "=================================="

# Check services
echo "📊 Service Status:"
curl -s http://localhost:8080/health | jq '.' || echo "❌ Webhook service down"
curl -s http://localhost:3284/status | jq '.' || echo "❌ AgentAPI service down"

# Check database
echo "🗄️ Database Status:"
psql $DATABASE_URL -c "SELECT version();" || echo "❌ Database connection failed"

# Check disk space
echo "💾 Disk Space:"
df -h | grep -E "(/$|/var|/tmp)"

# Check memory
echo "🧠 Memory Usage:"
free -h

# Check logs for errors
echo "📝 Recent Errors:"
tail -n 50 /var/log/pr-analyzer/app.log | grep -i error | tail -5
```

### Quick Status Commands

```bash
# Service status
systemctl status agentapi pr-webhook

# Port availability
netstat -tlnp | grep -E "(3284|8080)"

# Process status
ps aux | grep -E "(agentapi|webhook)"

# Log monitoring
tail -f /var/log/pr-analyzer/app.log
```

## 🔧 Common Issues & Solutions

### 1. AgentAPI Connection Issues

#### Symptom: `curl: (7) Failed to connect to localhost port 3284`

**Diagnosis:**
```bash
# Check if AgentAPI is running
ps aux | grep agentapi

# Check port binding
netstat -tlnp | grep 3284

# Check service status
systemctl status agentapi
```

**Solutions:**

**A. Service Not Running**
```bash
# Start AgentAPI service
sudo systemctl start agentapi

# Enable auto-start
sudo systemctl enable agentapi

# Check logs for startup errors
journalctl -u agentapi -f
```

**B. Port Already in Use**
```bash
# Find process using port 3284
lsof -i :3284

# Kill conflicting process
sudo kill -9 <PID>

# Restart AgentAPI
sudo systemctl restart agentapi
```

**C. Configuration Issues**
```bash
# Check configuration file
cat /etc/systemd/system/agentapi.service

# Verify environment variables
echo $ANTHROPIC_API_KEY

# Test AgentAPI manually
agentapi server -- claude --allowedTools "Bash(git*) Edit Replace"
```

### 2. Claude Code Authentication Errors

#### Symptom: `Error: Invalid API key` or `Authentication failed`

**Diagnosis:**
```bash
# Check API key format
echo $ANTHROPIC_API_KEY | wc -c  # Should be ~108 characters

# Test API key directly
curl -H "Authorization: Bearer $ANTHROPIC_API_KEY" \
  -H "Content-Type: application/json" \
  https://api.anthropic.com/v1/messages \
  -d '{"model":"claude-3-sonnet-20240229","max_tokens":10,"messages":[{"role":"user","content":"Hi"}]}'
```

**Solutions:**

**A. Invalid API Key**
```bash
# Get new API key from https://console.anthropic.com/
# Update environment variable
export ANTHROPIC_API_KEY="your-new-api-key"
echo 'export ANTHROPIC_API_KEY="your-new-api-key"' >> ~/.bashrc

# Update systemd service
sudo systemctl edit agentapi
# Add:
# [Service]
# Environment=ANTHROPIC_API_KEY=your-new-api-key

sudo systemctl restart agentapi
```

**B. API Key Not Set**
```bash
# Check if variable is set
env | grep ANTHROPIC

# Set in current session
export ANTHROPIC_API_KEY="your-api-key"

# Set permanently
echo 'export ANTHROPIC_API_KEY="your-api-key"' >> ~/.bashrc
source ~/.bashrc
```

**C. Network/Firewall Issues**
```bash
# Test connectivity to Anthropic API
curl -I https://api.anthropic.com

# Check firewall rules
sudo ufw status

# Allow outbound HTTPS
sudo ufw allow out 443
```

### 3. Database Connection Problems

#### Symptom: `FATAL: password authentication failed` or `Connection refused`

**Diagnosis:**
```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Test connection
psql $DATABASE_URL -c "SELECT 1;"

# Check connection string
echo $DATABASE_URL

# Check PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-17-main.log
```

**Solutions:**

**A. PostgreSQL Not Running**
```bash
# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Check if it's listening
netstat -tlnp | grep 5432
```

**B. Authentication Issues**
```bash
# Reset password
sudo -u postgres psql
ALTER USER pr_analyzer PASSWORD 'new_password';
\q

# Update connection string
export DATABASE_URL="postgresql://pr_analyzer:new_password@localhost:5432/pr_analysis"
echo 'export DATABASE_URL="postgresql://pr_analyzer:new_password@localhost:5432/pr_analysis"' >> ~/.bashrc
```

**C. Database Doesn't Exist**
```bash
# Create database
sudo -u postgres createdb pr_analysis

# Grant permissions
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE pr_analysis TO pr_analyzer;"
```

**D. Connection Limit Exceeded**
```bash
# Check current connections
sudo -u postgres psql -c "SELECT count(*) FROM pg_stat_activity;"

# Increase max connections (edit postgresql.conf)
sudo nano /etc/postgresql/17/main/postgresql.conf
# Set: max_connections = 200

sudo systemctl restart postgresql
```

### 4. Analysis Module Timeouts

#### Symptom: `Analysis timeout after 300 seconds`

**Diagnosis:**
```bash
# Check system resources
htop
df -h

# Check analysis logs
grep "timeout" /var/log/pr-analyzer/app.log

# Check module-specific logs
ls -la /var/log/pr-analyzer/modules/
```

**Solutions:**

**A. Increase Timeout**
```yaml
# config/local.yml
analysis:
  timeout: 900s  # Increase to 15 minutes
  modules:
    static_analysis:
      timeout: 600s
    dynamic_analysis:
      timeout: 1200s
```

**B. Reduce Parallel Jobs**
```yaml
# config/local.yml
analysis:
  parallel_jobs: 2  # Reduce from 4 to 2
```

**C. Optimize System Resources**
```bash
# Increase memory for Node.js
export NODE_OPTIONS="--max-old-space-size=4096"

# Optimize Go garbage collector
export GOGC=100

# Clear analysis cache
rm -rf /tmp/pr-analyzer-cache/*
```

### 5. Webhook Not Receiving Events

#### Symptom: No analysis triggered on PR creation

**Diagnosis:**
```bash
# Check webhook endpoint
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"test": true}'

# Check GitHub webhook configuration
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/repos/OWNER/REPO/hooks

# Check webhook logs
grep "webhook" /var/log/pr-analyzer/app.log | tail -10
```

**Solutions:**

**A. Webhook Service Down**
```bash
# Check service status
systemctl status pr-webhook

# Start service
sudo systemctl start pr-webhook

# Check logs
journalctl -u pr-webhook -f
```

**B. Firewall Blocking Requests**
```bash
# Check firewall status
sudo ufw status

# Allow webhook port
sudo ufw allow 8080

# For cloud instances, check security groups
```

**C. Incorrect Webhook URL**
```bash
# Update GitHub webhook URL
curl -X PATCH \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"config": {"url": "https://your-domain.com/webhook"}}' \
  https://api.github.com/repos/OWNER/REPO/hooks/HOOK_ID
```

**D. Invalid Webhook Secret**
```bash
# Check webhook secret
echo $WEBHOOK_SECRET

# Update GitHub webhook secret
curl -X PATCH \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"config": {"secret": "'$WEBHOOK_SECRET'"}}' \
  https://api.github.com/repos/OWNER/REPO/hooks/HOOK_ID
```

### 6. Linear Integration Issues

#### Symptom: Issues not created in Linear

**Diagnosis:**
```bash
# Test Linear API connection
curl -H "Authorization: Bearer $LINEAR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ viewer { id name } }"}' \
  https://api.linear.app/graphql

# Check Linear configuration
cat config/local.yml | grep -A 10 "linear:"

# Check Linear logs
grep "linear" /var/log/pr-analyzer/app.log | tail -10
```

**Solutions:**

**A. Invalid API Key**
```bash
# Get new API key from https://linear.app/settings/api
export LINEAR_API_KEY="your-new-api-key"
echo 'export LINEAR_API_KEY="your-new-api-key"' >> ~/.bashrc

# Restart services
sudo systemctl restart pr-webhook analysis-engine
```

**B. Team/Project Not Found**
```bash
# List available teams
curl -H "Authorization: Bearer $LINEAR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ teams { nodes { id name key } } }"}' \
  https://api.linear.app/graphql

# Update configuration with correct team ID
```

**C. Rate Limiting**
```bash
# Check rate limit headers in logs
grep "rate.limit" /var/log/pr-analyzer/app.log

# Reduce request frequency
# config/local.yml
linear:
  rate_limit:
    requests_per_minute: 30  # Reduce from default
```

### 7. Memory and Performance Issues

#### Symptom: High memory usage, slow analysis

**Diagnosis:**
```bash
# Check memory usage
free -h
ps aux --sort=-%mem | head -10

# Check disk I/O
iostat -x 1 5

# Check analysis performance
grep "execution_time" /var/log/pr-analyzer/app.log | tail -10
```

**Solutions:**

**A. Optimize Memory Usage**
```bash
# Increase swap space
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Add to /etc/fstab for persistence
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

**B. Tune Application Settings**
```yaml
# config/local.yml
analysis:
  parallel_jobs: 2  # Reduce parallel processing
  cache:
    max_size: 500   # Reduce cache size

performance:
  memory:
    max_heap_size: 1GB  # Limit heap size
  workers:
    analysis_workers: 2  # Reduce worker count
```

**C. Clean Up Resources**
```bash
# Clear analysis cache
rm -rf /tmp/pr-analyzer-cache/*

# Clean old logs
sudo logrotate -f /etc/logrotate.d/pr-analyzer

# Restart services to free memory
sudo systemctl restart pr-webhook analysis-engine agentapi
```

### 8. SSL/TLS Certificate Issues

#### Symptom: `SSL certificate verify failed`

**Diagnosis:**
```bash
# Check certificate validity
openssl s_client -connect api.anthropic.com:443 -servername api.anthropic.com

# Check system certificates
ls -la /etc/ssl/certs/

# Check curl CA bundle
curl-config --ca
```

**Solutions:**

**A. Update CA Certificates**
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install ca-certificates
sudo update-ca-certificates

# CentOS/RHEL
sudo yum update ca-certificates
```

**B. Configure Custom CA Bundle**
```bash
# Set CA bundle path
export CURL_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt
export SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

# Add to environment
echo 'export CURL_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt' >> ~/.bashrc
```

### 9. Docker Container Issues

#### Symptom: Containers failing to start or crashing

**Diagnosis:**
```bash
# Check container status
docker-compose ps

# Check container logs
docker-compose logs webhook-handler
docker-compose logs analysis-engine
docker-compose logs agentapi

# Check resource usage
docker stats
```

**Solutions:**

**A. Resource Constraints**
```yaml
# docker-compose.yml
services:
  analysis-engine:
    deploy:
      resources:
        limits:
          memory: 4G      # Increase memory limit
          cpus: '2.0'     # Increase CPU limit
```

**B. Volume Mount Issues**
```bash
# Check volume permissions
ls -la ./config ./logs

# Fix permissions
sudo chown -R $USER:$USER ./config ./logs
chmod -R 755 ./config ./logs
```

**C. Network Issues**
```bash
# Recreate network
docker-compose down
docker network prune
docker-compose up -d
```

## 🔍 Advanced Debugging

### Enable Debug Logging

```yaml
# config/local.yml
logging:
  level: debug

debug:
  enabled: true
  pprof_enabled: true
```

### Performance Profiling

```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Database Query Analysis

```sql
-- Enable query logging
ALTER SYSTEM SET log_statement = 'all';
ALTER SYSTEM SET log_min_duration_statement = 1000;  -- Log slow queries
SELECT pg_reload_conf();

-- Check slow queries
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC 
LIMIT 10;
```

### Network Debugging

```bash
# Monitor network traffic
sudo tcpdump -i any port 8080 -A

# Check DNS resolution
nslookup api.anthropic.com
nslookup api.linear.app

# Test connectivity
telnet api.anthropic.com 443
telnet api.linear.app 443
```

## 📊 Monitoring and Alerting

### Set Up Monitoring

```bash
# Install monitoring tools
docker-compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d

# Access dashboards
# Grafana: http://localhost:3000
# Prometheus: http://localhost:9090
# Jaeger: http://localhost:16686
```

### Key Metrics to Monitor

- **Response Time**: API endpoint response times
- **Error Rate**: HTTP 5xx error percentage
- **Memory Usage**: Application memory consumption
- **Database Connections**: Active database connections
- **Queue Length**: Analysis job queue length
- **Agent Sessions**: Active AgentAPI sessions

### Alert Conditions

```yaml
# prometheus/alerts.yml
groups:
  - name: pr-analyzer
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
      
      - alert: HighMemoryUsage
        expr: process_resident_memory_bytes / 1024 / 1024 > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage detected"
```

## 🆘 Getting Help

### Log Collection

Before seeking help, collect relevant logs:

```bash
# Create support bundle
./scripts/collect-logs.sh

# This creates: pr-analyzer-logs-$(date +%Y%m%d-%H%M%S).tar.gz
```

### Support Channels

1. **GitHub Issues**: [Create an issue](https://github.com/Zeeeepa/agentapi/issues)
2. **Discord Community**: [Join our Discord](https://discord.gg/pr-analyzer)
3. **Email Support**: support@pr-analyzer.dev
4. **Documentation**: [docs.pr-analyzer.dev](https://docs.pr-analyzer.dev)

### Information to Include

When reporting issues, include:

- **System Information**: OS, version, architecture
- **Configuration**: Relevant config file sections (redact secrets)
- **Logs**: Error messages and stack traces
- **Steps to Reproduce**: Detailed reproduction steps
- **Expected vs Actual**: What you expected vs what happened

---

**Remember**: Most issues can be resolved by checking logs, verifying configuration, and ensuring all services are running properly. When in doubt, restart services and check the logs for error messages.

