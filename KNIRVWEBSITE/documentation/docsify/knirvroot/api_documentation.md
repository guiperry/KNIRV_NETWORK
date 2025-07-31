

---

**Source**: KNIRVROOT/docs/TODOs/api_documentation.md

# KNIRVCHAIN API Documentation

This document provides comprehensive documentation for the KNIRVCHAIN API endpoints, including WebSocket and REST API interfaces.

## Table of Contents

1. [REST API](#rest-api)
   - [Health Check](#health-check)
   - [Metrics](#metrics)
   - [Alerts](#alerts)
   - [Events](#events)
   - [Windows](#windows)
   - [Analytics](#analytics)
2. [WebSocket API](#websocket-api)
   - [Connection](#connection)
   - [Messages](#messages)
   - [Subscriptions](#subscriptions)
   - [Alerts](#websocket-alerts)

## REST API

The REST API provides access to KNIRVCHAIN data and metrics. All endpoints are prefixed with `/api/v1`.

### Base URL

```
http://localhost:8081/api/v1
```

### Health Check

#### GET /health

Returns the health status of the API and its components.

**Response**:

```json
{
  "status": "ok",
  "timestamp": "2025-01-29T12:34:56Z",
  "service": "rest-api",
  "data_engine": {
    "running": true
  },
  "kafka": {
    "connected": false
  },
  "chromadb": {
    "connected": false
  }
}
```

### Metrics

#### GET /metrics

Returns the current metrics from the data engine.

**Response**:

```json
{
  "timestamp": "2025-01-29T12:34:56Z",
  "events_processed": 1234,
  "events_per_second": 12.34,
  "processing_time_ms": 5,
  "error_count": 0,
  "events_by_type": {
    "block_created": 100,
    "tx_submitted": 500,
    "tx_confirmed": 450,
    "dev_connected": 10,
    "dev_disconnected": 5
  }
}
```

### Alerts

#### GET /alerts

Returns all alerts.

**Query Parameters**:

- `active` (boolean): If true, returns only active (unresolved) alerts.

**Response**:

```json
[
  {
    "id": "error-rate-1234567890",
    "title": "High Error Rate",
    "description": "Error rate exceeds threshold",
    "level": 2,
    "timestamp": "2025-01-29T12:34:56Z",
    "source": "error",
    "data": {
      "error_rate": 15.5,
      "threshold": 10
    },
    "resolved": false
  }
]
```

#### PUT /alerts/{id}

Resolves an alert.

**Path Parameters**:

- `id` (string): The ID of the alert to resolve.

**Response**:

```json
{
  "id": "error-rate-1234567890",
  "resolved": true
}
```

### Events

#### GET /events

Returns recent events.

**Query Parameters**:

- `limit` (integer): Maximum number of events to return (default: 10, max: 100).
- `type` (string): Filter events by type.

**Response**:

```json
[
  {
    "id": "block_created-1234567890",
    "metadata": {
      "type": "block_created",
      "timestamp": "2025-01-29T12:34:56Z",
      "block_number": 12345,
      "hash": "0x1234567890abcdef",
      "tx_count": 10
    },
    "document": "block_created event at 2025-01-29T12:34:56Z",
    "type": "block_created",
    "timestamp": "2025-01-29T12:34:56Z"
  }
]
```

#### GET /events/search

Searches events using semantic search.

**Query Parameters**:

- `q` (string): Search query.
- `limit` (integer): Maximum number of events to return (default: 10, max: 100).

**Response**:

```json
[
  {
    "id": "block_created-1234567890",
    "metadata": {
      "type": "block_created",
      "timestamp": "2025-01-29T12:34:56Z",
      "block_number": 12345,
      "hash": "0x1234567890abcdef",
      "tx_count": 10
    },
    "document": "block_created event at 2025-01-29T12:34:56Z",
    "type": "block_created",
    "timestamp": "2025-01-29T12:34:56Z"
  }
]
```

#### GET /events/types

Returns all event types.

**Response**:

```json
[
  "blockchain",
  "block_created",
  "tx_submitted",
  "tx_confirmed",
  "network",
  "dev_connected",
  "dev_disconnected",
  "user",
  "command_executed",
  "page_view",
  "system",
  "error",
  "warning",
  "info"
]
```

### Windows

#### GET /windows

Returns all time windows.

**Response**:

```json
[
  {
    "start_time": "2025-01-29T12:30:00Z",
    "end_time": "2025-01-29T12:35:00Z",
    "metrics": {
      "event_count": 123,
      "event_rate": 0.41,
      "block_created_count": 5,
      "tx_submitted_count": 50
    },
    "event_counts": {
      "block_created": 5,
      "tx_submitted": 50,
      "tx_confirmed": 45
    }
  }
]
```

#### GET /windows/range

Returns time windows in a specific time range.

**Query Parameters**:

- `start` (string): Start time in RFC3339 format.
- `end` (string): End time in RFC3339 format.

**Response**:

```json
[
  {
    "start_time": "2025-01-29T12:30:00Z",
    "end_time": "2025-01-29T12:35:00Z",
    "metrics": {
      "event_count": 123,
      "event_rate": 0.41,
      "block_created_count": 5,
      "tx_submitted_count": 50
    },
    "event_counts": {
      "block_created": 5,
      "tx_submitted": 50,
      "tx_confirmed": 45
    }
  }
]
```

### Analytics

#### GET /analytics/users

Returns the number of active users in a time range.

**Query Parameters**:

- `start` (string): Start time in RFC3339 format.
- `end` (string): End time in RFC3339 format.

**Response**:

```json
{
  "start": "2025-01-29T12:00:00Z",
  "end": "2025-01-29T13:00:00Z",
  "active_users": 42
}
```

#### GET /analytics/rates

Returns the event rate in a time range.

**Query Parameters**:

- `start` (string): Start time in RFC3339 format.
- `end` (string): End time in RFC3339 format.

**Response**:

```json
{
  "start": "2025-01-29T12:00:00Z",
  "end": "2025-01-29T13:00:00Z",
  "event_rate": 0.75,
  "unit": "events/second"
}
```

## WebSocket API

The WebSocket API provides real-time updates from KNIRVCHAIN.

### Connection

Connect to the WebSocket server at:

```
ws://localhost:8080/ws
```

### Messages

#### Ping/Pong

**Client to Server**:

```json
{
  "type": "ping"
}
```

**Server to Client**:

```json
{
  "type": "pong",
  "time": "2025-01-29T12:34:56Z"
}
```

#### Subscribe

**Client to Server**:

```json
{
  "type": "subscribe",
  "topic": "alerts"
}
```

**Server to Client**:

```json
{
  "type": "subscribed",
  "topic": "alerts",
  "status": "success"
}
```

### Subscriptions

#### Events

Real-time events from the blockchain.

```json
{
  "type": "event",
  "event": {
    "type": "block_created",
    "timestamp": "2025-01-29T12:34:56Z",
    "data": {
      "block_number": 12345,
      "hash": "0x1234567890abcdef",
      "tx_count": 10
    }
  }
}
```

#### Metrics

Real-time metrics updates.

```json
{
  "type": "metrics",
  "metrics": {
    "timestamp": "2025-01-29T12:34:56Z",
    "events_processed": 1234,
    "events_per_second": 12.34,
    "processing_time_ms": 5,
    "error_count": 0,
    "events_by_type": {
      "block_created": 100,
      "tx_submitted": 500,
      "tx_confirmed": 450
    }
  }
}
```

### WebSocket Alerts

#### Alert Notification

```json
{
  "type": "alert",
  "alert": {
    "id": "error-rate-1234567890",
    "title": "High Error Rate",
    "description": "Error rate exceeds threshold",
    "level": 2,
    "timestamp": "2025-01-29T12:34:56Z",
    "source": "error",
    "data": {
      "error_rate": 15.5,
      "threshold": 10
    },
    "resolved": false
  }
}
```

#### Resolve Alert

**Client to Server**:

```json
{
  "type": "resolve_alert",
  "alert_id": "error-rate-1234567890"
}
```

**Server to Client**:

```json
{
  "type": "alert_resolved",
  "alert_id": "error-rate-1234567890",
  "success": true
}
```

## Error Handling

All API endpoints return appropriate HTTP status codes:

- `200 OK`: Request successful
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

Error responses include a JSON body with details:

```json
{
  "error": "Invalid parameter: limit must be a positive integer",
  "status": 400
}
```

## Rate Limiting

The API implements rate limiting to prevent abuse:

- REST API: 100 requests per minute per IP address
- WebSocket API: 10 messages per second per connection

Exceeding these limits will result in a `429 Too Many Requests` response.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
