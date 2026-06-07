"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { ControlPanel } from "@/app/components/sim/ControlPanel";
import { ApartmentPlan } from "@/app/components/sim/ApartmentPlan";
import { EventConsole } from "@/app/components/sim/EventConsole";
import { Card } from "@/app/components/ui";
import floorPlanData from "@/app/simulation/floor.json";
import dependencyConfig from "../../../../../services/simulation/configs/dependencies.json";
import entityConfig from "../../../../../services/simulation/configs/entities.json";
import layoutDeviceConfig from "../../../../../services/layout/internal/configs/devices.json";
import { adaptFloorData, type FloorPlanView, type WallSegment } from "@/app/simulation/floorAdapter";
import {
  buildSimulationStartPayload,
  buildTickPayload,
  normalizeLogLevel,
  resolveSimulationWsUrl,
  type SimEventInput,
  type SimStepPayload,
  type WsEnvelope,
  type WsStatus,
} from "@/app/simulation/wsClient";

import {
  scenarios as MOCK_SCENARIOS,
  deviceMarkers,
  rooms as MOCK_ROOMS,
  type Scenario,
  type Device,
  type DeviceMarker,
  type LogEvent,
  type LogLevel,
} from "@/app/simulation/Mockdata";

type Status = "empty" | "loading" | "running" | "paused" | "error";
type Speed = number;
type Filter = "ALL" | LogLevel;
type RunMode = "parallel" | "sequence";
type Point = { x: number; y: number };
type ExternalDevice = {
  id: string;
  name?: string;
  type?: string;
  x?: number;
  y?: number;
};
type SavedPlanDevice = {
  id: string;
  x: number;
  y: number;
};
type DependencyConfig = {
  triggers: Record<string, { description?: string; triggers: string[] }>;
};
type EntityConfig = {
  entities: Record<string, { description?: string }>;
};
type LayoutDeviceConfig = {
  device_types: Record<string, { name?: string; tracks?: string[] }>;
};

const PLAN_STORAGE_KEY = "simulation-plan-layout";
const FLOOR_STORAGE_KEYS = ["simulation-floor", "planner-floor-json", "parsed-floor", "floor-json"];
const SIM_DEPENDENCIES = dependencyConfig as DependencyConfig;
const SIM_ENTITIES = entityConfig as EntityConfig;
const LAYOUT_DEVICES = layoutDeviceConfig as LayoutDeviceConfig;

function readStorage(key: string) {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeStorage(key: string, value: string) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Storage can be unavailable in some browser privacy modes.
  }
}

function removeStorage(key: string) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.removeItem(key);
  } catch {
    // Storage can be unavailable in some browser privacy modes.
  }
}

function loadFloorSourceFromStorage() {
  if (typeof window === "undefined") return floorPlanData as unknown;

  for (const key of FLOOR_STORAGE_KEYS) {
    try {
      const raw = readStorage(key);
      if (!raw) continue;
      return JSON.parse(raw) as unknown;
    } catch {
      // Ignore stale or unrelated values from other pages.
    }
  }

  return floorPlanData as unknown;
}

const FIRE_DEVICE_MARKERS: DeviceMarker[] = [
  { id: "smoke_sensor", x: 0.79, y: 0.25 },
  { id: "co_sensor", x: 0.52, y: 0.38 },
  { id: "siren", x: 0.18, y: 0.70 },
  { id: "ventilation", x: 0.84, y: 0.64 },
  { id: "sprinkler_kitchen", x: 0.82, y: 0.32 },
  { id: "sprinkler_living", x: 0.48, y: 0.36 },
];
const WATER_DEVICE_MARKERS: DeviceMarker[] = [
  { id: "leak_sensor", x: 0.82, y: 0.72 },
  { id: "leak_sensor_bath", x: 0.85, y: 0.82 },
  { id: "water_flow", x: 0.77, y: 0.64 },
  { id: "water_valve", x: 0.91, y: 0.76 },
];
function speedToDelay(speed: Speed) {
  const s = Math.max(Number(speed) || 1, 0.1);
  return Math.round(700 / s);
}

function crossesWall(from: Point, to: Point, wall: WallSegment) {
  const eps = 0.0001;

  if (wall.kind === "segment") {
    return segmentsIntersect(from, to, wall.from, wall.to);
  }

  if (wall.kind === "vertical") {
    if (Math.abs(to.x - from.x) < eps) return false;
    const crossesX = (from.x < wall.x && to.x >= wall.x) || (from.x > wall.x && to.x <= wall.x);
    if (!crossesX) return false;
    const t = (wall.x - from.x) / (to.x - from.x);
    if (t < 0 || t > 1) return false;
    const yAtWall = from.y + (to.y - from.y) * t;
    return yAtWall >= wall.y1 && yAtWall <= wall.y2;
  }

  if (Math.abs(to.y - from.y) < eps) return false;
  const crossesY = (from.y < wall.y && to.y >= wall.y) || (from.y > wall.y && to.y <= wall.y);
  if (!crossesY) return false;
  const t = (wall.y - from.y) / (to.y - from.y);
  if (t < 0 || t > 1) return false;
  const xAtWall = from.x + (to.x - from.x) * t;
  return xAtWall >= wall.x1 && xAtWall <= wall.x2;
}

