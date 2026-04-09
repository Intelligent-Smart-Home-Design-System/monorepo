# WebSocket Protocol (Simulation UI)

This document defines the bidirectional WebSocket protocol between the UI and the backend
for running and visualizing apartment simulations.

## Connection

- Endpoint: `/ws/simulation`
- Transport: WebSocket (JSON messages)
- Encoding: UTF-8 JSON

## Message Envelope

Every message uses this common envelope:

```json
{
  "type": "string",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "optional-client-generated-id",
  "payload": {}
}
```

Fields:
- `type`: message name (see below)
- `ts`: ISO timestamp (client or server time)
- `reqId`: optional request id to match responses
- `payload`: message-specific data

## Client -> Server

### `hello`
Initial handshake. Client capabilities.

```json
{
  "type": "hello",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "client": "sim-ui",
    "version": "1.0.0",
    "features": ["multiscenario", "floor-v1"]
  }
}
```

### `floor:get`
Request current floor plan.

```json
{
  "type": "floor:get",
  "ts": "2026-02-18T12:00:00.000Z"
}
```

### `scenario:list`
Request available scenarios.

```json
{
  "type": "scenario:list",
  "ts": "2026-02-18T12:00:00.000Z"
}
```

### `simulation:start`
Start simulation.

```json
{
  "type": "simulation:start",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "mode": "parallel",
    "scenarioIds": ["scn_1", "scn_2"],
    "speed": 1.0
  }
}
```

### `simulation:pause`

```json
{
  "type": "simulation:pause",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

### `simulation:resume`

```json
{
  "type": "simulation:resume",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

### `simulation:stop`

```json
{
  "type": "simulation:stop",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001"
}
```

### `devices:update`
Client-side override for device positions (optional future feature).

```json
{
  "type": "devices:update",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "devices": [
      { "id": "motion_sensor_hall", "x": 0.52, "y": 0.78, "roomId": "hall" }
    ]
  }
}
```

## Server -> Client

### `hello:ack`

```json
{
  "type": "hello:ack",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "server": "sim-backend",
    "version": "1.0.0"
  }
}
```

### `floor`
Returns current floor plan (see `floor.json` format).

```json
{
  "type": "floor",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": { "...": "floor.json content" }
}
```

### `scenario:list`

```json
{
  "type": "scenario:list",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "scenarios": [
      { "id": "scn_1", "title": "Движение → включить свет", "chain": ["motion_sensor_hall", "hub", "lamp_hall"] }
    ]
  }
}
```

### `simulation:status`
Current state of the simulation.

```json
{
  "type": "simulation:status",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "state": "running",
    "mode": "parallel",
    "speed": 1.0,
    "scenarioIds": ["scn_1", "scn_2"]
  }
}
```

### `simulation:step`
One step update for visualization.

```json
{
  "type": "simulation:step",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "scenarioId": "scn_1",
    "stepIndex": 2,
    "activeDevice": "lamp_hall",
    "activeEdge": ["hub", "lamp_hall"]
  }
}
```

### `device:state`
Device state update.

```json
{
  "type": "device:state",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "id": "lamp_hall",
    "state": "active"
  }
}
```

### `log:event`
Console/event feed.

```json
{
  "type": "log:event",
  "ts": "2026-02-18T12:00:00.000Z",
  "payload": {
    "level": "INFO",
    "device": "hub",
    "message": "Rule matched"
  }
}
```

### `error`

```json
{
  "type": "error",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "code": "INVALID_REQUEST",
    "message": "Scenario not found"
  }
}
```

