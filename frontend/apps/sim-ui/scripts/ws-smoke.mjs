const url = process.env.NEXT_PUBLIC_SIM_WS_URL ?? "ws://127.0.0.1:8080/ws/simulation";
const reqId = `sim-ui-smoke-${Date.now()}`;
let phase = "connecting";
let tick = 0;
let passed = false;

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
        id: "room_1",
        name: "Smoke room",
        area: [
          [0, 0],
          [4, 0],
          [4, 4],
          [0, 4],
        ],
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

  if (phase === "hello" && message.type === "hello:ack") {
    phase = "start";
    send(ws, "simulation:start", {
      dtSim: 1,
      apartment: apartment(),
      devices: [
        { id: "lampSwitcher_smoke", type: "lampSwitcher", info: { id: "lampSwitcher_smoke", delay: 0, turned_on: false } },
        { id: "lamp_smoke", type: "lamp", info: { id: "lamp_smoke", delay: 0, turned_on: false } },
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
            kind: "human:trigger",
            turn_on: true,
            trigger: "lampSwitcher_smoke",
          },
        },
      ],
    });
    return;
  }

  if (phase === "tick" && message.type === "simulation:step") {
    const changes = message.payload?.stateChanges ?? [];
    const lampChanged = changes.some((change) => (change.entityId ?? change.entity_id) === "lamp_smoke");

    if (lampChanged) {
      phase = "stop";
      send(ws, "simulation:stop");
      return;
    }

    if (tick < 4) {
      send(ws, "simulation:tick", { tick: ++tick, inputs: [] });
      return;
    }

    throw new Error("WebSocket smoke test did not receive lamp state change");
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
