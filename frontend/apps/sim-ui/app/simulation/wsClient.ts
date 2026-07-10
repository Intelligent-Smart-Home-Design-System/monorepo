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

export type IncidentKind = "fire:spread" | "flood:spread" | "smoke:spread";

export type IncidentBlock = {
  id: string;
  roomID?: string;
  roomId?: string;
  x: number;
  y: number;
  size: number;
  intensity?: number;
  points: Array<[number, number]>;
};

export type IncidentZone = {
  roomID?: string;
  roomId?: string;
  blocks?: IncidentBlock[];
};

export type IncidentStatePayload = {
  kind?: string;
  incidents?: IncidentZone[];
};

export type LogEventPayload = {
  level?: LogLevel;
  device?: string;
  message?: string;
};

export function resolveSimulationWsUrl() {
  const token = getStoredAccessToken();
  if (!token) return null;

  const fromEnv = process.env.NEXT_PUBLIC_SIM_WS_URL?.trim();
  if (fromEnv) return withToken(fromEnv, token);

  const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  const base = apiBase || defaultGatewayOrigin();
  if (!base) return null;

  try {
    const url = new URL(base);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.pathname = "/api/v1/simulation/ws";
    url.search = "";
    return withToken(url.toString(), token);
  } catch {
    return null;
  }
}

function getStoredAccessToken() {
  if (typeof window === "undefined") return "";
  try {
    const raw = window.localStorage.getItem("smart-home-auth");
    if (!raw) return "";
    const parsed = JSON.parse(raw) as {
      tokens?: {
        access_token?: unknown;
      };
    };
    const token = parsed.tokens?.access_token;
    return typeof token === "string" ? token.trim() : "";
  } catch {
    return "";
  }
}

function browserOrigin() {
  if (typeof window === "undefined") return "";
  return window.location.origin;
}

function defaultGatewayOrigin() {
  const origin = browserOrigin();
  if (!origin) return "";

  const url = new URL(origin);
  if (url.port === "3001" || url.port === "3101") url.port = "8090";
  return url.toString();
}

function withToken(value: string, token: string) {
  const url = new URL(value, browserOrigin() || undefined);
  url.searchParams.set("token", token);
  return url.toString();
}

