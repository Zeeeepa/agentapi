# 📚 API Reference

Complete API documentation for the PR Analysis & CI/CD Automation System.

## 🌐 Base URLs

- **Production**: `https://api.pr-analyzer.dev`
- **Staging**: `https://staging-api.pr-analyzer.dev`
- **Local Development**: `http://localhost:8080`

## 🔐 Authentication

All API requests require authentication using Bearer tokens.

```bash
# Include in request headers
Authorization: Bearer YOUR_API_TOKEN
```

### Getting an API Token

```bash
# Login to get a token
curl -X POST https://api.pr-analyzer.dev/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your-username",
    "password": "your-password"
  }'
```

**Response**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-12-31T23:59:59Z",
  "user": {
    "id": "user-123",
    "username": "your-username",
    "role": "developer"
  }
}
```

## 📋 Webhook API

### POST /webhook

Receives GitHub webhook events for PR analysis.

**Headers**:
- `Content-Type: application/json`
- `X-Hub-Signature-256: sha256=<signature>`
- `X-GitHub-Event: pull_request`

**Request Body**:
```json
{
  "action": "opened",
  "number": 123,
  "pull_request": {
    "id": 123456789,
    "number": 123,
    "title": "Add new feature",
    "body": "This PR adds a new feature...",
    "state": "open",
    "head": {
      "sha": "abc123def456",
      "ref": "feature/new-feature",
      "repo": {
        "full_name": "owner/repo",
        "clone_url": "https://github.com/owner/repo.git"
      }
    },
    "base": {
      "sha": "def456abc123",
      "ref": "main"
    },
    "user": {
      "login": "developer",
      "id": 12345
    },
    "html_url": "https://github.com/owner/repo/pull/123"
  },
  "repository": {
    "id": 987654321,
    "name": "repo",
    "full_name": "owner/repo",
    "private": false,
    "clone_url": "https://github.com/owner/repo.git"
  }
}
```

**Response**:
```json
{
  "status": "accepted",
  "analysis_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "PR analysis queued successfully",
  "estimated_completion": "2024-01-01T12:05:00Z",
  "webhook_id": "wh_123456789"
}
```

**Status Codes**:
- `200 OK`: Webhook processed successfully
- `400 Bad Request`: Invalid webhook payload
- `401 Unauthorized`: Invalid signature
- `422 Unprocessable Entity`: Unsupported event type
- `500 Internal Server Error`: Server error

### GET /webhook/health

Health check endpoint for webhook service.

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "version": "1.0.0",
  "uptime": "72h30m15s"
}
```

### GET /webhook/metrics

Prometheus metrics endpoint.

**Response**: Prometheus format metrics

```
# HELP webhook_requests_total Total number of webhook requests
# TYPE webhook_requests_total counter
webhook_requests_total{status="success"} 1234
webhook_requests_total{status="error"} 56

# HELP webhook_request_duration_seconds Webhook request duration
# TYPE webhook_request_duration_seconds histogram
webhook_request_duration_seconds_bucket{le="0.1"} 100
webhook_request_duration_seconds_bucket{le="0.5"} 200
```

## 🔍 Analysis API

### POST /analysis

Create a new analysis job.

**Request Body**:
```json
{
  "repository": "owner/repo",
  "pr_number": 123,
  "head_sha": "abc123def456",
  "base_sha": "def456abc123",
  "modules": ["static", "dynamic", "security", "performance"],
  "priority": "normal",
  "options": {
    "include_tests": true,
    "max_execution_time": 600,
    "notification_webhook": "https://your-app.com/webhook"
  }
}
```

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "repository": "owner/repo",
  "pr_number": 123,
  "modules": ["static", "dynamic", "security", "performance"],
  "created_at": "2024-01-01T12:00:00Z",
  "estimated_completion": "2024-01-01T12:05:00Z"
}
```

### GET /analysis/{id}

Get analysis job details and status.

**Path Parameters**:
- `id` (string): Analysis job ID

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "repository": "owner/repo",
  "pr_number": 123,
  "head_sha": "abc123def456",
  "base_sha": "def456abc123",
  "modules": ["static", "dynamic", "security", "performance"],
  "progress": {
    "total_modules": 4,
    "completed_modules": 2,
    "current_module": "security",
    "percentage": 50
  },
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:02:30Z",
  "estimated_completion": "2024-01-01T12:05:00Z"
}
```

