const url = process.env.NEXT_PUBLIC_SIM_WS_URL ?? "ws://127.0.0.1:8080/ws/simulation";
const reqId = `sim-ui-smoke-${Date.now()}`;
let phase = "connecting";
let tick = 0;
let passed = false;
let edgeTriggered = false;

function send(ws, type, payload) {
  ws.send(JSON.stringify({ type, ts: new Date().toISOString(), reqId, payload }));
}

function apartment() {
  return {
    meta: { units: "meters" },
    walls: [],
    doors: [],
    windows: [],
    rooms: [
      {
        id: "room_smoke",
        name: "Smoke test room",
        area: [[0, 0], [2, 0], [2, 2], [0, 2]],
        walls: [],
        doors: [],
        windows: [],
      },
    ],
  };
}

const ws = new WebSocket(url);

ws.addEventListener("open", () => {
  phase = "hello";
  send(ws, "hello", { client: "sim-ui-smoke", version: "0.1.0", features: ["trigger"] });
});

ws.addEventListener("message", (event) => {
  const message = JSON.parse(event.data);

  if (message.reqId !== reqId) {
    throw new Error(`Unexpected reqId: ${message.reqId}`);
  }

  if (phase === "hello" && message.type === "hello:ack") {
    phase = "start";
    send(ws, "simulation:start", {
      dtSim: 0.05,
      apartment: apartment(),
      devices: [
        { id: "lampSwitcher_smoke", type: "switcher", info: { id: "lampSwitcher_smoke", delay: 0, turn_on: false } },
        { id: "lamp_smoke", type: "lamp", info: { id: "lamp_smoke", delay: 0, turn_on: false } },
      ],
      scenarios: [{ id: "lampSwitcher_smoke", edges: [{ to: "lamp_smoke", action: "trigger" }] }],
    });
    return;
  }

  if (phase === "start" && message.type === "simulation:started") {
    phase = "tick";
    send(ws, "simulation:tick", {
      tick: ++tick,
      inputs: [
        {
          entity_id: "lampSwitcher_smoke",
          payload: {
            kind: "device:trigger",
            turn_on: true,
            devices_payload: ["lampSwitcher_smoke"],
          },
        },
      ],
    });
    return;
  }

  if (phase === "tick" && message.type === "simulation:step") {
    if (message.payload?.tick !== tick) {
      throw new Error(`Unexpected simulation tick: ${message.payload?.tick}, expected ${tick}`);
    }
    for (const field of ["stateChanges", "triggeredEdges", "humans"]) {
      if (!Array.isArray(message.payload?.[field])) {
        throw new Error(`simulation:step.${field} must be an array`);
      }
    }
    const changes = message.payload?.stateChanges ?? [];
    const edges = message.payload?.triggeredEdges ?? [];
    edgeTriggered ||= edges.some(
      (edge) =>
        edge.from === "lampSwitcher_smoke" &&
        edge.to === "lamp_smoke" &&
        edge.action === "trigger"
    );
    const lampChanged = changes.some((change) => (change.entity_id ?? change.entityId) === "lamp_smoke");

    if (lampChanged && edgeTriggered) {
      phase = "stop";
      send(ws, "simulation:stop");
      return;
    }

    if (tick < 4) {
      send(ws, "simulation:tick", { tick: ++tick, inputs: [] });
      return;
    }

    throw new Error("WebSocket smoke test did not receive lamp state change and triggered edge");
  }

  if (phase === "stop" && message.type === "simulation:stopped") {
    passed = true;
    clearTimeout(timeout);
    console.log("WebSocket smoke test passed");
    ws.close();
  }

  if (message.type === "error") {
    throw new Error(`Backend error: ${JSON.stringify(message.payload)}`);
  }
});

ws.addEventListener("error", () => {
  throw new Error(`Cannot connect to ${url}`);
});

const timeout = setTimeout(() => {
  throw new Error(`WebSocket smoke test timed out in phase: ${phase}`);
}, 5000);

ws.addEventListener("close", () => {
  clearTimeout(timeout);
  process.exit(passed ? 0 : 1);
});