export function buildSimulationStartPayload(args: {
  floorSource: unknown;
  rooms: Room[];
  markers: DeviceMarker[];
  scenarios: Scenario[];
  deviceIds?: string[];
  deviceTypes?: Record<string, string | undefined>;
  speed: number;
}) {
  const deviceIds = Array.from(new Set([...args.scenarios.flatMap((scenario) => scenario.chain), ...(args.deviceIds ?? [])]));
  const markerMap = new Map(args.markers.map((marker) => [marker.id, marker]));
  const floorCoordinates = makeFloorCoordinateMapper(args.floorSource);
  const regularDevices = deviceIds.filter((id) => id !== "fire" && id !== "flood" && id !== "smoke");

  return {
    dtSim: Math.max(0.1, 1 / Math.max(args.speed || 1, 0.1)),
    apartment: simulationFloor(args.floorSource, args.rooms),
    devices: [
      ...regularDevices.map((id) => {
      const marker = markerMap.get(id);
      const position = marker ? floorCoordinates.toFloor(marker) : undefined;
      return {
        id,
        type: deviceTypeForBackend(id, args.deviceTypes?.[id]),
        info: {
          id,
          delay: 0,
          turned_on: false,
          x: position?.x,
          y: position?.y,
          radius: floorCoordinates.cellSize * 1.5,
        },
      };
      }),
      ...(["fire", "flood", "smoke"] as const).map((type) => ({
        id: type,
        type,
        info: { id: type, cellSize: floorCoordinates.cellSize },
      })),
    ],
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

export function buildIncidentActivation(floorSource: unknown, point: { x: number; y: number }, roomID: string) {
  const mapper = makeFloorCoordinateMapper(floorSource);
  const origin = mapper.toFloor(point);
  return { turn_on: true, x: origin.x, y: origin.y, roomID };
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

function simulationFloor(source: unknown, rooms: Room[]) {
  const floor = unwrapSimulationFloor(source);
  if (floor) {
    return {
      ...floor,
      meta: floor.meta && typeof floor.meta === "object" ? floor.meta : { units: "unknown" },
      walls: Array.isArray(floor.walls) ? floor.walls : [],
      doors: Array.isArray(floor.doors) ? floor.doors : [],
      windows: Array.isArray(floor.windows) ? floor.windows : [],
      rooms: floor.rooms,
    };
  }

  return {
    meta: { units: "ratio" },
    walls: [],
    doors: [],
    windows: [],
    rooms: rooms.map((room) => ({
      id: room.id,
      name: room.title,
      area: [
        [room.x, room.y],
        [room.x + room.w, room.y],
        [room.x + room.w, room.y + room.h],
        [room.x, room.y + room.h],
      ],
      walls: [],
      doors: [],
      windows: [],
    })),
  };
}

function unwrapSimulationFloor(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const record = value as Record<string, unknown>;
  if (Array.isArray(record.rooms) && Array.isArray(record.walls)) return record;

  for (const key of ["floor", "floorJson", "parsedFloor", "floor_plan", "parsed_floor_plan", "apartment", "plan"]) {
    const floor = unwrapSimulationFloor(record[key]);
    if (floor) return floor;
  }
  return null;
}

function makeFloorCoordinateMapper(source: unknown) {
  const points = collectCoordinatePoints(source);
  const rawScale = points.some(([x, y]) => Math.abs(x) > 1 || Math.abs(y) > 1);
  if (!rawScale || !points.length) {
    return { toFloor: (point: { x: number; y: number }) => point, cellSize: 0.05 };
  }

  const xs = points.map(([x]) => x);
  const ys = points.map(([, y]) => y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const width = Math.max(maxX - minX, 1);
  const height = Math.max(maxY - minY, 1);
  const padding = 0.04;
  const scale = 1 - padding * 2;

  return {
    toFloor: (point: { x: number; y: number }) => ({
      x: minX + ((point.x - padding) / scale) * width,
      y: minY + ((point.y - padding) / scale) * height,
    }),
    cellSize: Math.min(width, height) * 0.05,
  };
}

function collectCoordinatePoints(source: unknown) {
  const points: Array<[number, number]> = [];

  const visit = (value: unknown) => {
    if (!value || typeof value !== "object") return;
    if (Array.isArray(value)) {
      if (value.length >= 2 && typeof value[0] === "number" && typeof value[1] === "number") {
        if (Number.isFinite(value[0]) && Number.isFinite(value[1])) points.push([value[0], value[1]]);
        return;
      }
      value.forEach(visit);
      return;
    }

    const record = value as Record<string, unknown>;
    if (typeof record.x === "number" && Number.isFinite(record.x) && typeof record.y === "number" && Number.isFinite(record.y)) {
      points.push([record.x, record.y]);
    }
    Object.values(record).forEach(visit);
  };

  visit(source);
  return points;
}

function deviceTypeForBackend(id: string, type?: string) {
  const key = `${id} ${type ?? ""}`.toLowerCase();
  if (key.includes("smoke_sensor") || key.includes("smoke sensor")) return "smokeSensor";
  if (key.includes("fire_sensor") || key.includes("fire sensor")) return "fireSensor";
  if (key.includes("leak_sensor") || key.includes("flood_sensor") || key.includes("water_leak")) return "floodSensor";
  if (key.includes("lamp")) return "lamp";
  if (key.includes("siren")) return "lamp";
  if (key.includes("ventilation") || key.includes("valve") || key.includes("sprinkler")) return "lamp";
  if (key.includes("heater") || key.includes("ac") || key.includes("curtains") || key.includes("plug")) return "lamp";
  if (key.includes("switch") || key.includes("button") || key.includes("sensor")) return "lampSwitcher";
  if (key.includes("hub") || key.includes("gateway") || key.includes("controller")) return "lampSwitcher";
  return "lampSwitcher";
}