**Status Values**:
- `queued`: Job is queued for processing
- `running`: Analysis is in progress
- `completed`: Analysis finished successfully
- `failed`: Analysis failed with errors
- `cancelled`: Analysis was cancelled

### GET /analysis/{id}/results

Get complete analysis results.

**Query Parameters**:
- `format` (string, optional): Response format (`json`, `markdown`, `html`)
- `include_raw` (boolean, optional): Include raw analysis data

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "repository": "owner/repo",
  "pr_number": 123,
  "summary": {
    "total_issues": 15,
    "critical_issues": 2,
    "high_issues": 5,
    "medium_issues": 6,
    "low_issues": 2,
    "suggestions": 8,
    "auto_fixable": 12
  },
  "modules": {
    "static_analysis": {
      "status": "completed",
      "execution_time_ms": 15000,
      "issues": [
        {
          "id": "issue-001",
          "type": "unused_function",
          "severity": "medium",
          "file": "src/utils/helpers.ts",
          "line": 45,
          "column": 10,
          "message": "Function 'calculateTotal' is defined but never used",
          "suggestion": "Remove unused function or export it if needed",
          "confidence": 0.95,
          "auto_fixable": true
        }
      ]
    },
    "dynamic_analysis": {
      "status": "completed",
      "execution_time_ms": 45000,
      "performance_metrics": {
        "execution_time": "2.3s",
        "memory_usage": "45MB",
        "cpu_usage": "12%"
      },
      "hotspots": [
        {
          "function": "processLargeDataset",
          "file": "src/data/processor.ts",
          "line": 123,
          "execution_time": "1.8s",
          "calls": 1,
          "suggestion": "Consider implementing pagination or streaming"
        }
      ]
    },
    "security_analysis": {
      "status": "completed",
      "execution_time_ms": 30000,
      "vulnerabilities": [
        {
          "id": "vuln-001",
          "type": "sql_injection",
          "severity": "critical",
          "file": "src/database/queries.ts",
          "line": 67,
          "cwe": "CWE-89",
          "description": "Potential SQL injection vulnerability",
          "recommendation": "Use parameterized queries",
          "references": [
            "https://owasp.org/www-community/attacks/SQL_Injection"
          ]
        }
      ]
    },
    "performance_analysis": {
      "status": "completed",
      "execution_time_ms": 60000,
      "metrics": {
        "bundle_size": "2.3MB",
        "load_time": "1.2s",
        "first_contentful_paint": "0.8s",
        "largest_contentful_paint": "1.1s"
      },
      "optimizations": [
        {
          "type": "code_splitting",
          "potential_savings": "40%",
          "description": "Implement dynamic imports for route-based code splitting"
        }
      ]
    }
  },
  "linear_issues": [
    {
      "id": "linear-issue-123",
      "title": "Fix SQL injection vulnerability in queries.ts",
      "url": "https://linear.app/team/issue/ABC-123",
      "status": "open",
      "priority": "urgent"
    }
  ],
  "completed_at": "2024-01-01T12:05:00Z"
}
```

### GET /analysis/{id}/events

Server-Sent Events stream for real-time analysis updates.

**Response**: SSE stream

```
event: analysis_started
data: {"id": "550e8400-e29b-41d4-a716-446655440000", "timestamp": "2024-01-01T12:00:00Z"}

event: module_started
data: {"module": "static_analysis", "timestamp": "2024-01-01T12:00:05Z"}

event: module_completed
data: {"module": "static_analysis", "issues_found": 8, "timestamp": "2024-01-01T12:01:20Z"}

event: analysis_completed
data: {"id": "550e8400-e29b-41d4-a716-446655440000", "total_issues": 15, "timestamp": "2024-01-01T12:05:00Z"}
```

### DELETE /analysis/{id}

Cancel a running analysis job.

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "cancelled",
  "message": "Analysis job cancelled successfully",
  "cancelled_at": "2024-01-01T12:03:00Z"
}
```

### GET /analysis

List analysis jobs with filtering and pagination.

