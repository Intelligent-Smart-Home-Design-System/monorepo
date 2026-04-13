## WebSocket Protocol (Simulation UI)

This document defines the bidirectional WebSocket protocol between the UI and the backend
for running and visualizing apartment simulations.

## Connection

- Endpoint: `/ws/simulation`
- Transport: WebSocket (JSON messages)
- Encoding: UTF-8 JSON

## Design notes (important)

- The backend does **not** fetch the apartment plan from other modules for simulation. The UI already has the apartment/floor/devices produced by previous pipeline steps and provides everything required in `simulation:start`.
- UI drives the simulation in **lockstep**: every UI “tick” is a `simulation:tick` message. This is the simplest way to keep the simulation deterministic.
- “Pause / resume” is a UI concern. If the UI stops sending ticks, the backend naturally stops advancing the simulation.

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

### `simulation:start`
Start a simulation session. The UI provides all required data: apartment plan, devices and their IDs/types/positions, and scenario graph (device adjacency).

Notes:
- `dtSim` is the amount of **simulation time** advanced by every `simulation:tick`. The UI controls real-time speed by how often it sends ticks.
- `devices[].id` must be stable and unique within the session. Scenarios reference devices by these IDs.

```json
{
  "type": "simulation:start",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "dtSim": 1.0,
    "apartment": {
      "id": "apt_1",
      "floor": { "...": "ui floor plan payload (existing UI format)" }
    },
    "devices": [
      {
        "id": "lamp_hall",
        "type": "lamp",
        "roomId": "hall",
        "x": 0.52,
        "y": 0.78,
        "state": { "turned_on": false }
      },
      {
        "id": "motion_sensor_hall",
        "type": "motion_sensor",
        "roomId": "hall",
        "x": 0.40,
        "y": 0.70,
        "state": {}
      }
    ],
    "scenarios": [
      {
        "id": "motion_light_hall",
        "edges": [
          {
            "to": "lamp_hall",
            "action": "turn_on"
          }
        ]
      }
    ]
  }
}
```

### `simulation:tick`
Advance the simulation by one lockstep tick.

The UI SHOULD wait for the corresponding `simulation:step` response before sending the next tick (determinism + backpressure).

To avoid losing user inputs for a given tick (e.g. human movement), the UI SHOULD include all inputs collected since the previous tick inside this message. The backend applies these inputs and then advances the simulation by one tick atomically.

```json
{
  "type": "simulation:tick",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "tick": 1,
    "inputs": [
      {
        "kind": "human:move",
        "humanId": "player_1",
        "to": { "x": 0.60, "y": 0.78 }
      }
    ]
  }
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

### `simulation:started`
Acknowledges a successful start and echoes effective parameters.

```json
{
  "type": "simulation:started",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "dtSim": 1.0,
    "state": "running"
  }
}
```

### `simulation:status`
Current state of the simulation.

```json
{
  "type": "simulation:status",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "state": "running",
    "dtSim": 1.0,
    "tick": 1
  }
}
```

### `simulation:step`
One lockstep update for visualization. Prefer to send all changes for the tick in a single message.

```json
{
  "type": "simulation:step",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
  "payload": {
    "tick": 1,
    "simTime": 1.0,
    "stateChanges": [
      {
        "id": "lamp_hall",
        "patch": { "turned_on": true }
      }
    ],
    "triggeredEdges": [
      { "from": "motion_sensor_hall", "to": "lamp_hall", "action": "turn_on" }
    ],
    "humans": [
      { "id": "player_1", "x": 0.60, "y": 0.78 }
    ]
  }
}
```

### `log:event`
Console/event feed.

```json
{
  "type": "log:event",
  "ts": "2026-02-18T12:00:00.000Z",
  "reqId": "run-001",
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
