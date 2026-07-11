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

type BackendDeviceDTO = {
  id: string;
  type: string;
  info: Record<string, unknown>;
};

export function resolveSimulationWsUrl() {
  const fromEnv = process.env.NEXT_PUBLIC_SIM_WS_URL?.trim();
  if (fromEnv) return fromEnv;

  const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  if (!apiBase) return "ws://127.0.0.1:8080/ws/simulation";

  try {
    const url = new URL(apiBase);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    if (url.pathname === "/" || url.pathname === "") url.pathname = "/ws/simulation";
    return url.toString().replace(/\/$/, "");
  } catch {
    return "ws://127.0.0.1:8080/ws/simulation";
  }
}

export function buildSimulationStartPayload(args: {
  rooms: Room[];
  markers: DeviceMarker[];
  scenarios: Scenario[];
  deviceIds?: string[];
  deviceTypes?: Record<string, string | undefined>;
  speed: number;
  fireIncident?: { id: string; x: number; y: number; roomId: string };
}) {
  const deviceIds = Array.from(new Set([...args.scenarios.flatMap((scenario) => scenario.chain), ...(args.deviceIds ?? [])]));
  const markerMap = new Map(args.markers.map((marker) => [marker.id, marker]));
  const devices: BackendDeviceDTO[] = deviceIds.map((id) => {
    const marker = markerMap.get(id);
    const metric = marker ? toMetricPoint(marker) : undefined;

    return {
      id,
      type: deviceTypeForBackend(id, args.deviceTypes?.[id]),
      info: {
        id,
        delay: 0,
        turned_on: false,
        turn_on: false,
        x: metric?.x,
        y: metric?.y,
        radius: sensorRadiusForBackend(id, args.deviceTypes?.[id]),
      },
    };
  });

  if (args.fireIncident) {
    const fire = args.fireIncident;
    const metric = toMetricPoint(fire);
    devices.push({
      id: fire.id,
      type: "fire",
      info: {
        id: fire.id,
        delay: 0,
        turned_on: false,
        turn_on: false,
        x: metric.x,
        y: metric.y,
        roomID: fire.roomId,
        radius: undefined,
      },
    });
  }

  return {
    dtSim: Math.max(0.1, 1 / Math.max(args.speed || 1, 0.1)),
    apartment: buildApartmentFloor(args.rooms),
    devices,
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

function toMetricPoint(point: { x: number; y: number }) {
  const width = 10;
  const height = 7;
  return {
    x: Number((point.x * width).toFixed(3)),
    y: Number((point.y * height).toFixed(3)),
  };
}

function buildApartmentFloor(rooms: Room[]) {
  const width = 10;
  const height = 7;
  const apiRooms = rooms.map((room) => {
    const x1 = room.x * width;
    const y1 = room.y * height;
    const x2 = (room.x + room.w) * width;
    const y2 = (room.y + room.h) * height;
    return {
      id: room.id,
      name: room.title,
      area: [
        [Number(x1.toFixed(3)), Number(y1.toFixed(3))],
        [Number(x2.toFixed(3)), Number(y1.toFixed(3))],
        [Number(x2.toFixed(3)), Number(y2.toFixed(3))],
        [Number(x1.toFixed(3)), Number(y2.toFixed(3))],
      ],
    };
  });

  return {
    meta: { units: "meters" },
    rooms: apiRooms,
    walls: [],
    doors: [],
    windows: [],
  };
}

function deviceTypeForBackend(id: string, type?: string) {
  const key = `${id} ${type ?? ""}`.toLowerCase();
  if (key.includes("motion") || key.includes("presence") || key.includes("pir") || key.includes("mmwave")) return "radiusMoveSensorWithoutUpdate";
  if (key.includes("smoke") || key.includes("co") || key.includes("gas") || key.includes("leak")) return "radiusMoveSensorWithoutUpdate";
  if (key.includes("lamp")) return "lamp";
  if (key.includes("siren")) return "lamp";
  if (key.includes("ventilation") || key.includes("valve") || key.includes("sprinkler")) return "lamp";
  if (key.includes("heater") || key.includes("ac") || key.includes("curtains") || key.includes("plug")) return "lamp";
  if (key.includes("switch") || key.includes("button") || key.includes("sensor")) return "lampSwitcher";
  if (key.includes("hub") || key.includes("gateway") || key.includes("controller")) return "lampSwitcher";
  return "lampSwitcher";
}

function sensorRadiusForBackend(id: string, type?: string) {
  const key = `${id} ${type ?? ""}`.toLowerCase();
  if (key.includes("motion") || key.includes("presence") || key.includes("pir") || key.includes("mmwave")) return 1.4;
  if (key.includes("smoke") || key.includes("co") || key.includes("gas")) return 1.2;
  if (key.includes("leak") || key.includes("water")) return 0.9;
  return undefined;
}