**Query Parameters**:
- `repository` (string, optional): Filter by repository
- `status` (string, optional): Filter by status
- `limit` (integer, optional): Number of results per page (default: 20, max: 100)
- `offset` (integer, optional): Pagination offset (default: 0)
- `sort` (string, optional): Sort field (`created_at`, `updated_at`, `status`)
- `order` (string, optional): Sort order (`asc`, `desc`)

**Response**:
```json
{
  "jobs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "repository": "owner/repo",
      "pr_number": 123,
      "status": "completed",
      "created_at": "2024-01-01T12:00:00Z",
      "completed_at": "2024-01-01T12:05:00Z"
    }
  ],
  "pagination": {
    "total": 150,
    "limit": 20,
    "offset": 0,
    "has_next": true,
    "has_prev": false
  }
}
```

## 🤖 AgentAPI Integration

### GET /agentapi/status

Check AgentAPI connection and agent status.

**Response**:
```json
{
  "status": "connected",
  "agent": {
    "type": "claude",
    "version": "3.0.0",
    "status": "idle",
    "capabilities": ["edit", "bash", "replace", "create"],
    "session_id": "session-123"
  },
  "server": {
    "version": "1.0.0",
    "uptime": "24h15m30s",
    "active_sessions": 3
  },
  "last_activity": "2024-01-01T12:00:00Z"
}
```

### POST /agentapi/fix

Request automated fix for specific issues.

**Request Body**:
```json
{
  "analysis_id": "550e8400-e29b-41d4-a716-446655440000",
  "issues": [
    {
      "id": "issue-001",
      "priority": "high",
      "auto_fix": true
    },
    {
      "id": "issue-002",
      "priority": "medium",
      "auto_fix": true
    }
  ],
  "options": {
    "create_pr": true,
    "pr_title": "Automated fixes for analysis issues",
    "pr_description": "This PR contains automated fixes generated by the analysis system.",
    "target_branch": "main"
  }
}
```

**Response**:
```json
{
  "fix_session_id": "fix-session-456",
  "status": "started",
  "issues_to_fix": 2,
  "estimated_completion": "2024-01-01T12:10:00Z",
  "agent_session_url": "http://localhost:3284/chat"
}
```

### GET /agentapi/fix/{session_id}

Get status of an automated fix session.

**Response**:
```json
{
  "session_id": "fix-session-456",
  "status": "running",
  "progress": {
    "total_issues": 2,
    "completed_issues": 1,
    "current_issue": "issue-002",
    "percentage": 50
  },
  "fixes_applied": [
    {
      "issue_id": "issue-001",
      "status": "completed",
      "files_modified": ["src/utils/helpers.ts"],
      "description": "Removed unused function 'calculateTotal'"
    }
  ],
  "created_at": "2024-01-01T12:05:00Z",
  "updated_at": "2024-01-01T12:07:30Z"
}
```

### GET /agentapi/messages/{session_id}

Get conversation history for an agent session.

**Response**:
```json
{
  "session_id": "fix-session-456",
  "messages": [
    {
      "id": "msg-001",
      "type": "user",
      "content": "Please fix the unused function in src/utils/helpers.ts line 45",
      "timestamp": "2024-01-01T12:05:00Z"
    },
    {
      "id": "msg-002",
      "type": "agent",
      "content": "I'll help you fix the unused function. Let me examine the file first.",
      "timestamp": "2024-01-01T12:05:02Z"
    },
    {
      "id": "msg-003",
      "type": "agent",
      "content": "I've removed the unused function 'calculateTotal' from line 45. The function was not referenced anywhere in the codebase.",
      "timestamp": "2024-01-01T12:05:15Z"
    }
  ],
  "total_messages": 3
}
```

## 🔗 Linear Integration API

### GET /linear/teams

Get available Linear teams.

**Response**:
```json
{
  "teams": [
    {
      "id": "team-123",
      "name": "Engineering",
      "key": "ENG",
      "description": "Engineering team"
    },
    {
      "id": "team-456",
      "name": "Security",
      "key": "SEC",
      "description": "Security team"
    }
  ]
}
```

### POST /linear/issues

Create a Linear issue from analysis results.

**Request Body**:
```json
{
  "analysis_id": "550e8400-e29b-41d4-a716-446655440000",
  "issue_ids": ["issue-001", "issue-002"],
  "team_id": "team-123",
  "priority": "high",
  "template": "security_issue",
  "assignee_id": "user-789"
}
```

