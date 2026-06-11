import type { DeviceMarker, LogLevel, Room, Scenario } from "@/app/simulation/Mockdata";

export type WsStatus = "disabled" | "connecting" | "connected" | "disconnected" | "error";

export type WsEnvelope<T = unknown> = {
  type: string;
  ts: string;
  reqId?: string;
  payload?: T;
};

export type SimEventInput = {
  kind?: string;
  entityId: string;
  trigger?: string;
  devicesPayload?: string[];
  payload: Record<string, unknown>;
};

export type SimStateChange = {
  kind?: string;
  entityId?: string;
  entity_id?: string;
  payload?: unknown;
};

export type SimStepPayload = {
  tick: number;
  simTime?: number;
  stateChanges?: SimStateChange[];
  triggeredEdges?: Array<{ to: string; action?: string; data?: unknown[] }>;
  humans?: Array<{ id: string; type: string; info?: unknown }>;
};

export type LogEventPayload = {
  level?: LogLevel;
  device?: string;
  message?: string;
};

export function resolveSimulationWsUrl() {
  const fromEnv = process.env.NEXT_PUBLIC_SIM_WS_URL?.trim();
  if (fromEnv) return fromEnv;

  const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  if (!apiBase) return "ws://127.0.0.1:8080";

  try {
    const url = new URL(apiBase);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    return url.toString().replace(/\/$/, "");
  } catch {
    return "ws://127.0.0.1:8080";
  }
}

export function buildSimulationStartPayload(args: {
  rooms: Room[];
  markers: DeviceMarker[];
  scenarios: Scenario[];
  deviceIds?: string[];
  deviceTypes?: Record<string, string | undefined>;
  speed: number;
}) {
  const deviceIds = Array.from(new Set([...args.scenarios.flatMap((scenario) => scenario.chain), ...(args.deviceIds ?? [])]));
  const markerMap = new Map(args.markers.map((marker) => [marker.id, marker]));

  return {
    dtSim: Math.max(0.1, 1 / Math.max(args.speed || 1, 0.1)),
    apartment: buildApartmentField(),
    devices: deviceIds.map((id) => {
      const marker = markerMap.get(id);
      return {
        id,
        type: deviceTypeForBackend(id, args.deviceTypes?.[id]),
        info: {
          id,
          delay: 0,
          turned_on: false,
          x: marker?.x,
          y: marker?.y,
        },
      };
    }),
    scenarios: args.scenarios.flatMap((scenario) => {
      return scenario.chain.slice(0, -1).map((id, index) => ({
        id,
        edges: [
          {
            to: scenario.chain[index + 1],
            action: "trigger",
          },
        ],
      }));
    }),
  };
}

export function buildTickPayload(tick: number, inputs: SimEventInput[] = []) {
  return {
    tick,
    inputs: inputs.map((input) => {
      const kind = input.kind ?? "user";
      return {
        entity_id: input.entityId,
        payload: {
          kind,
          ...(input.trigger ? { trigger: input.trigger } : {}),
          ...input.payload,
          ...(input.devicesPayload?.length ? { devices_payload: input.devicesPayload } : {}),
        },
      };
    }),
  };
}

export function normalizeLogLevel(level: unknown): LogLevel {
  return level === "WARNING" || level === "ERROR" || level === "INFO" ? level : "INFO";
}

function buildApartmentField() {
  const width = 10;
  const height = 7;
  return {
    width,
    height,
    cells: Array.from({ length: height }, (_, y) =>
      Array.from({ length: width }, (_, x) => ({
        x,
        y,
        condition: false,
      }))
    ),
  };
}

function deviceTypeForBackend(id: string, type?: string) {
  const key = `${id} ${type ?? ""}`.toLowerCase();
  if (key.includes("lamp")) return "lamp";
  if (key.includes("siren")) return "lamp";
  if (key.includes("ventilation") || key.includes("valve") || key.includes("sprinkler")) return "lamp";
  if (key.includes("heater") || key.includes("ac") || key.includes("curtains") || key.includes("plug")) return "lamp";
  if (key.includes("switch") || key.includes("button") || key.includes("sensor")) return "lampSwitcher";
  if (key.includes("hub") || key.includes("gateway") || key.includes("controller")) return "lampSwitcher";
  return "lampSwitcher";
}