function segmentsIntersect(a: Point, b: Point, c: Point, d: Point) {
  const eps = 0.0001;

  function orientation(p: Point, q: Point, r: Point) {
    const value = (q.y - p.y) * (r.x - q.x) - (q.x - p.x) * (r.y - q.y);
    if (Math.abs(value) < eps) return 0;
    return value > 0 ? 1 : 2;
  }

  function onSegment(p: Point, q: Point, r: Point) {
    return (
      q.x <= Math.max(p.x, r.x) + eps &&
      q.x + eps >= Math.min(p.x, r.x) &&
      q.y <= Math.max(p.y, r.y) + eps &&
      q.y + eps >= Math.min(p.y, r.y)
    );
  }

  const o1 = orientation(a, b, c);
  const o2 = orientation(a, b, d);
  const o3 = orientation(c, d, a);
  const o4 = orientation(c, d, b);

  if (o1 !== o2 && o3 !== o4) return true;
  if (o1 === 0 && onSegment(a, c, b)) return true;
  if (o2 === 0 && onSegment(a, d, b)) return true;
  if (o3 === 0 && onSegment(c, a, d)) return true;
  if (o4 === 0 && onSegment(c, b, d)) return true;
  return false;
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function normalizeExternalDevice(raw: unknown): ExternalDevice | null {
  if (!raw || typeof raw !== "object") return null;

  const item = raw as Record<string, unknown>;
  const idCandidate = item.id ?? item.device_id ?? item.deviceId ?? item.type ?? item.device_type ?? item.name;
  if (typeof idCandidate !== "string" || !idCandidate.trim()) return null;

  const x = isFiniteNumber(item.x) ? item.x : undefined;
  const y = isFiniteNumber(item.y) ? item.y : undefined;

  return {
    id: idCandidate.trim(),
    name: typeof item.name === "string" ? item.name : undefined,
    type: typeof item.type === "string" ? item.type : typeof item.device_type === "string" ? item.device_type : undefined,
    x: x !== undefined && x >= 0 && x <= 1 ? x : undefined,
    y: y !== undefined && y >= 0 && y <= 1 ? y : undefined,
  };
}

function loadExternalDevicesFromStorage(): ExternalDevice[] {
  if (typeof window === "undefined") return [];

  const fromUrl = loadExternalDevicesFromUrl();
  if (fromUrl.length) {
    writeStorage("simulation-devices", JSON.stringify(fromUrl));
    try {
      const params = new URLSearchParams(window.location.search);
      params.delete("devices");
      const nextQuery = params.toString();
      window.history.replaceState(null, "", nextQuery ? `${window.location.pathname}?${nextQuery}` : window.location.pathname);
    } catch {
      // URL cleanup is nice to have, not required for the simulation.
    }
    return fromUrl;
  }

  const keys = ["simulation-devices", "sim-devices", "selectedDevices", "selected-devices", "devices"];
  const seen = new Set<string>();
  const devices: ExternalDevice[] = [];

  keys.forEach((key) => {
    try {
      const raw = readStorage(key);
      if (!raw) return;

      const parsed = JSON.parse(raw) as unknown;
      const list = Array.isArray(parsed)
        ? parsed
        : parsed && typeof parsed === "object" && Array.isArray((parsed as { devices?: unknown[] }).devices)
        ? (parsed as { devices: unknown[] }).devices
        : [];

      list.forEach((item) => {
        const device = normalizeExternalDevice(item);
        if (!device || seen.has(device.id)) return;
        seen.add(device.id);
        devices.push(device);
      });
    } catch {
      // Ignore unrelated localStorage values from other pages.
    }
  });

  return devices;
}

function loadExternalDevicesFromUrl(): ExternalDevice[] {
  if (typeof window === "undefined") return [];

  try {
    const raw = new URLSearchParams(window.location.search).get("devices");
    if (!raw) return [];

    const parsed = JSON.parse(raw) as unknown;
    const list = Array.isArray(parsed)
      ? parsed
      : parsed && typeof parsed === "object" && Array.isArray((parsed as { devices?: unknown[] }).devices)
      ? (parsed as { devices: unknown[] }).devices
      : [];

    const seen = new Set<string>();
    return list.flatMap((item) => {
      const device = normalizeExternalDevice(item);
      if (!device || seen.has(device.id)) return [];
      seen.add(device.id);
      return [device];
    });
  } catch {
    return [];
  }
}

function loadSavedPlanDevices(): SavedPlanDevice[] {
  if (typeof window === "undefined") return [];

  try {
    const raw = readStorage(PLAN_STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as unknown;
    const list =
      parsed && typeof parsed === "object" && Array.isArray((parsed as { devices?: unknown[] }).devices)
        ? (parsed as { devices: unknown[] }).devices
        : Array.isArray(parsed)
        ? parsed
        : [];

    const seen = new Set<string>();
    return list.flatMap((item) => {
      const device = normalizeExternalDevice(item);
      if (!device || device.x === undefined || device.y === undefined || seen.has(device.id)) return [];
      seen.add(device.id);
      return [{ id: device.id, x: device.x, y: device.y }];
    });
  } catch {
    return [];
  }
}

function deviceKind(id: string, type?: string) {
  const key = `${id} ${type ?? ""}`.toLowerCase();
  if (key.includes("motion") || key.includes("presence") || key.includes("pir") || key.includes("mmwave")) return "motion";
  if (key.includes("door") || key.includes("window") || key.includes("open")) return "door";
  if (key.includes("leak") || key.includes("water_flow")) return "leak";
  if (key.includes("smoke")) return "smoke";
  if (key.includes("co2")) return "air";
  if (key.includes("co") || key.includes("gas")) return "gas";
  if (key.includes("lux") || key.includes("light_sensor")) return "lux";
  if (key.includes("button") || key.includes("switch")) return "button";
  if (key.includes("lamp") || key.includes("light")) return "lamp";
  if (key.includes("siren")) return "siren";
  if (key.includes("ventilation") || key.includes("fan")) return "ventilation";
  if (key.includes("valve")) return "valve";
  if (key.includes("heater") || key.includes("ac") || key.includes("curtains") || key.includes("plug")) return "actuator";
  if (key.includes("hub") || key.includes("gateway") || key.includes("controller")) return "bridge";
  if (key.includes("notification")) return "notification";
  return "other";
}

function canonicalEntityType(id: string, type?: string) {
  const key = `${id} ${type ?? ""}`.toLowerCase();

  if (key.includes("motion") || key.includes("pir") || key.includes("mmwave")) return "motion_sensor";
  if (key.includes("presence")) return "presence_sensor";
  if (key.includes("lux") || key.includes("illumination") || key.includes("light_sensor")) return "illumination_sensor";
  if (key.includes("leak") || key.includes("water_leak")) return "water_leak_sensor";
  if (key.includes("gas")) return "gas_leak_sensor";
  if (key.includes("doorbell")) return "smart_doorbell";
  if (key.includes("door")) return "door_sensor";
  if (key.includes("window")) return "window_sensor";
  if (key.includes("button") || key.includes("switch") || key.includes("scene")) return "wireless_button_switch";
  if (key.includes("lock")) return "smart_lock";
  if (key.includes("camera")) return "camera";
  if (key.includes("dimmer")) return "smart_dimmer";
  if (key.includes("curtain")) return "curtains";
  if (key.includes("backlight")) return "built_in_backlight";
  if (key.includes("decorative") || key.includes("luminaire")) return "decorative_luminaire";
  if (key.includes("lamp") || key.includes("bulb") || key.includes("light")) return "smart_bulb";
  if (key.includes("siren")) return "smart_siren";

  return SIM_ENTITIES.entities[type ?? ""] ? type : undefined;
}

function isConfigCompatible(trigger: string, target: string, deviceTypes: Record<string, string | undefined>) {
  const triggerType = canonicalEntityType(trigger, deviceTypes[trigger]);
  const targetType = canonicalEntityType(target, deviceTypes[target]);
  if (!triggerType || !targetType) return undefined;

  const dependency = SIM_DEPENDENCIES.triggers[triggerType];
  if (!dependency) return undefined;

  return dependency.triggers.includes(targetType);
}

function isConfigTrigger(id: string, deviceTypes: Record<string, string | undefined>) {
  const type = canonicalEntityType(id, deviceTypes[id]);
  return Boolean(type && SIM_DEPENDENCIES.triggers[type]?.triggers.length);
}

function isConfigTarget(id: string, deviceTypes: Record<string, string | undefined>) {
  const type = canonicalEntityType(id, deviceTypes[id]);
  if (!type) return false;
  return Object.values(SIM_DEPENDENCIES.triggers).some((dependency) => dependency.triggers.includes(type));
}

function buildScenarioTitle(trigger: string, target: string) {
  const triggerKind = deviceKind(trigger);
  const targetKind = deviceKind(target);

  if (triggerKind === "motion" && targetKind === "lamp") return `${trigger} → включить свет`;
  if (triggerKind === "door" && targetKind === "lamp") return `${trigger} → включить свет`;
  if (triggerKind === "door" && targetKind === "siren") return `${trigger} → тревога`;
  if (triggerKind === "leak" && targetKind === "valve") return `${trigger} → перекрыть воду`;
  if (triggerKind === "leak" && targetKind === "siren") return `${trigger} → аварийный сигнал`;
  if ((triggerKind === "smoke" || triggerKind === "gas" || triggerKind === "air") && targetKind === "siren") return `${trigger} → сирена`;
  if ((triggerKind === "smoke" || triggerKind === "gas" || triggerKind === "air") && targetKind === "ventilation") return `${trigger} → вентиляция`;
  if ((triggerKind === "lux" || triggerKind === "button") && targetKind === "lamp") return `${trigger} → свет`;
  return `${trigger} → ${target}`;
}

function scenarioCategoryFor(trigger: string, target: string): Scenario["category"] {
  const triggerKind = deviceKind(trigger);
  const targetKind = deviceKind(target);

  if (triggerKind === "smoke" || triggerKind === "gas" || triggerKind === "air") return "fire_gas";
  if (triggerKind === "leak" || targetKind === "valve") return "water";
  if (triggerKind === "door" && targetKind === "siren") return "security";
  if (targetKind === "lamp" || triggerKind === "motion" || triggerKind === "lux" || triggerKind === "button") return "lighting";
  return "service";
}

function buildPlacedScenarios(placedIds: string[], bridgeId: string | undefined, deviceTypes: Record<string, string | undefined>): Scenario[] {
  const placed = Array.from(new Set(placedIds));
  const triggers = placed.filter((id) => {
    const kind = deviceKind(id, deviceTypes[id]);
    return isConfigTrigger(id, deviceTypes) || ["motion", "door", "leak", "smoke", "gas", "air", "lux", "button"].includes(kind);
  });
  const targets = placed.filter((id) => {
    const kind = deviceKind(id, deviceTypes[id]);
    return isConfigTarget(id, deviceTypes) || ["lamp", "siren", "ventilation", "valve", "actuator", "notification"].includes(kind);
  });
  const scenarios: Scenario[] = [];

  triggers.forEach((trigger) => {
    targets.forEach((target) => {
      if (trigger === target) return;
      const triggerKind = deviceKind(trigger, deviceTypes[trigger]);
      const targetKind = deviceKind(target, deviceTypes[target]);
      const configCompatible = isConfigCompatible(trigger, target, deviceTypes);
      const compatible =
        configCompatible ??
        ((triggerKind === "motion" && targetKind === "lamp") ||
          (triggerKind === "door" && ["lamp", "siren", "notification"].includes(targetKind)) ||
          (triggerKind === "leak" && ["valve", "siren", "notification"].includes(targetKind)) ||
          (["smoke", "gas", "air"].includes(triggerKind) && ["siren", "ventilation", "notification"].includes(targetKind)) ||
          (["lux", "button"].includes(triggerKind) && ["lamp", "actuator"].includes(targetKind)));

      if (!compatible) return;

      const chain = bridgeId && bridgeId !== trigger && bridgeId !== target ? [trigger, bridgeId, target] : [trigger, target];
      scenarios.push({
        id: `placed_${chain.join("_to_")}`,
        title: buildScenarioTitle(trigger, target),
        description: "Собрано из устройств на плане",
        chain,
        category: scenarioCategoryFor(trigger, target),
      });
    });
  });

  return scenarios.slice(0, 40);
}

export default function SimulationPage() {
  const baseScenarios = useMemo<Scenario[]>(() => MOCK_SCENARIOS, []);
  const [floorSource] = useState<unknown>(() => loadFloorSourceFromStorage());
  const adaptedFloor = useMemo(() => adaptFloorData(floorSource, MOCK_ROOMS, deviceMarkers), [floorSource]);
  const roomsForPlan = adaptedFloor.rooms;
  const floorPlanForView: FloorPlanView = adaptedFloor.floorPlan;
  const baseDeviceMarkers = adaptedFloor.markers;
  const placementMarkers = adaptedFloor.placementMarkers;
  const blockingWalls = floorPlanForView.blockers?.length ? floorPlanForView.blockers : [];
  const [externalDevices] = useState<ExternalDevice[]>(() => loadExternalDevicesFromStorage());
  const [savedPlanDevices] = useState<SavedPlanDevice[]>(() => loadSavedPlanDevices());

  const [status, setStatus] = useState<Status>("empty");
  const [speed, setSpeed] = useState<Speed>(1);
  const [filter, setFilter] = useState<Filter>("ALL");
  const [search, setSearch] = useState("");

  const [runMode, setRunMode] = useState<RunMode>("parallel");

  const [events, setEvents] = useState<LogEvent[]>([]);
  const [activeNodes, setActiveNodes] = useState<string[]>([]);
  const [activeEdges, setActiveEdges] = useState<Array<[string, string]>>([]);
  const [manualDeviceState, setManualDeviceState] = useState<Record<string, boolean>>({});
  const [lastEvent, setLastEvent] = useState<LogEvent | null>(null);
  const [runScenarios, setRunScenarios] = useState<Scenario[]>([]);
  const runStepRef = useRef(-1);
  const runSeqIndexRef = useRef(0);
  const runSeqStepRef = useRef(-1);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const fireTimersRef = useRef<Array<ReturnType<typeof setTimeout>>>([]);
  const waterTimersRef = useRef<Array<ReturnType<typeof setTimeout>>>([]);
  const motionTimersRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({});
  const motionScenarioTimersRef = useRef<Array<ReturnType<typeof setTimeout>>>([]);
  const activeMotionSensorsRef = useRef<Set<string>>(new Set());
  const wsRef = useRef<WebSocket | null>(null);
  const wsReqIdRef = useRef("sim-ui");
  const wsTickRef = useRef(0);
  const backendRunActiveRef = useRef(false);
  const wsStartAckTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wsReconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const floorWarningsLoggedRef = useRef(false);
  const [devicePositions, setDevicePositions] = useState<DeviceMarker[]>(() => {
    const savedMarkers = savedPlanDevices.map((device) => ({ id: device.id, x: device.x, y: device.y }));
    const knownMarkerIds = new Set([...placementMarkers.map((marker) => marker.id), ...savedPlanDevices.map((device) => device.id)]);
    const externalMarkers = externalDevices
      .filter((device) => device.x !== undefined && device.y !== undefined && !knownMarkerIds.has(device.id))
      .map((device) => ({ id: device.id, x: device.x as number, y: device.y as number, label: device.name }));

    return [...baseDeviceMarkers, ...FIRE_DEVICE_MARKERS, ...WATER_DEVICE_MARKERS, ...externalMarkers, ...savedMarkers];
  });
  const [placedDeviceIds, setPlacedDeviceIds] = useState<string[]>(() => {
    const placementIds = placementMarkers.map((marker) => marker.id);
    const savedIds = savedPlanDevices.map((device) => device.id);
    const externalPlacedIds = externalDevices.filter((device) => device.x !== undefined && device.y !== undefined).map((device) => device.id);
    return Array.from(new Set([...placementIds, ...savedIds, ...externalPlacedIds]));
  });
  const [fireMode, setFireMode] = useState(false);
  const [firePoint, setFirePoint] = useState<Point | null>(null);
  const [firePoints, setFirePoints] = useState<Point[]>([]);
  const [fireActive, setFireActive] = useState(false);
  const [fireActiveDeviceIds, setFireActiveDeviceIds] = useState<string[]>([]);
  const [waterMode, setWaterMode] = useState(false);
  const [waterPoint, setWaterPoint] = useState<Point | null>(null);
  const [waterPoints, setWaterPoints] = useState<Point[]>([]);
  const [waterActive, setWaterActive] = useState(false);
  const [waterActiveDeviceIds, setWaterActiveDeviceIds] = useState<string[]>([]);
  const [motionActiveDeviceIds, setMotionActiveDeviceIds] = useState<string[]>([]);
  const [disasterMessage, setDisasterMessage] = useState<string | null>(null);
  const [wsStatus, setWsStatus] = useState<WsStatus>("connecting");

  const externalDeviceMap = useMemo(() => new Map(externalDevices.map((device) => [device.id, device])), [externalDevices]);
  const deviceTypeMap = useMemo<Record<string, string | undefined>>(() => {
    return Object.fromEntries(externalDevices.map((device) => [device.id, device.type]));
  }, [externalDevices]);
  const bridgeId = useMemo(() => placedDeviceIds.find((id) => deviceKind(id, externalDeviceMap.get(id)?.type) === "bridge"), [placedDeviceIds, externalDeviceMap]);
  const placedScenarios = useMemo(() => buildPlacedScenarios(placedDeviceIds, bridgeId, deviceTypeMap), [placedDeviceIds, bridgeId, deviceTypeMap]);
  const scenarios = useMemo<Scenario[]>(() => {
    const byId = new Map<string, Scenario>();
    const placedSet = new Set(placedDeviceIds);
    placedScenarios.forEach((scenario) => byId.set(scenario.id, scenario));
    baseScenarios
      .filter((scenario) => scenario.chain.every((id) => placedSet.has(id)))
      .forEach((scenario) => byId.set(scenario.id, scenario));
    return Array.from(byId.values());
  }, [baseScenarios, placedDeviceIds, placedScenarios]);
  const selectedScenarioIds = useMemo(() => scenarios.map((scenario) => scenario.id), [scenarios]);
  const selectedScenarios = scenarios;
  const availableDeviceIds = useMemo(() => {
    const ids = new Set<string>();

    if (externalDevices.length) {
      externalDevices.forEach((device) => ids.add(device.id));
      placedDeviceIds.forEach((id) => ids.add(id));
      return Array.from(ids);
    }

    Object.keys(LAYOUT_DEVICES.device_types ?? {}).forEach((id) => ids.add(id));
    baseScenarios.forEach((scenario) => scenario.chain.forEach((id) => ids.add(id)));
    scenarios.forEach((scenario) => scenario.chain.forEach((id) => ids.add(id)));
    return Array.from(ids);
  }, [baseScenarios, externalDevices, placedDeviceIds, scenarios]);

  const devicesForPlan = useMemo<Device[]>(() => {
    const activeSelectedIds = [...fireActiveDeviceIds, ...waterActiveDeviceIds, ...motionActiveDeviceIds].filter((id) =>
      placedDeviceIds.includes(id)
    );
    const ids = Array.from(new Set([...placedDeviceIds, ...activeSelectedIds]));

    return ids.map((id) => ({
      id,
      name: externalDeviceMap.get(id)?.name,
      type: externalDeviceMap.get(id)?.type,
      status:
        activeNodes.includes(id) ||
        manualDeviceState[id] ||
        fireActiveDeviceIds.includes(id) ||
        waterActiveDeviceIds.includes(id) ||
        motionActiveDeviceIds.includes(id)
          ? "active"
          : "idle",
    }));
  }, [placedDeviceIds, activeNodes, manualDeviceState, fireActiveDeviceIds, waterActiveDeviceIds, motionActiveDeviceIds, externalDeviceMap]);

  useEffect(() => {
    if (typeof window === "undefined") return;

    const markers = new Map(devicePositions.map((marker) => [marker.id, marker]));
    const devices = placedDeviceIds.flatMap((id) => {
      const marker = markers.get(id);
      if (!marker) return [];
      return [{ id, x: marker.x, y: marker.y }];
    });

    if (!devices.length) {
      removeStorage(PLAN_STORAGE_KEY);
      return;
    }

    writeStorage(PLAN_STORAGE_KEY, JSON.stringify({ version: 1, devices }));
  }, [devicePositions, placedDeviceIds]);

  function onMoveDevice(id: string, x: number, y: number) {
    setPlacedDeviceIds((ids) => (ids.includes(id) ? ids : [...ids, id]));
    setDevicePositions((prev) => {
      const idx = prev.findIndex((m) => m.id === id);
      if (idx === -1) return [...prev, { id, x, y }];
      const copy = prev.slice();
      copy[idx] = { ...copy[idx], x, y };
      return copy;
    });
  }

  function onDropDevice(id: string, x: number, y: number) {
    onMoveDevice(id, x, y);
    addEvent(id, "Устройство размещено на плане", "INFO");
  }

  function suggestedDevicePosition(id: string) {
    const roomTitle = id.toLowerCase();
    const room =
      roomsForPlan.find((item) => roomTitle.includes(item.id) || roomTitle.includes(item.title)) ??
      (roomTitle.includes("leak") || roomTitle.includes("water") ? roomsForPlan.find((item) => item.id === "bath") : undefined) ??
      (roomTitle.includes("smoke") || roomTitle.includes("gas") ? roomsForPlan.find((item) => item.id === "kitchen") : undefined) ??
      (roomTitle.includes("lamp") || roomTitle.includes("motion") || roomTitle.includes("door") ? roomsForPlan.find((item) => item.id === "hall") : undefined) ??
      roomsForPlan.find((item) => item.id === "living") ??
      roomsForPlan[0];
    const index = placedDeviceIds.length;
    const col = index % 3;
    const row = Math.floor(index / 3) % 3;

    return {
      x: Math.min(0.96, Math.max(0.04, room.x + room.w * (0.28 + col * 0.22))),
      y: Math.min(0.94, Math.max(0.06, room.y + room.h * (0.25 + row * 0.22))),
    };
  }

  function onPlaceDevice(id: string) {
    const marker = markerFor(id);
    const point = marker ?? suggestedDevicePosition(id);
    onDropDevice(id, point.x, point.y);
  }

  function onRemoveDevice(id: string) {
    setPlacedDeviceIds((ids) => ids.filter((deviceId) => deviceId !== id));
    setActiveNodes((ids) => ids.filter((deviceId) => deviceId !== id));
    setActiveEdges((edges) => edges.filter(([from, to]) => from !== id && to !== id));
    setManualDeviceState((state) => {
      const next = { ...state };
      delete next[id];
      return next;
    });
    setFireActiveDeviceIds((ids) => ids.filter((deviceId) => deviceId !== id));
    setWaterActiveDeviceIds((ids) => ids.filter((deviceId) => deviceId !== id));
    setMotionActiveDeviceIds((ids) => ids.filter((deviceId) => deviceId !== id));
    activeMotionSensorsRef.current.delete(id);
    if (motionTimersRef.current[id]) {
      clearTimeout(motionTimersRef.current[id]);
      delete motionTimersRef.current[id];
    }
    clearMotionScenarioTimers();
    addEvent(id, "Устройство убрано с плана", "INFO");
  }

  const chainGroups = useMemo(() => {
    const palette = ["#0071e3", "#30d158", "#ff9f0a", "#bf5af2", "#ff375f"];
    return selectedScenarios.map((s, i) => ({
      id: s.id,
      chain: s.chain,
      color: palette[i % palette.length],
    }));
  }, [selectedScenarios]);

  function nowTs() {
    return new Date().toLocaleTimeString("ru-RU", { hour12: false });
  }

  function addEvent(device: string, message: string, level: LogLevel = "INFO") {
    const event: LogEvent = {
      id: `${device}-${Date.now()}-${Math.random().toString(36).slice(2)}`,
      ts: nowTs(),
      level,
      device,
      message,
    };
    setEvents((prev) => [...prev, event]);
    setLastEvent(event);
  }

  useEffect(() => {
    if (floorWarningsLoggedRef.current) return;
    if (!adaptedFloor.warnings.length) return;

    floorWarningsLoggedRef.current = true;
    adaptedFloor.warnings.forEach((warning) => {
      addEvent("floor", warning, "WARNING");
    });
    // addEvent intentionally writes to the log once for the loaded floor source.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [adaptedFloor.warnings]);

  function sendWsMessage(type: string, payload?: unknown) {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return false;

    const message: WsEnvelope = {
      type,
      ts: new Date().toISOString(),
      reqId: wsReqIdRef.current,
      payload,
    };

    ws.send(JSON.stringify(message));
    return true;
  }

  function sendSimulationTick(inputs: SimEventInput[] = []) {
    if (!backendRunActiveRef.current) return false;
    wsTickRef.current += 1;
    return sendWsMessage("simulation:tick", buildTickPayload(wsTickRef.current, inputs));
  }

  function clearStartAckTimer() {
    if (!wsStartAckTimerRef.current) return;
    clearTimeout(wsStartAckTimerRef.current);
    wsStartAckTimerRef.current = null;
  }

  function startLocalSimulation(message: string, level: LogLevel = "WARNING") {
    clearStartAckTimer();
    backendRunActiveRef.current = false;
    addEvent("websocket", message, level);
    setStatus("running");
  }

  function triggerMotionSensor(sensorId: string, point: Point) {
    const wasActive = activeMotionSensorsRef.current.has(sensorId);
    activeMotionSensorsRef.current.add(sensorId);
    setMotionActiveDeviceIds((ids) => Array.from(new Set([...ids, sensorId])));

    if (!wasActive) {
      addEvent(sensorId, "Датчик движения обнаружил жителя", "INFO");
      sendSimulationTick([
        {
          kind: "human:trigger",
          entityId: "resident",
          trigger: sensorId,
          devicesPayload: [sensorId],
          payload: { turn_on: true, to: { x: point.x, y: point.y }, x: point.x, y: point.y },
        },
      ]);
      runMotionTriggeredScenarios(sensorId);
    }

    if (motionTimersRef.current[sensorId]) clearTimeout(motionTimersRef.current[sensorId]);
    motionTimersRef.current[sensorId] = setTimeout(() => {
      activeMotionSensorsRef.current.delete(sensorId);
      setMotionActiveDeviceIds((ids) => ids.filter((id) => id !== sensorId));
      sendSimulationTick([
        {
          kind: "device:trigger",
          entityId: sensorId,
          trigger: sensorId,
          devicesPayload: [sensorId],
          payload: { turn_on: false, to: { x: point.x, y: point.y }, x: point.x, y: point.y },
        },
      ]);
      delete motionTimersRef.current[sensorId];
    }, 1800);
  }

  function triggerDeviceFromPlan(deviceId: string) {
    const nextTurnOn = !manualDeviceState[deviceId];

    setManualDeviceState((state) => ({ ...state, [deviceId]: nextTurnOn }));
    setActiveNodes(nextTurnOn ? [deviceId] : []);
    addEvent(deviceId, nextTurnOn ? "Устройство включено вручную" : "Устройство выключено вручную", "INFO");
    sendSimulationTick([
      {
        kind: "human:trigger",
        entityId: "resident",
        trigger: deviceId,
        devicesPayload: [deviceId],
        payload: { turn_on: nextTurnOn },
      },
    ]);
  }

  function clearMotionScenarioTimers() {
    motionScenarioTimersRef.current.forEach((timer) => clearTimeout(timer));
    motionScenarioTimersRef.current = [];
  }

  function runMotionTriggeredScenarios(sensorId: string) {
    const scenariosToRun = selectedScenarios.filter((scenario) => scenario.chain[0] === sensorId);
    if (!scenariosToRun.length) return;

    clearMotionScenarioTimers();

    scenariosToRun.forEach((scenario) => {
      scenario.chain.forEach((deviceId, index) => {
        const timer = setTimeout(() => {
          const edge = index > 0 ? ([scenario.chain[index - 1], deviceId] as [string, string]) : null;
          setActiveNodes([deviceId]);
          setActiveEdges(edge ? [edge] : []);

          if (index > 0) {
            addEvent(deviceId, index === scenario.chain.length - 1 ? "Устройство сработало по датчику движения" : "Передан сигнал от датчика", "INFO");
          }
        }, index * 520);
        motionScenarioTimersRef.current.push(timer);
      });

      const clearEdgeTimer = setTimeout(() => {
        setActiveEdges([]);
      }, scenario.chain.length * 520 + 900);
      motionScenarioTimersRef.current.push(clearEdgeTimer);
    });
  }

  function applyBackendStep(payload: SimStepPayload) {
    const changes = payload.stateChanges ?? [];
    const activeIds = changes.map((change) => change.entityId).filter(Boolean);

    if (activeIds.length) setActiveNodes(activeIds);
    if (activeIds.length) {
      setManualDeviceState((state) => {
        const next = { ...state };
        changes.forEach((change) => {
          const rawPayload = typeof change.payload === "object" && change.payload !== null ? change.payload : {};
          if ("turn_on" in rawPayload) next[change.entityId] = Boolean((rawPayload as { turn_on?: boolean }).turn_on);
        });
        return next;
      });
    }

    changes.forEach((change) => {
      const rawPayload = typeof change.payload === "object" && change.payload !== null ? change.payload : {};
      const state = "turn_on" in rawPayload ? (rawPayload as { turn_on?: boolean }).turn_on : undefined;
      addEvent(change.entityId, state === undefined ? "Состояние обновлено бэкендом" : `Состояние: ${state ? "включено" : "выключено"}`, "INFO");
    });
  }

  function handleWsMessage(raw: string) {
    let message: WsEnvelope;
    try {
      message = JSON.parse(raw) as WsEnvelope;
    } catch {
      addEvent("websocket", "Бэк прислал некорректный JSON", "ERROR");
      return;
    }

    if (message.type === "hello:ack") {
      addEvent("websocket", "Соединение с симулятором установлено", "INFO");
      return;
    }

    if (message.type === "simulation:started") {
      clearStartAckTimer();
      backendRunActiveRef.current = true;
      setStatus("running");
      addEvent("websocket", "Бэкенд запустил симуляцию", "INFO");
      return;
    }

    if (message.type === "simulation:stopped") {
      backendRunActiveRef.current = false;
      addEvent("websocket", "Бэкенд остановил симуляцию", "INFO");
      setStatus("empty");
      return;
    }

    if (message.type === "simulation:status") {
      const payload = (message.payload ?? {}) as { state?: Status };
      if (payload.state === "empty" || payload.state === "loading" || payload.state === "running" || payload.state === "paused" || payload.state === "error") {
        setStatus(payload.state);
      }
      return;
    }

    if (message.type === "simulation:step") {
      applyBackendStep((message.payload ?? {}) as SimStepPayload);
      return;
    }

    if (message.type === "device:state") {
      const payload = (message.payload ?? {}) as { id?: string; state?: string; turn_on?: boolean };
      if (!payload.id) return;
      const turnOn = typeof payload.turn_on === "boolean" ? payload.turn_on : payload.state === "active" || payload.state === "on";
      setManualDeviceState((state) => ({ ...state, [payload.id as string]: turnOn }));
      setActiveNodes(turnOn ? [payload.id] : []);
      addEvent(payload.id, turnOn ? "Устройство включено бэкендом" : "Устройство выключено бэкендом", "INFO");
      return;
    }

    if (message.type === "log:event") {
      const payload = (message.payload ?? {}) as { level?: unknown; device?: string; message?: string };
      addEvent(payload.device ?? "backend", payload.message ?? "Событие от бэкенда", normalizeLogLevel(payload.level));
      return;
    }

    if (message.type === "error") {
      clearStartAckTimer();
      backendRunActiveRef.current = false;
      const payload = (message.payload ?? {}) as { code?: string; message?: string };
      addEvent("backend", `${payload.code ?? "ERROR"}: ${payload.message ?? "Ошибка симуляции"}`, "ERROR");
      setStatus("error");
    }
  }

  function distance(a: Point, b: Point) {
    return Math.hypot(a.x - b.x, a.y - b.y);
  }

  function hasLineOfSight(from: Point, to: Point) {
    return !blockingWalls.some((wall) => crossesWall(from, to, wall));
  }

  function markerFor(id: string) {
    return devicePositions.find((marker) => marker.id === id);
  }

  function availableMarkerFor(id: string) {
    if (!placedDeviceIds.includes(id)) return undefined;
    return markerFor(id);
  }

  function roomTitleForPoint(point: Point) {
    const room = roomsForPlan.find((r) => point.x >= r.x && point.x <= r.x + r.w && point.y >= r.y && point.y <= r.y + r.h);
    if (room) return room.title;

    if (point.x >= 0.7 && point.y >= 0.6) return "ванная";
    if (point.x >= 0.68 && point.y < 0.6) return "кухня";
    if (point.y >= 0.59 && point.x < 0.7) return "прихожая";
    if (point.x >= 0.35 && point.x < 0.7 && point.y < 0.6) return "гостиная";
    if (point.x < 0.35 && point.y < 0.43) return "спальня 2";
    if (point.x < 0.35) return "спальня";
    return "неизвестная зона";
  }

  function clampFirePoint(point: Point) {
    return {
      x: Math.min(0.955, Math.max(0.045, point.x)),
      y: Math.min(0.94, Math.max(0.06, point.y)),
    };
  }

  function buildFireSpreadWaves(origin: Point) {
    function isBlocked(from: Point, to: Point) {
      return blockingWalls.some((wall) => crossesWall(from, to, wall));
    }

    function cellKey(point: Point) {
      return `${point.x.toFixed(3)}:${point.y.toFixed(3)}`;
    }

    const cells = roomsForPlan.flatMap((room) => {
      const points: Point[] = [];
      const step = 0.065;
      for (let x = room.x + step * 0.8; x < room.x + room.w - step * 0.45; x += step) {
        for (let y = room.y + step * 0.8; y < room.y + room.h - step * 0.45; y += step) {
          const point = clampFirePoint({ x, y });
          if (distance(point, origin) > 0.018) points.push(point);
        }
      }
      return points;
    });

    const nearest = cells
      .map((cell, index) => ({ cell, index, distance: distance(origin, cell) }))
      .filter(({ cell }) => !isBlocked(origin, cell))
      .sort((a, b) => a.distance - b.distance)[0];
    if (!nearest) return [];

    const waves: Point[][] = [];
    const visited = new Set<number>();
    let frontier = [nearest];

    frontier.forEach(({ index }) => visited.add(index));

    while (frontier.length && waves.length < 22) {
      const nextCandidates = new Map<number, { cell: Point; index: number; distance: number }>();
      frontier.forEach(({ cell }) => {
        cells.forEach((candidate, index) => {
          if (visited.has(index)) return;
          const d = distance(cell, candidate);
          if (d > 0.125) return;
          if (isBlocked(cell, candidate)) return;
          const existing = nextCandidates.get(index);
          if (!existing || d < existing.distance) {
            nextCandidates.set(index, { cell: candidate, index, distance: d });
          }
        });
      });

      const next = Array.from(nextCandidates.values()).sort((a, b) => distance(origin, a.cell) - distance(origin, b.cell));
      const wave = Array.from(new Map((waves.length === 0 ? [...frontier, ...next] : next).map((item) => [cellKey(item.cell), item.cell])).values());
      if (wave.length === 0) break;
      waves.push(wave);

      frontier = next;
      frontier.forEach(({ index }) => visited.add(index));
    }

    return waves;
  }

  function buildWaterSpreadWaves(origin: Point) {
    function isBlocked(from: Point, to: Point) {
      return blockingWalls.some((wall) => crossesWall(from, to, wall));
    }

    function cellKey(point: Point) {
      return `${point.x.toFixed(3)}:${point.y.toFixed(3)}`;
    }

    const cells = roomsForPlan.flatMap((room) => {
      const points: Point[] = [];
      const step = 0.072;
      for (let x = room.x + step * 0.65; x < room.x + room.w - step * 0.4; x += step) {
        for (let y = room.y + step * 0.65; y < room.y + room.h - step * 0.4; y += step) {
          const point = clampFirePoint({ x, y });
          if (distance(point, origin) > 0.018) points.push(point);
        }
      }
      return points;
    });

    const nearest = cells
      .map((cell, index) => ({ cell, index, distance: distance(origin, cell) }))
      .filter(({ cell }) => !isBlocked(origin, cell))
      .sort((a, b) => a.distance - b.distance)[0];
    if (!nearest) return [];

    const waves: Point[][] = [];
    const visited = new Set<number>();
    let frontier = [nearest];

    frontier.forEach(({ index }) => visited.add(index));

    while (frontier.length && waves.length < 20) {
      const nextCandidates = new Map<number, { cell: Point; index: number; distance: number }>();
      frontier.forEach(({ cell }) => {
        cells.forEach((candidate, index) => {
          if (visited.has(index)) return;
          const d = distance(cell, candidate);
          if (d > 0.135) return;
          if (isBlocked(cell, candidate)) return;
          const existing = nextCandidates.get(index);
          if (!existing || d < existing.distance) {
            nextCandidates.set(index, { cell: candidate, index, distance: d });
          }
        });
      });

      const next = Array.from(nextCandidates.values()).sort((a, b) => distance(origin, a.cell) - distance(origin, b.cell));
      const wave = Array.from(new Map((waves.length === 0 ? [...frontier, ...next] : next).map((item) => [cellKey(item.cell), item.cell])).values());
      if (wave.length === 0) break;
      waves.push(wave);

      frontier = next;
      frontier.forEach(({ index }) => visited.add(index));
    }

    return waves;
  }

  function clearFireTimers() {
    fireTimersRef.current.forEach((timer) => clearTimeout(timer));
    fireTimersRef.current = [];
  }

  function scheduleFireEvent(delay: number, action: () => void) {
    const timer = setTimeout(action, delay);
    fireTimersRef.current.push(timer);
  }

  function clearWaterTimers() {
    waterTimersRef.current.forEach((timer) => clearTimeout(timer));
    waterTimersRef.current = [];
  }

  function scheduleWaterEvent(delay: number, action: () => void) {
    const timer = setTimeout(action, delay);
    waterTimersRef.current.push(timer);
  }

  function resetFire() {
    clearFireTimers();
    setFireMode(false);
    setFirePoint(null);
    setFirePoints([]);
    setFireActive(false);
    setFireActiveDeviceIds([]);
    setDisasterMessage(null);
  }

  function resetWater() {
    clearWaterTimers();
    setWaterMode(false);
    setWaterPoint(null);
    setWaterPoints([]);
    setWaterActive(false);
    setWaterActiveDeviceIds([]);
    setDisasterMessage(null);
  }

  function startFireAt(point: Point) {
    clearFireTimers();
    setFireMode(false);
    setFirePoint(point);
    setFirePoints([point]);
    setFireActive(true);
    setFireActiveDeviceIds([]);
    setDisasterMessage(null);

    const roomTitle = roomTitleForPoint(point);
    addEvent("fire", `Начало пожара: очаг в зоне "${roomTitle}"`, "WARNING");
    sendSimulationTick([{ kind: "environment:trigger", entityId: "resident", trigger: "fire", payload: { x: point.x, y: point.y, room: roomTitle } }]);

    const smokeSensor = availableMarkerFor("smoke_sensor");
    const coSensor = availableMarkerFor("co_sensor");
    const sprinklers = ["sprinkler_kitchen", "sprinkler_living"]
      .map((id) => ({ id, marker: availableMarkerFor(id) }))
      .filter((item): item is { id: string; marker: DeviceMarker } => Boolean(item.marker))
      .sort((a, b) => distance(point, a.marker) - distance(point, b.marker));

    const spreadWaves = buildFireSpreadWaves(point);
    let fireDetected = false;
    let sprinklerStarted = false;
    let fireLocalized = false;

    function triggerDetection(device: string, message: string) {
      if (fireDetected) return;
      fireDetected = true;
      setFireActiveDeviceIds((ids) => Array.from(new Set([...ids, device])));
      addEvent(device, message, "WARNING");

      scheduleFireEvent(600, () => {
        if (!placedDeviceIds.includes("siren")) return;
        setFireActiveDeviceIds((ids) => Array.from(new Set([...ids, "siren"])));
        addEvent("siren", "Сработала пожарная сирена", "WARNING");
      });

      scheduleFireEvent(1200, () => {
        if (!placedDeviceIds.includes("ventilation")) return;
        setFireActiveDeviceIds((ids) => Array.from(new Set([...ids, "ventilation"])));
        addEvent("ventilation", "Вентиляция переведена в аварийный режим удаления дыма", "INFO");
      });
    }

    function triggerSprinkler(id: string) {
      if (!fireDetected || sprinklerStarted || fireLocalized) return;
      sprinklerStarted = true;
      setFireActiveDeviceIds((ids) => Array.from(new Set([...ids, id])));
      addEvent(id, "Спринклер начал тушение очага", "INFO");

      scheduleFireEvent(2400, () => {
        fireLocalized = true;
        setFireActive(false);
        setFirePoints((points) => points.slice(0, Math.min(points.length, 28)));
        addEvent("fire", "Пожар локализован автоматической системой", "INFO");
      });
    }

    spreadWaves.forEach((wave, index) => {
      const delay = 2200 + index * 1850;
      scheduleFireEvent(delay, () => {
        if (fireLocalized) return;
        setFirePoints((points) => {
          const seen = new Set(points.map((p) => `${p.x.toFixed(3)}:${p.y.toFixed(3)}`));
          const fresh = wave.filter((p) => {
            const key = `${p.x.toFixed(3)}:${p.y.toFixed(3)}`;
            if (seen.has(key)) return false;
            seen.add(key);
            return true;
          });
          return [...points, ...fresh.slice(0, Math.max(0, 90 - points.length))];
        });

        if (smokeSensor && wave.some((p) => distance(p, smokeSensor) <= 0.16 && hasLineOfSight(p, smokeSensor))) {
          triggerDetection("smoke_sensor", "Датчик дыма обнаружил задымление");
        } else if (coSensor && wave.some((p) => distance(p, coSensor) <= 0.18 && hasLineOfSight(p, coSensor))) {
          triggerDetection("co_sensor", "CO-датчик обнаружил опасную концентрацию");
        }

        const reachedSprinkler = sprinklers.find(({ marker }) => wave.some((p) => distance(p, marker) <= 0.16 && hasLineOfSight(p, marker)));
        if (reachedSprinkler) triggerSprinkler(reachedSprinkler.id);

        if (index === 0 || index % 3 === 0) {
          const rooms = Array.from(new Set(wave.map(roomTitleForPoint))).slice(0, 3).join(", ");
          addEvent("fire", `Фронт огня расширился по площади: ${rooms}`, index >= 5 ? "ERROR" : "WARNING");
        }
      });
    });

    scheduleFireEvent(900, () => {
      if (smokeSensor && distance(point, smokeSensor) <= 0.2 && hasLineOfSight(point, smokeSensor)) {
        triggerDetection("smoke_sensor", "Датчик дыма обнаружил задымление");
      } else if (coSensor && distance(point, coSensor) <= 0.22 && hasLineOfSight(point, coSensor)) {
        triggerDetection("co_sensor", "CO-датчик обнаружил опасную концентрацию");
      } else {
        addEvent("fire", "Дым еще не дошел до датчиков", "WARNING");
      }
    });

    scheduleFireEvent(8200, () => {
      if (!fireDetected) addEvent("fire", "Пожар еще не обнаружен датчиками", "ERROR");
      if (fireDetected && !sprinklerStarted && !fireLocalized) addEvent("sprinkler", "Спринклеры пока не достигнуты фронтом огня", "WARNING");
    });

    const finalDelay = 2200 + Math.max(spreadWaves.length - 1, 0) * 1850 + 1700;
    scheduleFireEvent(finalDelay, () => {
      if (fireDetected || fireLocalized) return;
      setFireActive(false);
      setDisasterMessage("Житель сгорел заживо");
      addEvent("resident", "Житель сгорел заживо: пожар не был обнаружен системой", "ERROR");
    });
  }

  function startWaterAt(point: Point) {
    clearWaterTimers();
    setWaterMode(false);
    setWaterPoint(point);
    setWaterPoints([point]);
    setWaterActive(true);
    setWaterActiveDeviceIds([]);
    setDisasterMessage(null);

    const roomTitle = roomTitleForPoint(point);
    addEvent("flood", `Начало потопа: вода появилась в зоне "${roomTitle}"`, "WARNING");
    sendSimulationTick([{ kind: "environment:trigger", entityId: "resident", trigger: "flood", payload: { x: point.x, y: point.y, room: roomTitle } }]);

    const sensors = ["leak_sensor", "leak_sensor_bath", "water_flow"]
      .map((id) => ({ id, marker: availableMarkerFor(id) }))
      .filter((item): item is { id: string; marker: DeviceMarker } => Boolean(item.marker))
      .sort((a, b) => distance(point, a.marker) - distance(point, b.marker));

    const spreadWaves = buildWaterSpreadWaves(point);
    let waterDetected = false;
    let valveClosed = false;
    let floodLocalized = false;

    function triggerDetection(id: string) {
      if (waterDetected) return;
      waterDetected = true;
      setWaterActiveDeviceIds((ids) => Array.from(new Set([...ids, id])));
      addEvent(id, id === "water_flow" ? "Датчик расхода заметил аномальный поток воды" : "Датчик протечки обнаружил воду", "WARNING");

      scheduleWaterEvent(800, () => {
        if (!placedDeviceIds.includes("water_valve")) return;
        valveClosed = true;
        setWaterActiveDeviceIds((ids) => Array.from(new Set([...ids, "water_valve"])));
        addEvent("water_valve", "Клапан перекрыл подачу воды", "INFO");
      });

      scheduleWaterEvent(1400, () => {
        if (!placedDeviceIds.includes("siren")) return;
        setWaterActiveDeviceIds((ids) => Array.from(new Set([...ids, "siren"])));
        addEvent("siren", "Сработал аварийный сигнал потопа", "WARNING");
      });

      scheduleWaterEvent(4600, () => {
        if (!valveClosed) return;
        floodLocalized = true;
        setWaterActive(false);
        setWaterPoints((points) => points.slice(0, Math.min(points.length, 34)));
        addEvent("flood", "Потоп локализован: подача воды перекрыта", "INFO");
      });
    }

    spreadWaves.forEach((wave, index) => {
      const delay = 1300 + index * 1450;
      scheduleWaterEvent(delay, () => {
        if (floodLocalized) return;
        setWaterPoints((points) => {
          const seen = new Set(points.map((p) => `${p.x.toFixed(3)}:${p.y.toFixed(3)}`));
          const fresh = wave.filter((p) => {
            const key = `${p.x.toFixed(3)}:${p.y.toFixed(3)}`;
            if (seen.has(key)) return false;
            seen.add(key);
            return true;
          });
          return [...points, ...fresh.slice(0, Math.max(0, 80 - points.length))];
        });

        const reachedSensor = sensors.find(({ marker }) => wave.some((p) => distance(p, marker) <= 0.18 && hasLineOfSight(p, marker)));
        if (reachedSensor) triggerDetection(reachedSensor.id);

        if (index === 0 || index % 3 === 0) {
          const rooms = Array.from(new Set(wave.map(roomTitleForPoint))).slice(0, 3).join(", ");
          addEvent("flood", `Вода распространилась по площади: ${rooms}`, index >= 6 ? "ERROR" : "WARNING");
        }
      });
    });

    scheduleWaterEvent(600, () => {
      const nearestSensor = sensors.find(({ marker }) => distance(point, marker) <= 0.2 && hasLineOfSight(point, marker));
      if (nearestSensor) {
        triggerDetection(nearestSensor.id);
      } else {
        addEvent("flood", "Вода еще не дошла до датчиков протечки", "WARNING");
      }
    });

    scheduleWaterEvent(7600, () => {
      if (!waterDetected) addEvent("flood", "Потоп еще не обнаружен датчиками", "ERROR");
      if (waterDetected && !valveClosed && !floodLocalized) addEvent("water_valve", "Клапан пока не получил команду перекрытия", "ERROR");
    });

    const finalDelay = 1300 + Math.max(spreadWaves.length - 1, 0) * 1450 + 1500;
    scheduleWaterEvent(finalDelay, () => {
      if (waterDetected || floodLocalized) return;
      setWaterActive(false);
      setDisasterMessage("Житель утонул");
      addEvent("resident", "Житель утонул: потоп не был обнаружен системой", "ERROR");
    });
  }

  function onStart() {
    if (selectedScenarios.length === 0) return;

    setStatus("loading");
    setEvents([]);
    setLastEvent(null);
    setActiveNodes([]);
    setMotionActiveDeviceIds([]);
    activeMotionSensorsRef.current.clear();
    Object.values(motionTimersRef.current).forEach((timer) => clearTimeout(timer));
    motionTimersRef.current = {};
    clearMotionScenarioTimers();
    setActiveEdges([]);
    setRunScenarios(selectedScenarios);
    runStepRef.current = -1;
    runSeqIndexRef.current = 0;
    runSeqStepRef.current = -1;

    wsTickRef.current = 0;
    backendRunActiveRef.current = false;
    wsReqIdRef.current = `sim-ui-${Date.now()}`;
    clearStartAckTimer();

    const sentToBackend = sendWsMessage(
      "simulation:start",
      buildSimulationStartPayload({
        rooms: roomsForPlan,
        markers: devicePositions,
        scenarios: selectedScenarios,
        deviceIds: placedDeviceIds,
        deviceTypes: Object.fromEntries(devicesForPlan.map((device) => [device.id, device.type])),
        speed,
      })
    );

    if (!sentToBackend) {
      startLocalSimulation("Бэк симуляции недоступен, сценарий запущен локально");
      return;
    }

    addEvent("websocket", "Запрос на запуск отправлен, ждём подтверждение бэка", "INFO");
    wsStartAckTimerRef.current = setTimeout(() => {
      startLocalSimulation("Бэк не подтвердил запуск за 2 секунды, сценарий продолжен локально");
    }, 2000);
  }

  function onPause() {
    setStatus((s) => (s === "running" ? "paused" : s));
  }

  function onStop() {
    const shouldStopBackend = backendRunActiveRef.current;
    clearStartAckTimer();
    backendRunActiveRef.current = false;
    if (shouldStopBackend) sendWsMessage("simulation:stop");
    setStatus("empty");
    setEvents([]);
    setLastEvent(null);
    setActiveNodes([]);
    setActiveEdges([]);
    setManualDeviceState({});
    setMotionActiveDeviceIds([]);
    activeMotionSensorsRef.current.clear();
    Object.values(motionTimersRef.current).forEach((timer) => clearTimeout(timer));
    motionTimersRef.current = {};
    clearMotionScenarioTimers();
    setRunScenarios([]);
    runStepRef.current = -1;
    runSeqIndexRef.current = 0;
    runSeqStepRef.current = -1;
  }

  function onClear() {
    setEvents([]);
    setLastEvent(null);
    setActiveNodes([]);
    setActiveEdges([]);
    setManualDeviceState({});
    setMotionActiveDeviceIds([]);
    activeMotionSensorsRef.current.clear();
    Object.values(motionTimersRef.current).forEach((timer) => clearTimeout(timer));
    motionTimersRef.current = {};
    clearMotionScenarioTimers();
  }

  function onClearDevices() {
    removeStorage(PLAN_STORAGE_KEY);
    setPlacedDeviceIds([]);
    setActiveNodes([]);
    setActiveEdges([]);
    setManualDeviceState({});
    setFireActiveDeviceIds([]);
    setWaterActiveDeviceIds([]);
    setMotionActiveDeviceIds([]);
    activeMotionSensorsRef.current.clear();
    Object.values(motionTimersRef.current).forEach((timer) => clearTimeout(timer));
    motionTimersRef.current = {};
    clearMotionScenarioTimers();
    addEvent("plan", "Все устройства убраны с плана", "INFO");
  }

  useEffect(() => {
    if (status !== "running" || runScenarios.length === 0) {
      if (timerRef.current) clearInterval(timerRef.current);
      timerRef.current = null;
      return;
    }

    const delay = speedToDelay(speed);

    timerRef.current = setInterval(() => {
      if (runMode === "parallel") {
        const maxLen = Math.max(...runScenarios.map((s) => s.chain.length), 0);
        const next = runStepRef.current + 1;
        if (next >= maxLen) {
          setStatus("paused");
          return;
        }
        runStepRef.current = next;

        const nodes = runScenarios.map((s) => s.chain[next]).filter(Boolean);
        const edges = runScenarios
          .map((s) => (next > 0 ? [s.chain[next - 1], s.chain[next]] : null))
          .filter((e): e is [string, string] => !!e && !!e[0] && !!e[1])
          .map((e) => [e[0], e[1]] as [string, string]);

        setActiveNodes(nodes);
        setActiveEdges(edges);

        setEvents((prev: LogEvent[]) => {
          const appended = runScenarios.flatMap((s) =>
            s.chain[next]
              ? [
                  {
                    id: `${s.id}-${next}-${Date.now()}`,
                    ts: nowTs(),
                    level: "INFO" as const,
                    device: s.chain[next],
                    message: `Шаг ${next + 1}`,
                  } satisfies LogEvent,
                ]
              : []
          );
          const nextEvents = [...prev, ...appended];
          setLastEvent(appended[appended.length - 1] ?? prev[prev.length - 1] ?? null);
          return nextEvents;
        });
      } else {
        const currentScenario = runScenarios[runSeqIndexRef.current];
        if (!currentScenario) {
          setStatus("paused");
          return;
        }

        let nextStep = runSeqStepRef.current + 1;
        let nextScenarioIndex = runSeqIndexRef.current;

        if (nextStep >= currentScenario.chain.length) {
          nextScenarioIndex += 1;
          const nextScenario = runScenarios[nextScenarioIndex];
          if (!nextScenario) {
            setStatus("paused");
            return;
          }
          nextStep = 0;
        }

        const scenario = runScenarios[nextScenarioIndex];
        runSeqIndexRef.current = nextScenarioIndex;
        runSeqStepRef.current = nextStep;

        const node = scenario.chain[nextStep];
        const edge = nextStep > 0 ? [scenario.chain[nextStep - 1], scenario.chain[nextStep]] : null;

        setActiveNodes(node ? [node] : []);
        setActiveEdges(edge && edge[0] && edge[1] ? [[edge[0], edge[1]]] : []);

        if (node) {
          const ev: LogEvent = {
            id: `${scenario.id}-${nextStep}-${Date.now()}`,
            ts: nowTs(),
            level: "INFO",
            device: node,
            message: `Шаг ${nextStep + 1} • ${scenario.title}`,
          };
          setEvents((prev) => [...prev, ev]);
          setLastEvent(ev);
        }
      }

      sendSimulationTick();
    }, delay);

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      timerRef.current = null;
    };
    // sendSimulationTick reads the current WebSocket ref and tick ref, so it is safe for this interval.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, speed, runMode, runScenarios]);

  useEffect(() => {
    const url = resolveSimulationWsUrl();
    let disposed = false;

    function scheduleReconnect() {
      if (disposed || wsReconnectTimerRef.current) return;
      wsReconnectTimerRef.current = setTimeout(() => {
        wsReconnectTimerRef.current = null;
        connect();
      }, 2500);
    }

    function connect() {
      if (disposed) return;
      setWsStatus("connecting");

      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.addEventListener("open", () => {
        setWsStatus("connected");
        sendWsMessage("hello", {
          client: "sim-ui",
          version: "0.1.0",
          features: ["multiscenario", "floor-v1", "fire", "flood", "human-move", "device-trigger"],
        });
      });

      ws.addEventListener("message", (event) => {
        if (typeof event.data === "string") handleWsMessage(event.data);
      });

      ws.addEventListener("close", () => {
        if (wsRef.current === ws) {
          wsRef.current = null;
          backendRunActiveRef.current = false;
          setWsStatus("disconnected");
          scheduleReconnect();
        }
      });

      ws.addEventListener("error", () => {
        setWsStatus("error");
      });
    }

    connect();

    return () => {
      disposed = true;
      clearStartAckTimer();
      if (wsReconnectTimerRef.current) {
        clearTimeout(wsReconnectTimerRef.current);
        wsReconnectTimerRef.current = null;
      }
      const ws = wsRef.current;
      wsRef.current = null;
      ws?.close();
    };
    // handleWsMessage uses current state setters; this connection is intentionally created once per page mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    return () => {
      clearFireTimers();
      clearWaterTimers();
      clearStartAckTimer();
      if (wsReconnectTimerRef.current) clearTimeout(wsReconnectTimerRef.current);
      Object.values(motionTimersRef.current).forEach((timer) => clearTimeout(timer));
      clearMotionScenarioTimers();
    };
  }, []);

  const statusText =
    status === "loading"
      ? "Загрузка"
      : status === "running"
      ? "Выполняется"
      : status === "paused"
      ? "Пауза"
      : status === "error"
      ? "Ошибка"
      : "Готово";
  const recentEvents = events.slice(-3).reverse();
  const activeDeviceText = activeNodes.length ? activeNodes.join(", ") : "—";
  const wsStatusText =
    wsStatus === "connected"
      ? "Бэк подключен"
      : wsStatus === "connecting"
      ? "Подключение"
      : wsStatus === "error"
      ? "Ошибка подключения"
      : wsStatus === "disabled"
      ? "Локально"
      : "Переподключение";

  return (
    <main
      className="sim-page"
    >
      <div className="sim-shell">
        <Card className="sim-card">
          <ControlPanel
            scenarios={scenarios}
            selectedScenarioIds={selectedScenarioIds}
            placedDeviceIds={placedDeviceIds}
            availableDeviceIds={availableDeviceIds}
            onPlaceDevice={onPlaceDevice}
            runMode={runMode}
            onSetRunMode={setRunMode}
            status={status}
            speed={speed}
            filter={filter}
            search={search}
            onStart={onStart}
            onPause={onPause}
            onStop={onStop}
            onClear={onClear}
            onClearDevices={onClearDevices}
            onSetSpeed={setSpeed}
            onSetFilter={setFilter}
            onSetSearch={setSearch}
          />

          <div className="sim-workspace">
            <div className="sim-stage">
              <ApartmentPlan
                rooms={roomsForPlan}
                floorPlan={floorPlanForView}
                markers={devicePositions}
                devices={devicesForPlan}
                chains={chainGroups}
                activeNodes={activeNodes}
                activeEdges={activeEdges}
                lastEvent={lastEvent}
                onMoveDevice={onMoveDevice}
                onDropDevice={onDropDevice}
                onRemoveDevice={onRemoveDevice}
                fireMode={fireMode}
                firePoint={firePoint}
                firePoints={firePoints}
                fireActive={fireActive}
                fireActiveDeviceIds={fireActiveDeviceIds}
                onToggleFireMode={() => setFireMode((value) => !value)}
                onPlaceFire={startFireAt}
                onResetFire={resetFire}
                waterMode={waterMode}
                waterPoint={waterPoint}
                waterPoints={waterPoints}
                waterActive={waterActive}
                waterActiveDeviceIds={waterActiveDeviceIds}
                onToggleWaterMode={() => setWaterMode((value) => !value)}
                onPlaceWater={startWaterAt}
                onResetWater={resetWater}
                disasterMessage={disasterMessage}
                onPersonMove={(point, devicesPayload) =>
                  sendSimulationTick([
                    {
                      kind: "human:move",
                      entityId: "resident",
                      devicesPayload,
                      payload: { to: { x: point.x, y: point.y }, x: point.x, y: point.y },
                    },
                  ])
                }
                onMotionSensorTrigger={triggerMotionSensor}
                onDeviceTrigger={triggerDeviceFromPlan}
              />

              <div className="console-wrap">
                <EventConsole title="Консоль событий" events={events} filter={filter} search={search} />
              </div>
            </div>

            <aside className="right-rail">
              <section className="rail-card rail-card-hero">
                <div className="rail-eyebrow">Состояние</div>
                <div className="rail-status">{statusText}</div>
                <div className="metric-grid">
                  <div className="metric-tile">
                    <span>{selectedScenarios.length}</span>
                    <small> сценариев</small>
                  </div>
                  <div className="metric-tile">
                    <span>{devicesForPlan.length}</span>
                    <small> устройств</small>
                  </div>
                  <div className="metric-tile">
                    <span>{events.length}</span>
                    <small> событий</small>
                  </div>
                  <div className="metric-tile">
                    <span>{speed.toFixed(1)}x</span>
                    <small> скорость</small>
                  </div>
                </div>
              </section>

              <section className="rail-card">
                <div className="panel-title">Активность</div>
                <div className="activity-line">
                  <span>Активный узел</span>
                  <strong>{activeDeviceText}</strong>
                </div>
                <div className="activity-line">
                  <span>Последнее событие</span>
                  <strong>{lastEvent ? `${lastEvent.device}: ${lastEvent.message}` : "—"}</strong>
                </div>
                <div className="activity-line">
                  <span>WebSocket</span>
                  <strong>{wsStatusText}</strong>
                </div>
              </section>

              <section className="rail-card">
                <div className="panel-title">Выбранные сценарии</div>
                {selectedScenarios.length ? (
                  <div className="scenario-list">
                    {selectedScenarios.map((s, index) => (
                      <div className="scenario-row" key={s.id}>
                        <span>{index + 1}</span>
                        <strong>{s.title}</strong>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="rail-empty">Сценарии пока не выбраны</div>
                )}
              </section>

              <section className="rail-card">
                <div className="panel-title">Устройства</div>
                {devicesForPlan.length ? (
                  <div className="device-list">
                    {devicesForPlan.map((d) => (
                      <div key={d.id} className="device-row">
                        <div className="device-id">{d.id}</div>
                        <div className="device-status">{d.status}</div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="rail-empty">Появятся после выбора сценария</div>
                )}
              </section>

              <section className="rail-card">
                <div className="panel-title">Последние события</div>
                {recentEvents.length ? (
                  <div className="event-mini-list">
                    {recentEvents.map((event) => (
                      <div className="event-mini-row" key={event.id}>
                        <span>{event.ts}</span>
                        <strong>{event.device}</strong>
                        <small>{event.message}</small>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="rail-empty">Лента событий пустая</div>
                )}
              </section>
            </aside>
          </div>
        </Card>
      </div>
    </main>
  );
}