**Response**:
```json
{
  "linear_issue": {
    "id": "linear-issue-123",
    "identifier": "ENG-456",
    "title": "Fix security vulnerabilities in authentication module",
    "url": "https://linear.app/team/issue/ENG-456",
    "status": "backlog",
    "priority": "high",
    "assignee": {
      "id": "user-789",
      "name": "John Doe",
      "email": "john@company.com"
    }
  },
  "created_at": "2024-01-01T12:00:00Z"
}
```

### GET /linear/issues/{analysis_id}

Get Linear issues created for an analysis.

**Response**:
```json
{
  "analysis_id": "550e8400-e29b-41d4-a716-446655440000",
  "issues": [
    {
      "id": "linear-issue-123",
      "identifier": "ENG-456",
      "title": "Fix security vulnerabilities",
      "url": "https://linear.app/team/issue/ENG-456",
      "status": "in_progress",
      "priority": "high",
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:30:00Z"
    }
  ]
}
```

## 📊 Metrics and Reporting API

### GET /metrics/summary

Get system-wide metrics summary.

**Query Parameters**:
- `period` (string): Time period (`1h`, `24h`, `7d`, `30d`)
- `repository` (string, optional): Filter by repository

**Response**:
```json
{
  "period": "24h",
  "summary": {
    "total_analyses": 45,
    "completed_analyses": 42,
    "failed_analyses": 3,
    "average_duration": "4m 32s",
    "total_issues_found": 234,
    "issues_auto_fixed": 187,
    "auto_fix_success_rate": 0.799
  },
  "by_severity": {
    "critical": 12,
    "high": 34,
    "medium": 89,
    "low": 99
  },
  "by_module": {
    "static_analysis": {
      "executions": 42,
      "average_duration": "1m 15s",
      "issues_found": 156
    },
    "security_analysis": {
      "executions": 42,
      "average_duration": "2m 30s",
      "issues_found": 45
    },
    "performance_analysis": {
      "executions": 40,
      "average_duration": "3m 45s",
      "issues_found": 33
    }
  }
}
```

### GET /metrics/repositories

Get metrics by repository.

**Response**:
```json
{
  "repositories": [
    {
      "name": "owner/repo1",
      "analyses": 25,
      "issues_found": 123,
      "auto_fix_rate": 0.85,
      "average_duration": "3m 45s",
      "last_analysis": "2024-01-01T11:30:00Z"
    },
    {
      "name": "owner/repo2",
      "analyses": 20,
      "issues_found": 89,
      "auto_fix_rate": 0.72,
      "average_duration": "5m 12s",
      "last_analysis": "2024-01-01T10:15:00Z"
    }
  ]
}
```

### GET /reports/analysis/{id}

Generate detailed analysis report.

**Query Parameters**:
- `format` (string): Report format (`pdf`, `html`, `markdown`)
- `include_raw_data` (boolean): Include raw analysis data

**Response**: Binary data (PDF) or HTML/Markdown content

## 🔧 Configuration API

### GET /config

Get current system configuration.

**Response**:
```json
{
  "analysis": {
    "max_parallel_jobs": 5,
    "default_timeout": 600,
    "enabled_modules": ["static", "dynamic", "security", "performance"]
  },
  "integrations": {
    "linear": {
      "enabled": true,
      "auto_create_issues": true,
      "default_team": "team-123"
    },
    "agentapi": {
      "enabled": true,
      "auto_fix_enabled": true,
      "agent_type": "claude"
    }
  },
  "notifications": {
    "webhook_enabled": true,
    "email_enabled": false,
    "slack_enabled": true
  }
}
```

### PUT /config

Update system configuration.

**Request Body**:
```json
{
  "analysis": {
    "max_parallel_jobs": 8,
    "default_timeout": 900
  },
  "integrations": {
    "linear": {
      "auto_create_issues": false
    }
  }
}
```

