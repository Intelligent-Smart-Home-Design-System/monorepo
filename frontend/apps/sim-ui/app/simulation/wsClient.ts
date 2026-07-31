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
  triggeredEdges?: Array<{ from: string; to: string; action?: string; data?: unknown[] }>;
  humans?: Array<{ id: string; type: string; info?: unknown }>;
};

export type HumanMoveState = {
  position: { x: number; y: number };
  roomID?: string;
  status?: string;
};

export type HumanRouteState = {
  status: "running" | "paused" | "stopped" | "completed" | "blocked" | "idle" | string;
};

export type DeviceLevelState = {
  value: number;
  percent: number;
};

export type DeviceValueControl = {
  value: number;
  min: number;
  max: number;
  step: number;
  unit: string;
  label: string;
};

export type IncidentKind = "fire:spread" | "flood:spread" | "smoke:spread";

export const SIMULATION_TICK_INTERVAL_MS = 50;
export const SIMULATION_DT_SECONDS = SIMULATION_TICK_INTERVAL_MS / 1000;

export type IncidentBlock = {
  id: string;
  roomID?: string;
  roomId?: string;
  x: number;
  y: number;
  size: number;
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

type BackendDeviceMode = "boolean" | "percent" | "observed";

type BackendDeviceDescriptor = {
  type: string;
  mode: BackendDeviceMode;
};

export function resolveSimulationWsUrl() {
  const token = getStoredAccessToken();
  if (!token) return null;

  const fromEnv = process.env.NEXT_PUBLIC_SIM_WS_URL?.trim();
  if (fromEnv) return withToken(fromEnv, token);

  const base = defaultGatewayOrigin();
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
  human: {
    id: string;
    position: { x: number; y: number };
    roomID: string;
  };
  dependencies?: Record<string, string[]>;
}) {
  const deviceIds = Array.from(new Set([...args.scenarios.flatMap((scenario) => scenario.chain), ...(args.deviceIds ?? [])]));
  const markerMap = new Map(args.markers.map((marker) => [marker.id, marker]));
  const floorCoordinates = makeFloorCoordinateMapper(args.floorSource);
  const humanPosition = floorCoordinates.toFloor(args.human.position);
  const regularDevices = deviceIds.filter((id) => id !== "fire" && id !== "flood" && id !== "smoke");
  let backendScenarios = [];

  if (args.dependencies && Object.keys(args.dependencies).length > 0) {
    backendScenarios = Object.entries(args.dependencies).map(([triggerId, targetIds]) => ({
      id: triggerId,
      edges: targetIds.map((targetId) => ({
        to: targetId,
        action: "trigger",
      })),
    }));
  } else {
    backendScenarios = args.scenarios.flatMap((scenario) => {
      return scenario.chain.slice(0, -1).map((id, index) => ({
        id,
        edges: [
          {
            to: scenario.chain[index + 1],
            action: "trigger",
          },
        ],
      }));
    });
  }

  return {
    dtSim: SIMULATION_DT_SECONDS,
    apartment: simulationFloor(args.floorSource, args.rooms),
    devices: [
      ...regularDevices.map((id) => {
        const marker = markerMap.get(id);
        const position = marker ? floorCoordinates.toFloor(marker) : undefined;
        const descriptor = backendDeviceDescriptor(id, args.deviceTypes?.[id]);
        return {
          id,
          type: descriptor.type,
          info: {
            id,
            delay: 0,
            turn_on: false,
            percents: 0,
            temperature: 20,
            timeout: 1,
            x: position?.x,
            y: position?.y,
            radius: floorCoordinates.cellSize * 1.5,
          },
        };
      }),
      {
        id: args.human.id,
        type: "human",
        info: {
          id: args.human.id,
          x: humanPosition.x,
          y: humanPosition.y,
          roomID: args.human.roomID,
          routeStep: floorCoordinates.cellSize * 3,
        },
      },
      ...(["fire", "flood", "smoke"] as const).map((type) => ({
        id: type,
        type,
        info: { id: type, cellSize: floorCoordinates.cellSize },
      })),
    ],
    scenarios: backendScenarios,
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

export function buildDeviceToggleInput(
  deviceId: string,
  deviceType: string | undefined,
  turnOn: boolean
): SimEventInput | null {
  const descriptor = backendDeviceDescriptor(deviceId, deviceType);
  if (descriptor.mode === "observed") return null;

  return {
    kind: "device:trigger",
    entityId: deviceId,
    payload: { turn_on: turnOn },
  };
}

export function deviceUsesPercentControl(deviceId: string, deviceType?: string) {
  return backendDeviceDescriptor(deviceId, deviceType).mode === "percent";
}

export function buildDevicePercentInput(
  deviceId: string,
  deviceType: string | undefined,
  value: number
): SimEventInput | null {
  const descriptor = backendDeviceDescriptor(deviceId, deviceType);
  if (descriptor.mode !== "percent" || !Number.isFinite(value)) return null;

  return {
    kind: "device:trigger",
    entityId: deviceId,
    payload: { percents: Math.min(100, Math.max(0, Math.round(value))) },
  };
}

export function deviceValueControl(
  deviceId: string,
  deviceType?: string
): Omit<DeviceValueControl, "value"> & { defaultValue: number } | undefined {
  const type = backendDeviceDescriptor(deviceId, deviceType).type;

  if (type === "airConditioner") {
    return { min: 16, max: 30, step: 1, unit: "°C", label: "Температура", defaultValue: 20 };
  }
  if (type === "smartFloor") {
    return { min: 5, max: 40, step: 1, unit: "°C", label: "Температура", defaultValue: 20 };
  }
  if (type === "thermostat") {
    return { min: 0, max: 100, step: 1, unit: "%", label: "Уровень нагрева", defaultValue: 20 };
  }

  return undefined;
}

export function buildDeviceValueInput(
  deviceId: string,
  deviceType: string | undefined,
  value: number
): SimEventInput | null {
  const control = deviceValueControl(deviceId, deviceType);
  if (!control || !Number.isFinite(value)) return null;

  const bounded = Math.min(control.max, Math.max(control.min, value));
  return {
    kind: "device:trigger",
    entityId: deviceId,
    payload: { temperature: control.step >= 1 ? Math.round(bounded) : bounded },
  };
}

export function readDeviceValueState(
  payload: unknown,
  deviceId: string,
  deviceType?: string
) {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return undefined;
  if (!deviceValueControl(deviceId, deviceType)) return undefined;
  return numericStateField(payload as Record<string, unknown>, ["temperature"]);
}

export function buildHumanMoveInput(
  floorSource: unknown,
  humanID: string,
  position: { x: number; y: number }
): SimEventInput {
  const target = makeFloorCoordinateMapper(floorSource).toFloor(position);
  return {
    kind: "human:move",
    entityId: humanID,
    payload: { to: target },
  };
}

export function buildHumanRouteInput(
  floorSource: unknown,
  humanID: string,
  points: Array<{ x: number; y: number }>,
  speed: number
): SimEventInput {
  const mapper = makeFloorCoordinateMapper(floorSource);
  return {
    kind: "human:route",
    entityId: humanID,
    payload: {
      action: "start",
      route: points.map((point) => mapper.toFloor(point)),
      speed,
    },
  };
}

export function buildHumanRouteControlInput(
  humanID: string,
  action: "pause" | "resume" | "stop"
): SimEventInput {
  return {
    kind: "human:route",
    entityId: humanID,
    payload: { action },
  };
}

export function readHumanMoveState(payload: unknown, floorSource: unknown): HumanMoveState | null {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return null;
  const state = payload as Record<string, unknown>;
  if (state.kind !== "human:move" && state.status !== "moved" && state.status !== "No move") return null;
  if (!state.to || typeof state.to !== "object" || Array.isArray(state.to)) return null;

  const target = state.to as Record<string, unknown>;
  if (typeof target.x !== "number" || !Number.isFinite(target.x) || typeof target.y !== "number" || !Number.isFinite(target.y)) {
    return null;
  }

  return {
    position: makeFloorCoordinateMapper(floorSource).toView({ x: target.x, y: target.y }),
    roomID: typeof state.roomID === "string" ? state.roomID : undefined,
    status: typeof state.status === "string" ? state.status : undefined,
  };
}

export function readHumanRouteState(payload: unknown): HumanRouteState | null {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return null;
  const state = payload as Record<string, unknown>;
  if (state.kind !== "human:route" || typeof state.status !== "string") return null;
  return { status: state.status };
}

export function readDeviceActiveState(payload: unknown) {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return undefined;
  const state = payload as Record<string, unknown>;

  if (typeof state.turn_on === "boolean") return state.turn_on;
  if (typeof state.state === "string") {
    const value = state.state.toLowerCase();
    if (value === "active" || value === "on") return true;
    if (value === "idle" || value === "off") return false;
  }

  return undefined;
}

export function readDevicePercentState(payload: unknown) {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return undefined;
  const value = (payload as Record<string, unknown>).percents;
  if (typeof value !== "number" || !Number.isFinite(value)) return undefined;
  return Math.min(100, Math.max(0, Math.round(value)));
}

export function readDeviceLevelState(
  payload: unknown,
  deviceId: string,
  deviceType?: string
): DeviceLevelState | undefined {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) return undefined;
  const state = payload as Record<string, unknown>;
  const key = `${deviceType ?? ""} ${deviceId}`.toLowerCase().replaceAll("-", "_");

  if (key.includes("temperature_sensor") || key.includes("temp_sensor")) {
    return physicalLevelState(state, ["temperature", "temperature_c", "value"], -40, 40);
  }
  if (key.includes("humidity")) {
    return physicalLevelState(state, ["humidity", "humidity_percent", "value"], 0, 100);
  }
  if (key.includes("co2")) {
    return physicalLevelState(state, ["co2", "co2_ppm", "ppm", "value"], 0, 50_000);
  }
  if (key.includes("illumination") || key.includes("lux") || key.includes("light_sensor")) {
    return physicalLevelState(state, ["illuminance", "lux", "value"], 0, 100_000);
  }
  if (key.includes("noise") || key.includes("sound")) {
    return physicalLevelState(state, ["noise", "noise_db", "decibels", "value"], 0, 120);
  }
  if (key.includes("pressure")) {
    return physicalLevelState(state, ["pressure", "pressure_hpa", "value"], 300, 1_100);
  }
  return undefined;
}

function physicalLevelState(
  state: Record<string, unknown>,
  fields: string[],
  min: number,
  max: number
) {
  const value = numericStateField(state, fields);
  return value === undefined ? undefined : levelState(value, min, max);
}

function numericStateField(state: Record<string, unknown>, fields: string[]) {
  for (const field of fields) {
    const value = state[field];
    if (typeof value === "number" && Number.isFinite(value)) return value;
  }
  return undefined;
}

function levelState(value: number, min: number, max: number): DeviceLevelState {
  const bounded = Math.min(max, Math.max(min, value));
  return {
    value,
    percent: ((bounded - min) / (max - min)) * 100,
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
    return {
      toFloor: (point: { x: number; y: number }) => point,
      toView: (point: { x: number; y: number }) => point,
      cellSize: 0.05,
    };
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
    toView: (point: { x: number; y: number }) => ({
      x: padding + ((point.x - minX) / width) * scale,
      y: padding + ((point.y - minY) / height) * scale,
    }),
    cellSize: Math.min(width, height) * 0.0125,
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

function backendDeviceDescriptor(id: string, type?: string): BackendDeviceDescriptor {
  const key = `${type ?? ""} ${id}`.toLowerCase().replaceAll("-", "_");

  if (key.includes("smoke_sensor") || key.includes("gas_leak_sensor")) {
    return { type: "smokeSensor", mode: "observed" };
  }
  if (key.includes("fire_sensor")) return { type: "fireSensor", mode: "observed" };
  if (key.includes("flood_sensor") || key.includes("water_leak_sensor") || key.includes("water_leak")) {
    return { type: "floodSensor", mode: "observed" };
  }
  if (key.includes("motion_sensor") || key.includes("presence_sensor") || key.includes("pir") || key.includes("mmwave")) {
    return { type: "radiusMoveSensorWithUpdate", mode: "observed" };
  }
  if (key.includes("camera")) return { type: "camera", mode: "observed" };
  if (key.includes("illumination_sensor") || key.includes("light_sensor") || key.includes("lux_sensor")) {
    return { type: "sensorWithIntStatus", mode: "observed" };
  }
  if (key.includes("sensor") || key.includes("wireless_button") || key.includes("button")) {
    return { type: "sensorWithoutUpdate", mode: "boolean" };
  }

  if (key.includes("smart_dimmer") || key.includes("dimmer")) return { type: "smartDimmer", mode: "percent" };
  if (key.includes("curtain")) return { type: "smartCurtains", mode: "percent" };
  if (key.includes("smart_lamp") || key.includes("smart_bulb")) return { type: "smartLamp", mode: "percent" };
  if (key.includes("lamp") || key.includes("light")) return { type: "lamp", mode: "boolean" };

  if (key.includes("siren")) return { type: "siren", mode: "boolean" };
  if (key.includes("smart_lock") || key.includes("lock")) return { type: "smartLock", mode: "boolean" };
  if (key.includes("doorbell")) return { type: "smartDoorbell", mode: "boolean" };
  if (key.includes("window")) return { type: "window", mode: "boolean" };
  if (key.includes("door")) return { type: "door", mode: "boolean" };

  if (key.includes("air_conditioner") || key.includes("conditioner")) return { type: "airConditioner", mode: "boolean" };
  if (key.includes("thermostat") || key.includes("radiator") || key.includes("heater")) {
    return { type: "thermostat", mode: "boolean" };
  }
  if (key.includes("smart_floor") || key.includes("heated_floor")) return { type: "smartFloor", mode: "boolean" };
  if (key.includes("smart_tv") || key.includes("television") || key.includes(" tv")) return { type: "tv", mode: "boolean" };
  if (key.includes("subwoofer") || key.includes("smart_speaker") || key.includes("speaker")) {
    return { type: "subwoofer", mode: "boolean" };
  }

  return { type: "switcher", mode: "boolean" };
}
