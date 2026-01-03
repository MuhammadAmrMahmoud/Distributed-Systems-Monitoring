# Distributed Health Monitoring System

A distributed, event-driven system for monitoring external services and broadcasting their health status in real-time using WebSockets.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Features](#features)
- [Technology Stack](#technology-stack)
- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Authentication](#authentication)
- [Installation & Setup](#installation--setup)
- [Running the Application](#running-the-application)
- [API Documentation](#api-documentation)
- [Protocols](#protocols)
- [WebSocket Events](#websocket-events)
- [gRPC Health Check](#grpc-health-check)
- [Database Schema](#database-schema)
- [System Components](#system-components)
- [Workflow](#workflow)
- [Error Handling](#error-handling)
- [Monitoring & Logging](#monitoring--logging)

## Overview

The Distributed Health Monitoring System is a scalable solution designed to:

1. **Monitor External Services**: Periodically check the health of external HTTP endpoints
2. **Queue-based Processing**: Use RabbitMQ for asynchronous job distribution
3. **Real-time Notifications**: Broadcast service state changes via WebSocket connections
4. **Persistent Logging**: Store all health check results and state transitions in PostgreSQL
5. **State Management**: Track service status with configurable failure thresholds

This system decouples monitoring tasks into three independent, scalable components: a **Scheduler** that creates jobs, a **Worker** that executes health checks, and a **WebSocket Hub** that broadcasts state changes to clients.

## Architecture

```
┌─────────────┐
│  Scheduler  │  (Reads services every 5s, creates jobs)
└──────┬──────┘
       │
       ▼
┌────────────────┐
│   RabbitMQ     │  (Job Queue)
└────────┬───────┘
         │
         ▼
    ┌────────┐
    │ Worker │  (Executes health checks, updates state)
    └────┬───┘
         │
         ├─────────────┐
         │             │
         ▼             ▼
    ┌────────┐    ┌─────────┐
    │Database│    │WebSocket│
    │  Logs  │    │   Hub   │
    └────────┘    └────┬────┘
                       │
                       ▼
                  Connected Clients
```

### Key Design Principles

1. **Asynchronous Processing**: Job queue decouples scheduling from execution
2. **Eventual Consistency**: Services are checked independently based on configured intervals
3. **State Transitions**: Broadcasts occur only when service status changes
4. **Failure Threshold**: Services only mark as DOWN after N consecutive failures (configurable)
5. **Persistent Audit Trail**: All checks are logged for historical analysis

## Project Structure

```
.
├── main.go                     # Application entry point
├── config.json                 # Configuration file
├── dockerfile                  # Docker container definition
├── docker-compose.yaml         # Multi-container orchestration
├── go.mod                      # Go module dependencies
│
├── config/
│   ├── config.go              # Configuration loading and database connection
│   └── config.json            # (referenced from root)
│
├── models/
│   └── models.go              # Data models (ExternalService, ServiceCheckLog)
│
├── cache/
│   └── cache.go               # In-memory cache for services
│
├── Repository/
│   └── Repository.go          # Database layer (CRUD operations)
│
└── Service/
    ├── Service.go             # HTTP handlers and WebSocket setup
    ├── broadcast.go           # WebSocket hub and event broadcasting
    ├── scheduler.go           # Job scheduler (creates tasks)
    ├── worker.go              # Job worker (executes health checks)
    └── service.go             # (may contain additional service logic)
```

## Features

✅ **Health Check Monitoring**
- Configurable HTTP methods (GET, POST, PUT, DELETE, PATCH)
- Custom timeouts per service
- Response time tracking

✅ **Intelligent State Management**
- Configurable failure thresholds
- Tracks consecutive failures
- Marks services DOWN only after threshold is exceeded

✅ **Real-time Notifications**
- WebSocket broadcasting for instant client updates
- State change events with timestamps
- Graceful client connection handling

✅ **Distributed Architecture**
- RabbitMQ for queue-based job distribution
- Horizontal scalability (multiple workers possible)
- PostgreSQL for durable data storage

✅ **Comprehensive Logging**
- Append-only check logs with status codes and latency
- State transition tracking
- Error messages and diagnostics

✅ **Docker Support**
- Pre-configured Docker Compose setup
- PostgreSQL container with persistent volumes
- RabbitMQ with management UI

✅ **gRPC Health Checks**
- Monitor gRPC services with latency tracking
- Connection state verification
- Configurable timeouts

✅ **Security**
- Basic authentication on protected endpoints
- Username/password configuration
- Configurable per endpoint

## Technology Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Language | Go | 1.25.1 |
| Web Framework | Gin | v1.11.0 |
| WebSocket | Gorilla WebSocket | v1.5.3 |
| gRPC | Google gRPC | Latest |
| Message Queue | RabbitMQ | 3.13-management |
| Database | PostgreSQL | 16-alpine |
| ORM | GORM | Latest |

## Prerequisites

### Local Development
- Go 1.25.1+
- PostgreSQL 16+
- RabbitMQ 3.13+

### Docker (Recommended)
- Docker
- Docker Compose

## Configuration

The application is configured via `config.json`:

```json
{
  "postgresql": {
    "host": "postgres",              // PostgreSQL host
    "port": 5432,                    // PostgreSQL port
    "user": "health_user",           // Database user
    "password": "health_password",   // Database password
    "database": "health_db",         // Database name
    "sslmode": "disable",            // SSL mode
    "max_open_conns": 25,            // Connection pool size
    "max_idle_conns": 10             // Idle connections
  },
  "rabbitmq": {
    "host": "rabbitmq",              // RabbitMQ host
    "port": 5672,                    // RabbitMQ port
    "username": "guest",             // RabbitMQ username
    "password": "guest",             // RabbitMQ password
    "vhost": "/",                    // Virtual host
    "queue_name": "health_checks",   // Queue name
    "exchange": "",                  // Exchange name (empty = default)
    "routing_key": "health_checks"   // Routing key
  },
  "server": {
    "address": ":8080"               // Server listen address
  }
}
```

## Authentication

The system implements **HTTP Basic Authentication** for protected endpoints.

### Protected Endpoints

The following endpoints require authentication:
- `POST /health-app/externalServices/register` - Register a new service
- `GET /health-app/externalServices/list` - List all services

### Configuration

Basic auth credentials are configured in `config.json` for simplicity:

```json
{
  "auth": {
    "username": "admin",
    "password": "secure_password"
  }
}
```

### Using Basic Auth

**cURL Example:**
```bash
curl -X GET http://localhost:8080/health-app/externalServices/list \
  -u admin:secure_password
```

**JavaScript/Fetch Example:**
```javascript
const credentials = btoa('admin:secure_password');
const response = await fetch('http://localhost:8080/health-app/externalServices/list', {
  headers: {
    'Authorization': 'Basic ' + credentials
  }
});
```

**Python Example:**
```python
import requests
from requests.auth import HTTPBasicAuth

response = requests.get(
    'http://localhost:8080/health-app/externalServices/list',
    auth=HTTPBasicAuth('admin', 'secure_password')
)
```

### Unauthorized Response

If credentials are missing or incorrect:
```json
{
  "error": "unauthorized"
}
```

HTTP Status: `401 Unauthorized`

## Installation & Setup

### Option 1: Docker Compose (Recommended)

```bash
# Clone or navigate to project directory
cd Distributed_Systems_Monitoring

# Start all services
docker-compose up --build

# View logs
docker-compose logs -f app

# Stop services
docker-compose down

# Clean up volumes
docker-compose down -v
```

**Verify Services:**
- API: http://localhost:8080/ping
- RabbitMQ Management: http://localhost:15672 (guest/guest)
- PostgreSQL: localhost:5432

### Option 2: Local Development

```bash
# Install dependencies
go mod download
go mod tidy

# Start PostgreSQL (ensure it's running)
# Start RabbitMQ (ensure it's running)

# Run application
go run main.go
```

## Running the Application

### With Docker Compose
```bash
docker-compose up --build
```

### Locally
```bash
go run main.go
```

**Expected Console Output:**
```
[SCHEDULER] started
[WORKER] check_completed service=Example service status=UP latency_ms=45
[STATE_TRANSITION] service=Example service from=UNKNOWN to=UP at=2025-12-31T10:30:45Z
[WS] State change broadcast: service_id=1
```

## API Documentation

### Health Check
```http
GET /ping
```
**Response:**
```json
{
  "message": "pong"
}
```

### Register Service

```http
POST /health-app/externalServices/register
Content-Type: application/json

{
  "name": "Example API",
  "url": "http://host.docker.internal:9000/health",       <!--if called from local machine use host.docker.internal instead of localhost -->
  "protocol": "HTTP",                                     <!-- HTTP or gRPC are supported -->
  "http_method": "GET",
  "interval": 60,
  "timeout_seconds": 10,
  "failure_threshold": 3
}
```

**Response (201 Created):**
```json
{
  "message": "service registered successfully",
  "service": {
    "id": 1,
    "name": "Example API",
    "url": "http://host.docker.internal:9000/health",
    "http_method": "GET",
    "interval": 60,
    "timeout_seconds": 10,
    "failure_threshold": 3,
    "status": "UP",
    "consecutive_failures": 0,
    "created_at": "2025-12-31T10:00:00Z",
    "updated_at": "2025-12-31T10:00:00Z"
  }
}
```

### List Services

```http
GET /health-app/externalServices/list
```

**Response (200 OK):**
```json
{
  "services": {
    "1": {
      "id": 1,
      "name": "Example API",
      "status": "UP",
      "last_checked_at": "2025-12-31T10:30:45Z",
      ...
    }
  }
}
```

### Get Health Check Logs

```http
GET /health-app/healthLogs/:serviceId?limit=100&offset=0
```

**Parameters:**
- `serviceId` (required): Service ID
- `limit` (optional): Maximum results (default: 100)
- `offset` (optional): Pagination offset (default: 0)

**Response (200 OK):**
```json
{
  "logs": [
    {
      "id": 1,
      "external_service_id": 1,
      "status": "UP",
      "status_code": 200,
      "response_time_ms": 45,
      "error_message": "",
      "checked_at": "2025-12-31T10:30:45Z"
    }
  ]
}
```

## Protocols

The system uses three communication protocols to enable comprehensive health monitoring across different service types:

### 1. HTTP/HTTPS (Health Check Protocol)

**Purpose:** Monitor REST APIs and HTTP-based services

**Characteristics:**
- Executes actual HTTP requests to service endpoints
- Measures response time and status codes
- Supports multiple HTTP methods for flexibility
- Status codes < 400 mark service as UP
- Network-level failure detection

**Supported HTTP Methods:**
- `GET` - Retrieve health status (default)
- `POST` - POST health check requests
- `PUT` - PUT-based health checks
- `DELETE` - DELETE operation checks
- `PATCH` - PATCH operation checks

**Health Check Configuration:**
```json
{
  "name": "REST API Service",
  "url": "https://api.example.com/health",
  "protocol": "HTTP",
  "http_method": "GET",
  "interval": 30,
  "timeout_seconds": 5,
  "failure_threshold": 3
}
```

**Health Check Behavior:**
- Makes real HTTP request to the configured URL
- Response status code < 400 = **UP**
- Response status code ≥ 400 = **DOWN**
- Timeout or connection error = **DOWN**
- Tracks latency in milliseconds
- Logs response status code and error messages

**Example Workflow:**
1. Scheduler creates job: `GET https://api.example.com/health`
2. Worker executes request within 5-second timeout
3. Response: HTTP 200 → Status = UP, Latency = 45ms
4. Service state updated, client notified via WebSocket

### 2. WebSocket (Real-time Events)

**Purpose:** Deliver instant state change notifications to connected monitoring clients

**Characteristics:**
- Persistent TCP connection maintained by server
- Full-duplex, bidirectional communication
- Minimal latency event streaming
- Automatic client-side updates without polling
- Graceful connection handling

**Connection Lifecycle:**
```javascript
// Connect to WebSocket hub
const ws = new WebSocket("ws://localhost:8080/ws");

// Handle connection established
ws.onopen = () => {
  console.log("Connected to health monitoring hub");
};

// Receive state change events
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log(`${message.name} changed from ${message.from} to ${message.to}`);
};

// Handle disconnection
ws.onclose = () => {
  console.log("Disconnected. Reconnecting...");
  // Implement reconnection logic
};
```

**Event Types Broadcast:**

**Service State Change Event** - Sent when service status transitions (UP ↔ DOWN)
```json
{
  "type": "service_state_change",
  "service_id": 1,
  "name": "Example API",
  "from": "UP",
  "to": "DOWN",
  "timestamp": "2025-12-31T10:30:45Z"
}
```

**How State Changes Trigger:**
- Worker marks service DOWN after consecutive failures exceed threshold
- Worker marks service UP after successful check following DOWN status
- Event broadcasts only on actual state transition (not on every check)
- All connected clients receive notification simultaneously

**Broadcasting Mechanism:**
```
Worker Updates State → StateChangeEvent Created → Broadcast via WebSocket Hub → All Clients Notified
```

**Client Use Cases:**
- Real-time monitoring dashboard updates
- Instant alerting systems (Slack, email, SMS)
- Live health metrics display
- Automated incident response triggers

### 3. gRPC (Microservice Health Verification)

**Purpose:** Monitor distributed gRPC microservices efficiently

**Characteristics:**
- HTTP/2 based transport protocol
- Verifies gRPC service connectivity
- Measures connection establishment latency
- Tracks gRPC connection state machine
- Minimal overhead for microservice checks

**Connection States Monitored:**
| State | Description |
|-------|-------------|
| `Ready` | Service is healthy and accepting connections |
| `Connecting` | Attempting to establish connection |
| `Idle` | Connection established but idle |
| `TransientFailure` | Temporary connectivity issue |
| `Shutdown` | Service shutdown or unavailable |

**gRPC Health Check Configuration:**
```json
{
  "name": "gRPC Order Service",
  "url": "grpc.example.com:50051",
  "protocol": "gRPC",
  "timeout_seconds": 5,
  "interval": 30,
  "failure_threshold": 3
}
```

**Health Check Implementation:**
```go
// Worker calls gRPC checker with address and timeout
result := grpc.Check_gRPC("grpc.example.com:50051", 5*time.Second)

// Returns:
// - IsHealthy: true if state == Ready
// - Latency: Connection establishment time
// - StatusCode: gRPC connection state
// - Error: Connection error details (if any)
```

**gRPC Check Behavior:**
- Initiates non-blocking connection to gRPC service
- Connection state must be `Ready` for service to mark as UP
- Any non-Ready state = DOWN
- Tracks connection latency and error details
- No service-level RPC calls required (connection check only)

**Example Workflow:**
1. Scheduler creates job for `grpc.example.com:50051`
2. Worker dials gRPC endpoint with 5-second timeout
3. Connection state = `Ready` → Status = UP, Latency = 12ms
4. Service marked UP, WebSocket broadcasts to clients
5. Next interval: repeat check

### Protocol Comparison

| Feature | HTTP | WebSocket | gRPC |
|---------|------|-----------|------|
| **Direction** | One-way (Client→Server) | Bidirectional | One-way (Client→Server) |
| **Connection Type** | Request-response | Persistent | Connection verification |
| **Latency** | Medium (per request) | Low (instant broadcast) | Very Low (connection check) |
| **Bandwidth** | Medium | Low | Minimal |
| **Target Services** | REST APIs, HTTP endpoints | Connected clients | gRPC microservices |
| **Data Format** | JSON responses | JSON events | gRPC binary protocol |
| **Typical Response Time** | 50-500ms | Instant (<1ms) | 5-50ms |
| **Use Case** | Health check execution | State change notification | Microservice monitoring |
| **Failure Detection** | HTTP status codes | Event delivery | Connection state |

## WebSocket Events

### Connection

```javascript
const ws = new WebSocket("ws://localhost:8080/ws");

ws.onopen = () => {
  console.log("Connected to health monitoring hub");
};
```

### Service State Change Event

**Event Format:**
```json
{
  "type": "service_state_change",
  "service_id": 1,
  "name": "Example API",
  "from": "UP",
  "to": "DOWN",
  "timestamp": "2025-12-31T10:30:45Z"
}
```

**Listener Example:**
```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  if (message.type === "service_state_change") {
    console.log(`${message.name} changed from ${message.from} to ${message.to}`);
    // Update UI, trigger alerts, etc.
  }
};
```

### Disconnection

```javascript
ws.onclose = () => {
  console.log("Disconnected from health monitoring hub");
  // Implement reconnection logic
};

ws.onerror = (error) => {
  console.error("WebSocket error:", error);
};
```

## gRPC Health Check

The system includes a **gRPC health checker** for monitoring gRPC services alongside HTTP services.

### Overview

The gRPC health check module allows you to monitor gRPC service endpoints by attempting to establish connections and verifying their availability.

**Location:** [grpc/grpc.go](grpc/grpc.go)

### Features

- **Connection State Verification**: Checks if gRPC service is in `Ready` state
- **Latency Tracking**: Measures connection establishment time
- **Error Handling**: Captures connection errors with detailed diagnostics
- **Timeout Support**: Configurable timeout per check

### gRPC Health Check Function

```go
func Check_gRPC(address string, timeout time.Duration) models.GRPCHealthResult
```

**Parameters:**
- `address`: gRPC service address (e.g., `localhost:50051`)
- `timeout`: Maximum time to wait for connection (e.g., `5 * time.Second`)

**Returns:**
```go
type GRPCHealthResult struct {
  IsHealthy  bool              // Whether service is reachable
  Latency    time.Duration     // Connection time
  StatusCode connectivity.State // gRPC connection state
  Error      error             // Error if connection failed
}
```

### Connection States

| State | Meaning |
|-------|---------|
| `Ready` | Service is healthy and ready |
| `Connecting` | Attempting to connect |
| `Idle` | Connection idle |
| `TransientFailure` | Temporary connection issue |
| `Shutdown` | Service shutdown |

### Usage Example

**Checking a gRPC Service:**
```go
result := grpc.Check_gRPC("localhost:50051", 5*time.Second)

if result.IsHealthy {
  fmt.Printf("✓ gRPC service UP (latency: %dms)\n", result.Latency.Milliseconds())
} else {
  fmt.Printf("✗ gRPC service DOWN: %v\n", result.Error)
}
```

## Database Schema

### ExternalService Table

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | BIGSERIAL | PRIMARY KEY | Service identifier |
| name | VARCHAR(255) | NOT NULL, UNIQUE | Service name |
| url | VARCHAR(500) | NOT NULL | Health check URL |
| http_method | VARCHAR(10) | NOT NULL, DEFAULT='GET' | HTTP method |
| interval | BIGINT | NOT NULL, DEFAULT=60 | Check interval (seconds) |
| timeout_seconds | BIGINT | NOT NULL, DEFAULT=10 | Request timeout (seconds) |
| failure_threshold | BIGINT | NOT NULL, DEFAULT=3 | Failures before marking DOWN |
| status | VARCHAR(20) | NOT NULL, DEFAULT='UP' | Current status (UP/DOWN) |
| consecutive_failures | BIGINT | NOT NULL, DEFAULT=0 | Failed checks count |
| last_checked_at | TIMESTAMP | Nullable | Last check timestamp |
| created_at | TIMESTAMP | NOT NULL | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL | Update timestamp |

### ServiceCheckLog Table

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | BIGSERIAL | PRIMARY KEY | Log entry identifier |
| external_service_id | BIGINT | NOT NULL, INDEX | Reference to service |
| status | VARCHAR(20) | NOT NULL | Check result (UP/DOWN) |
| status_code | INT | Nullable | HTTP status code |
| response_time_ms | BIGINT | NOT NULL | Response time (milliseconds) |
| error_message | TEXT | Nullable | Error details |
| checked_at | TIMESTAMP | NOT NULL, INDEX | Check timestamp |

**Indexes:**
- `external_services.name` (UNIQUE)
- `external_services.status`
- `service_check_logs.external_service_id`
- `service_check_logs.checked_at`
- Composite: `(external_service_id, checked_at)`

## System Components

### 1. Scheduler

**Location:** [Service/scheduler.go](Service/scheduler.go)

**Responsibility:** Periodically create health check jobs

**Workflow:**
1. Runs every 5 seconds
2. Fetches all registered services from database
3. Determines which services need checking (based on `last_checked_at` + `interval`)
4. Creates `HealthCheckJob` messages
5. Publishes jobs to RabbitMQ queue

**Configuration:**
```go
// Check interval (hardcoded in Service.go)
ticker := time.NewTicker(5 * time.Second)
```

**Error Handling:**
- Logs fetch failures but continues
- Individual job schedule failures don't stop scheduler

### 2. Worker

**Location:** [Service/worker.go](Service/worker.go)

**Responsibility:** Execute health checks and update service state

**Workflow:**
1. Connects to RabbitMQ and consumes jobs
2. For each job:
   - Loads service details from database
   - Performs HTTP request with configured method/timeout
   - Measures response time
   - Saves check result to database
   - Updates service state (success/failure tracking)
   - Broadcasts state change if status changed
   - Acknowledges message

**Key Features:**
- Configurable HTTP timeouts
- Latency tracking
- Error message capture
- Status code tracking
- Failure threshold enforcement

**Error Handling:**
- Invalid jobs are NACKed without requeue
- Missing services are logged and NACKed
- HTTP errors are recorded but don't crash worker
- Messages only ACKed after full processing

### 3. WebSocket Hub & Broadcasting

**Location:** [Service/broadcast.go](Service/broadcast.go)

**Responsibility:** Manage WebSocket connections and broadcast events

**Hub Structure:**
```go
type Hub struct {
  clients    map[*Client]bool    // Connected clients
  broadcast  chan []byte         // Broadcast channel
  register   chan *Client        // Register channel
  unregister chan *Client        // Unregister channel
}
```

**Features:**
- Goroutine-safe client management
- Buffered broadcast channel (256 capacity)
- Non-blocking message delivery
- Automatic dead client cleanup

**Broadcasting:**
- Only on state transitions (UP→DOWN or DOWN→UP)
- Includes service ID, name, old status, new status, timestamp
- JSON formatted

### 4. HTTP Server & API

**Location:** [Service/Service.go](Service/Service.go)

**Responsibility:** HTTP endpoint handling and WebSocket upgrades

**Routes:**
- `GET /ping` - Health check
- `POST /health-app/externalServices/register` - Register service
- `GET /health-app/externalServices/list` - List services
- `GET /health-app/healthLogs/:serviceId` - Get check logs
- `GET /ws` - WebSocket upgrade

**Error Handling:**
- Validates request payloads
- Returns appropriate HTTP status codes
- Comprehensive error messages

### 5. Repository (Database Layer)

**Location:** [Repository/Repository.go](Repository/Repository.go)

**Responsibility:** All database operations

**Operations:**
- `RegisterService()` - Insert new service with validation
- `GetAllServices()` - Fetch all services
- `GetServiceByName()` - Find service by name
- `SaveServiceCheckLog()` - Append check result
- `UpdateServiceState()` - Update status and consecutive failures
- `GetServiceCheckLogs()` - Fetch paginated logs

**State Update Logic:**
- Success: Resets consecutive failures, marks UP
- Failure: Increments consecutive failures, marks DOWN if threshold reached
- Returns `StateChange` struct if status changed

## Workflow

### Health Check Lifecycle

```
1. SCHEDULER PHASE (every 5 seconds)
   ├─ Load all services
   ├─ Check if service.last_checked_at + interval < now
   └─ Create HealthCheckJob and publish to RabbitMQ
   
2. QUEUE PHASE
   └─ Job sits in RabbitMQ queue
   
3. WORKER PHASE
   ├─ Consume job from queue
   ├─ Load service from database
   ├─ Perform HTTP request (with timeout)
   │  ├─ On success (HTTP < 400): status = UP
   │  └─ On error: status = DOWN, capture error message
   ├─ Save ServiceCheckLog entry (append-only)
   ├─ Update service state:
   │  ├─ Calculate new consecutive_failures
   │  ├─ Determine new status (up/down)
   │  └─ Return StateChange if status changed
   ├─ Broadcast if state changed:
   │  └─ Send WebSocket event to all connected clients
   └─ ACK message to RabbitMQ
   
4. CLIENT PHASE (WebSocket)
   └─ Receive state change event and update UI
```

### Service Status Transitions

```
Initial State: UP (assumed)

Failure Sequence:
  UP
  ├─ Check 1 fails → consecutive_failures = 1 → status = UP
  ├─ Check 2 fails → consecutive_failures = 2 → status = UP
  ├─ Check 3 fails → consecutive_failures = 3 → status = DOWN ✓ BROADCAST
  └─ Check 4 fails → consecutive_failures = 4 → status = DOWN (no broadcast)

Recovery Sequence:
  DOWN
  ├─ Check succeeds → consecutive_failures = 0 → status = UP ✓ BROADCAST
  └─ Check succeeds → consecutive_failures = 0 → status = UP (no broadcast)
```

## Error Handling

### Scheduler Errors
| Error | Handling | Impact |
|-------|----------|--------|
| Database unavailable | Logged, scheduler continues | Checks paused temporarily |
| RabbitMQ unavailable | Logged, scheduler continues | Jobs not queued |
| Job marshal error | Logged, job skipped | That check skipped |

### Worker Errors
| Error | Handling | Impact |
|-------|----------|--------|
| Invalid job format | NACKed (no requeue) | Message discarded |
| Service not found | Logged, NACKed | Uses last known service state |
| HTTP request timeout | Recorded in log | Service marked DOWN |
| HTTP request error | Recorded in log | Service marked DOWN |
| Database write failed | Logged | State change not persisted (eventual consistency) |

### WebSocket Errors
| Error | Handling |
|-------|----------|
| Upgrade failure | HTTP 400 error returned |
| Client read error | Client unregistered |
| Client write error | Client deleted from hub |
| Connection closed | Auto-cleanup via defer |

## Monitoring & Logging

### Log Format

All logs use the format: `[COMPONENT] message key=value key=value`

**Examples:**
```
[SCHEDULER] started
[SCHEDULER] schedule_failed service=API_1 err=connection timeout

[WORKER] check_completed service=API_1 status=UP latency_ms=45 error=
[WORKER] invalid_job err=json unmarshal failed
[WORKER] state_update_failed service=API_1 err=database error

[STATE_TRANSITION] service=API_1 from=UP to=DOWN at=2025-12-31T10:30:45Z

[WS] upgrade_failed err=bad upgrade request
[WS] read_error err=connection closed
[WS] write_error err=broken pipe
[WS] marshal_failed service=API_1 err=json encoding error
```

### Key Metrics to Monitor

1. **Check Latency**: `response_time_ms` in ServiceCheckLog
2. **Failure Rate**: Count of checks with status=DOWN
3. **State Changes**: Monitor WebSocket broadcast frequency
4. **Queue Depth**: RabbitMQ management UI shows pending jobs
5. **Database Connections**: Monitor connection pool usage
6. **Worker Health**: Frequency of worker logs

### Debugging

**Check Service Health:**
```bash
curl http://localhost:8080/health-app/externalServices/list
```

**View Recent Logs:**
```bash
curl http://localhost:8080/health-app/healthLogs/1?limit=50
```

**Monitor RabbitMQ:**
- Visit http://localhost:15672
- Check queue depth under "health_checks" queue

**View Application Logs:**
```bash
docker-compose logs -f app
```

---

## Quick Start Example

### 1. Start Services
```bash
docker-compose up -d
```

### 2. Register a Service
```bash
curl -X POST http://localhost:8080/health-app/externalServices/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Google API",
    "url": "https://www.google.com",
    "http_method": "GET",
    "interval": 30,
    "timeout_seconds": 5,
    "failure_threshold": 2
  }'
```

### 3. Connect to WebSocket
```javascript
const ws = new WebSocket("ws://localhost:8080/ws");
ws.onmessage = (e) => console.log(JSON.parse(e.data));
```

### 4. View Health Logs
```bash
curl http://localhost:8080/health-app/healthLogs/1?limit=10
```

---

## Production Considerations

### Scalability
- **Horizontal Scaling**: Run multiple worker instances consuming from same queue
- **Load Balancing**: Use reverse proxy (Nginx) for API
- **Database**: Consider read replicas for logs queries

### High Availability
- Run scheduler and workers in separate containers/pods
- Use persistent volumes for PostgreSQL
- Implement connection pooling and retry logic

### Security
- Use strong RabbitMQ credentials (change `guest/guest`)
- Enable PostgreSQL authentication
- Add HTTPS/TLS for API and WebSocket
- Implement rate limiting on endpoints
- Validate service URLs to prevent SSRF

### Monitoring & Alerting
- Integrate with Prometheus for metrics
- Set up alerts for service DOWN status
- Monitor queue depth for bottlenecks
- Track worker error rates

### Performance Tuning
- Adjust connection pool sizes in config.json
- Tune check intervals based on service criticality
- Use appropriate failure thresholds
- Monitor and optimize database indexes

---

**Last Updated:** December 31, 2025