**Response**:
```json
{
  "message": "Configuration updated successfully",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

## 🚨 Error Handling

### Error Response Format

All API errors follow a consistent format:

```json
{
  "error": {
    "code": "ANALYSIS_TIMEOUT",
    "message": "Analysis job timed out after 600 seconds",
    "details": {
      "analysis_id": "550e8400-e29b-41d4-a716-446655440000",
      "timeout_duration": 600,
      "modules_completed": ["static", "security"],
      "modules_failed": ["dynamic", "performance"]
    },
    "timestamp": "2024-01-01T12:10:00Z",
    "request_id": "req-123456789"
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Invalid request format or parameters |
| `UNAUTHORIZED` | 401 | Invalid or missing authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `ANALYSIS_TIMEOUT` | 408 | Analysis job timed out |
| `ANALYSIS_FAILED` | 422 | Analysis job failed |
| `AGENT_UNAVAILABLE` | 503 | AgentAPI service unavailable |
| `INTERNAL_ERROR` | 500 | Internal server error |

## 📝 Rate Limiting

API requests are rate limited per user/API key:

- **Default**: 100 requests per minute
- **Burst**: 20 requests per second
- **Analysis creation**: 10 requests per minute

Rate limit headers are included in responses:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```

## 🔄 Webhooks

### Outgoing Webhooks

The system can send webhooks for various events:

#### Analysis Completed

```json
{
  "event": "analysis.completed",
  "timestamp": "2024-01-01T12:05:00Z",
  "data": {
    "analysis_id": "550e8400-e29b-41d4-a716-446655440000",
    "repository": "owner/repo",
    "pr_number": 123,
    "status": "completed",
    "summary": {
      "total_issues": 15,
      "critical_issues": 2,
      "auto_fixable": 12
    },
    "linear_issues_created": 3,
    "fixes_applied": 8
  }
}
```

#### Analysis Failed

```json
{
  "event": "analysis.failed",
  "timestamp": "2024-01-01T12:05:00Z",
  "data": {
    "analysis_id": "550e8400-e29b-41d4-a716-446655440000",
    "repository": "owner/repo",
    "pr_number": 123,
    "status": "failed",
    "error": {
      "code": "MODULE_TIMEOUT",
      "message": "Security analysis module timed out",
      "module": "security_analysis"
    }
  }
}
```

### Webhook Configuration

Configure outgoing webhooks via the API:

```bash
curl -X POST https://api.pr-analyzer.dev/webhooks \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-app.com/webhook",
    "events": ["analysis.completed", "analysis.failed"],
    "secret": "your-webhook-secret"
  }'
```

## 📖 SDK Examples

### JavaScript/TypeScript

```typescript
import { PRAnalyzerClient } from '@pr-analyzer/sdk';

const client = new PRAnalyzerClient({
  apiKey: 'your-api-key',
  baseUrl: 'https://api.pr-analyzer.dev'
});

// Create analysis
const analysis = await client.analysis.create({
  repository: 'owner/repo',
  prNumber: 123,
  modules: ['static', 'security']
});

// Get results
const results = await client.analysis.getResults(analysis.id);

// Listen for real-time updates
client.analysis.onUpdate(analysis.id, (update) => {
  console.log('Analysis update:', update);
});
```

### Python

```python
from pr_analyzer import PRAnalyzerClient

client = PRAnalyzerClient(
    api_key='your-api-key',
    base_url='https://api.pr-analyzer.dev'
)

# Create analysis
analysis = client.analysis.create(
    repository='owner/repo',
    pr_number=123,
    modules=['static', 'security']
)

# Get results
results = client.analysis.get_results(analysis.id)

# Request automated fixes
fix_session = client.agentapi.fix(
    analysis_id=analysis.id,
    issues=[issue.id for issue in results.critical_issues]
)
```

### Go

```go
package main

import (
    "context"
    "github.com/pr-analyzer/go-sdk"
)

func main() {
    client := pranalyzer.NewClient("your-api-key")
    
    // Create analysis
    analysis, err := client.Analysis.Create(context.Background(), &pranalyzer.CreateAnalysisRequest{
        Repository: "owner/repo",
        PRNumber:   123,
        Modules:    []string{"static", "security"},
    })
    if err != nil {
        panic(err)
    }
    
    // Get results
    results, err := client.Analysis.GetResults(context.Background(), analysis.ID)
    if err != nil {
        panic(err)
    }
}
```

---

For more examples and detailed SDK documentation, visit our [Developer Portal](https://developers.pr-analyzer.dev).

