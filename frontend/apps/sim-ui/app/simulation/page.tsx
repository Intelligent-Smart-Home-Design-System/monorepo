"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { ControlPanel } from "@/app/components/sim/ControlPanel";
import { ApartmentPlan } from "@/app/components/sim/ApartmentPlan";
import type { IncidentPolygon } from "@/app/components/sim/ApartmentPlan";
import { EventConsole } from "@/app/components/sim/EventConsole";
import { Card } from "@/app/components/ui";
import floorPlanData from "@/app/simulation/floor.json";
import dependencyConfig from "../../../../../services/simulation/configs/dependencies.json";
import entityConfig from "../../../../../services/simulation/configs/entities.json";
import layoutDeviceConfig from "../../../../../services/layout/internal/configs/devices.json";
import { adaptFloorData, type FloorPlanView } from "@/app/simulation/floorAdapter";
import {
  buildIncidentActivation,
  buildSimulationStartPayload,
  buildTickPayload,
  normalizeLogLevel,
  resolveSimulationWsUrl,
  type IncidentKind,
  type IncidentStatePayload,
  type SimEventInput,
  type SimStateChange,
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

interface PlacedDevice {
  id: string;
  x: number;
  y: number;
}
type Status = "empty" | "loading" | "running" | "paused" | "error";
type Speed = number;
type Filter = "ALL" | LogLevel;
type RunMode = "parallel" | "sequence";
type Point = { x: number; y: number };
type RawPoint = [number, number];
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
  types: Record<string, { name?: string; tracks?: string[] }>;
  traits?: any;
};

const PLAN_STORAGE_KEY = "simulation-plan-layout";
const FLOOR_STORAGE_KEYS = ["simulation-floor", "planner-floor-json", "parsed-floor", "floor-json"];
const HEARTBEAT_INTERVAL_MS = 25_000;
const CONNECTION_STALE_MS = 60_000;
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

function getStateChangeEntityId(change: SimStateChange) {
  return change.entityId ?? change.entity_id ?? "";
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

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function isIncidentKind(value: unknown): value is IncidentKind {
  return value === "fire:spread" || value === "flood:spread" || value === "smoke:spread";
}

function collectRawFloorPoints(value: unknown): RawPoint[] {
  const points: RawPoint[] = [];

  function visit(item: unknown) {
    if (!item || typeof item !== "object") return;
    if (Array.isArray(item)) {
      if (item.length >= 2 && typeof item[0] === "number" && Number.isFinite(item[0]) && typeof item[1] === "number" && Number.isFinite(item[1])) {
        points.push([item[0], item[1]]);
        return;
      }
      item.forEach(visit);
      return;
    }

    const record = item as Record<string, unknown>;
    const x = typeof record.x === "number" ? record.x : undefined;
    const y = typeof record.y === "number" ? record.y : undefined;
    if (x !== undefined && y !== undefined && Number.isFinite(x) && Number.isFinite(y)) {
      points.push([x, y]);
    }

    Object.values(record).forEach(visit);
  }

  visit(value);
  return points;
}

function makeIncidentPointNormalizer(floorSource: unknown) {
  const sourcePoints = collectRawFloorPoints(floorSource);
  const hasRawScale = sourcePoints.some(([x, y]) => Math.abs(x) > 1 || Math.abs(y) > 1);
  if (!hasRawScale || !sourcePoints.length) return ([x, y]: RawPoint): Point => ({ x, y });

  const xs = sourcePoints.map(([x]) => x);
  const ys = sourcePoints.map(([, y]) => y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const width = Math.max(maxX - minX, 1);
  const height = Math.max(maxY - minY, 1);
  const padding = 0.04;
  const scale = 1 - padding * 2;

  return ([x, y]: RawPoint): Point => ({
    x: padding + ((x - minX) / width) * scale,
    y: padding + ((y - minY) / height) * scale,
  });
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

function loadTriggerDeviceIds(): string[] {
  if (typeof window === "undefined") return [];

  const ids = new Set<string>();
  try {
    const rawQuery = new URLSearchParams(window.location.search).get("trigger_ids");
    rawQuery
      ?.split(",")
      .map((item) => item.trim())
      .filter(Boolean)
      .forEach((id) => ids.add(id));
  } catch {
    // Query parsing is best-effort.
  }

  try {
    const raw = readStorage("simulation-trigger-device-ids");
    const parsed = raw ? (JSON.parse(raw) as unknown) : [];
    if (Array.isArray(parsed)) {
      parsed.filter((item): item is string => typeof item === "string" && item.trim().length > 0).forEach((id) => ids.add(id));
    }
  } catch {
    // Ignore stale localStorage values.
  }

  return Array.from(ids);
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

function buildPlacedScenarios(
  placedIds: string[],
  bridgeId: string | undefined,
  deviceTypes: Record<string, string | undefined>,
  preferredTriggerIds: string[]
): Scenario[] {
  const placed = Array.from(new Set(placedIds));
  const preferredTriggers = new Set(preferredTriggerIds);
  const triggers = placed.filter((id) => {
    const kind = deviceKind(id, deviceTypes[id]);
    return preferredTriggers.has(id) || isConfigTrigger(id, deviceTypes) || ["motion", "door", "leak", "smoke", "gas", "air", "lux", "button"].includes(kind);
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
  const normalizeIncidentPoint = useMemo(() => makeIncidentPointNormalizer(floorSource), [floorSource]);
  const roomsForPlan = adaptedFloor.rooms;
  const floorPlanForView: FloorPlanView = adaptedFloor.floorPlan;
  const baseDeviceMarkers = adaptedFloor.markers;
  const placementMarkers = adaptedFloor.placementMarkers;
  const [externalDevices] = useState<ExternalDevice[]>(() => loadExternalDevicesFromStorage());
  const [savedPlanDevices] = useState<SavedPlanDevice[]>(() => loadSavedPlanDevices());
  const [preferredTriggerIds] = useState<string[]>(() => loadTriggerDeviceIds());

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
  const motionTimersRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({});
  const motionScenarioTimersRef = useRef<Array<ReturnType<typeof setTimeout>>>([]);
  const activeMotionSensorsRef = useRef<Set<string>>(new Set());
  const wsRef = useRef<WebSocket | null>(null);
  const wsReqIdRef = useRef("sim-ui");
  const wsTickRef = useRef(0);
  const backendRunActiveRef = useRef(false);
  const shouldResumeBackendRef = useRef(false);
  const lastStartPayloadRef = useRef<ReturnType<typeof buildSimulationStartPayload> | null>(null);
  const pendingIncidentRef = useRef<{ inputs: SimEventInput[]; onSent: () => void } | null>(null);
  const wsStartAckTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wsReconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastPongAtRef = useRef(0);
  const pendingStepSinceRef = useRef(0);
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
  const [fireActive, setFireActive] = useState(false);
  const [waterMode, setWaterMode] = useState(false);
  const [waterPoint, setWaterPoint] = useState<Point | null>(null);
  const [waterActive, setWaterActive] = useState(false);
  const [incidentPolygons, setIncidentPolygons] = useState<IncidentPolygon[]>([]);
  const [motionActiveDeviceIds, setMotionActiveDeviceIds] = useState<string[]>([]);
  const [wsStatus, setWsStatus] = useState<WsStatus>("connecting");
  const [wsError, setWsError] = useState<string | null>(null);

  const [planDependencies, setPlanDependencies] = useState<Record<string, string[]>>(() => {
    if (typeof window === "undefined") return {};
    try {
      const raw = window.localStorage.getItem("simulation-plan-dependencies");
      return raw ? JSON.parse(raw) : {};
    } catch {
      return {};
    }
  });

  useEffect(() => {
    if (typeof window === "undefined") return;

    try {
      const urlParams = new URLSearchParams(window.location.search);
      const configRaw = urlParams.get("config");

      if (configRaw) {
        const { layout, dependencies } = JSON.parse(decodeURIComponent(configRaw));

        window.localStorage.setItem("simulation-plan-layout", JSON.stringify(layout));
        window.localStorage.setItem("simulation-plan-dependencies", JSON.stringify(dependencies));

        setTimeout(() => {
          setPlanDependencies(dependencies);

          const list: PlacedDevice[] = layout?.devices || [];
          if (list.length) {
            setPlacedDeviceIds(list.map((d) => d.id));
            setDevicePositions(list.map((d) => ({ id: d.id, x: d.x, y: d.y })));
          }
        }, 0);

        urlParams.delete("config");
        const nextQuery = urlParams.toString();
        window.history.replaceState(
          null,
          "",
          nextQuery ? `${window.location.pathname}?${nextQuery}` : window.location.pathname
        );
      }
    } catch (err) {
      console.error("Не удалось распарсить конфигурацию симуляции из URL:", err);
    }
  }, []);

  const externalDeviceMap = useMemo(() => new Map(externalDevices.map((device) => [device.id, device])), [externalDevices]);
  const deviceTypeMap = useMemo<Record<string, string | undefined>>(() => {
    return Object.fromEntries(externalDevices.map((device) => [device.id, device.type]));
  }, [externalDevices]);
  const bridgeId = useMemo(() => placedDeviceIds.find((id) => deviceKind(id, externalDeviceMap.get(id)?.type) === "bridge"), [placedDeviceIds, externalDeviceMap]);
  const placedScenarios = useMemo(
    () => buildPlacedScenarios(placedDeviceIds, bridgeId, deviceTypeMap, preferredTriggerIds),
    [placedDeviceIds, bridgeId, deviceTypeMap, preferredTriggerIds]
  );
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

    Object.keys(LAYOUT_DEVICES.types ?? {}).forEach((id) => ids.add(id));
    baseScenarios.forEach((scenario) => scenario.chain.forEach((id) => ids.add(id)));
    scenarios.forEach((scenario) => scenario.chain.forEach((id) => ids.add(id)));
    return Array.from(ids);
  }, [baseScenarios, externalDevices, placedDeviceIds, scenarios]);

  const devicesForPlan = useMemo<Device[]>(() => {
    const activeSelectedIds = motionActiveDeviceIds.filter((id) => placedDeviceIds.includes(id));
    const ids = Array.from(new Set([...placedDeviceIds, ...activeSelectedIds]));

    return ids.map((id) => ({
      id,
      name: externalDeviceMap.get(id)?.name,
      type: externalDeviceMap.get(id)?.type,
      status:
        activeNodes.includes(id) ||
        manualDeviceState[id] ||
        motionActiveDeviceIds.includes(id)
          ? "active"
          : "idle",
    }));
  }, [placedDeviceIds, activeNodes, manualDeviceState, motionActiveDeviceIds, externalDeviceMap]);

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
    const sent = sendWsMessage("simulation:tick", buildTickPayload(wsTickRef.current, inputs));
    if (sent && !pendingStepSinceRef.current) pendingStepSinceRef.current = Date.now();
    return sent;
  }

  function clearStartAckTimer() {
    if (!wsStartAckTimerRef.current) return;
    clearTimeout(wsStartAckTimerRef.current);
    wsStartAckTimerRef.current = null;
  }

  function failSimulationStart(message: string) {
    clearStartAckTimer();
    backendRunActiveRef.current = false;
    shouldResumeBackendRef.current = false;
    pendingIncidentRef.current = null;
    addEvent("websocket", message, "ERROR");
    setWsError(message);
    setStatus("error");
  }

  function triggerMotionSensor(sensorId: string, point: Point) {
    const wasActive = activeMotionSensorsRef.current.has(sensorId);
    activeMotionSensorsRef.current.add(sensorId);
    setMotionActiveDeviceIds((ids) => Array.from(new Set([...ids, sensorId])));

    if (!wasActive) {
      addEvent(sensorId, "Датчик движения обнаружил жителя", "INFO");

      const connectedExecutors = planDependencies[sensorId] || [];
      const affectedDevices = [sensorId, ...connectedExecutors];

      sendSimulationTick([
        {
          kind: "human:trigger",
          entityId: "resident",
          trigger: sensorId,
          devicesPayload: affectedDevices,
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
    const incidentSnapshot = incidentPolygonsFromChanges(changes);
    if (incidentSnapshot.kinds.size) {
      setIncidentPolygons((current) => [
        ...current.filter((polygon) => !incidentSnapshot.kinds.has(polygon.kind)),
        ...incidentSnapshot.polygons,
      ]);
      if (incidentSnapshot.kinds.has("fire:spread")) {
        const active = incidentSnapshot.polygons.some((polygon) => polygon.kind === "fire:spread");
        setFireActive(active);
        if (!active) setFirePoint(null);
      }
      if (incidentSnapshot.kinds.has("flood:spread")) {
        const active = incidentSnapshot.polygons.some((polygon) => polygon.kind === "flood:spread");
        setWaterActive(active);
        if (!active) setWaterPoint(null);
      }
    }

    setActiveNodes((current) => {
      const next = new Set(current);
      changes.forEach((change) => {
        const entityId = getStateChangeEntityId(change);
        const rawPayload = typeof change.payload === "object" && change.payload !== null ? change.payload : {};
        if (!entityId || !("turn_on" in rawPayload)) return;
        if (Boolean((rawPayload as { turn_on?: boolean }).turn_on)) next.add(entityId);
        else next.delete(entityId);
      });
      return Array.from(next);
    });
    setManualDeviceState((state) => {
      const next = { ...state };
      changes.forEach((change) => {
        const entityId = getStateChangeEntityId(change);
        if (!entityId) return;
        const rawPayload = typeof change.payload === "object" && change.payload !== null ? change.payload : {};
        if ("turn_on" in rawPayload) next[entityId] = Boolean((rawPayload as { turn_on?: boolean }).turn_on);
      });
      return next;
    });

    changes.forEach((change) => {
      const entityId = getStateChangeEntityId(change);
      if (!entityId) return;
      const rawPayload = typeof change.payload === "object" && change.payload !== null ? change.payload : {};
      const state = "turn_on" in rawPayload ? (rawPayload as { turn_on?: boolean }).turn_on : undefined;
      addEvent(entityId, state === undefined ? "Состояние обновлено бэкендом" : `Состояние: ${state ? "включено" : "выключено"}`, "INFO");
    });
  }

  function incidentPolygonsFromChanges(changes: SimStateChange[]) {
    const kinds = new Set<IncidentKind>();
    const polygonsByKind = new Map<IncidentKind, IncidentPolygon[]>();
    changes.forEach((change) => {
      const payload = change.payload as IncidentStatePayload | undefined;
      if (!payload || typeof payload !== "object" || !isIncidentKind(payload.kind)) return;
      const kind = payload.kind;
      kinds.add(kind);

      const polygons = (payload.incidents ?? []).flatMap((zone) =>
        (zone.blocks ?? []).flatMap((block) => {
          if (!Array.isArray(block.points) || block.points.length < 3) return [];
          const points = block.points
            .filter((point): point is RawPoint => Array.isArray(point) && point.length >= 2 && isFiniteNumber(point[0]) && isFiniteNumber(point[1]))
            .map(normalizeIncidentPoint)
            .filter((point) => point.x >= 0 && point.x <= 1 && point.y >= 0 && point.y <= 1);
          if (points.length < 3) return [];

          return [
            {
              id: `${kind}:${block.id}`,
              kind,
              points,
            },
          ];
        })
      );
      polygonsByKind.set(kind, polygons);
    });
    return { kinds, polygons: Array.from(polygonsByKind.values()).flat() };
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
      if (shouldResumeBackendRef.current && lastStartPayloadRef.current) {
        sendWsMessage("simulation:start", lastStartPayloadRef.current);
        addEvent("websocket", "Восстанавливаем симуляцию после переподключения", "INFO");
      }
      return;
    }

    if (message.type === "pong") {
      lastPongAtRef.current = Date.now();
      return;
    }

    if (message.type === "simulation:started") {
      clearStartAckTimer();
      backendRunActiveRef.current = true;
      shouldResumeBackendRef.current = true;
      setStatus("running");
      addEvent("websocket", "Бэкенд запустил симуляцию", "INFO");
      const pendingIncident = pendingIncidentRef.current;
      pendingIncidentRef.current = null;
      if (pendingIncident && sendSimulationTick(pendingIncident.inputs)) pendingIncident.onSent();
      return;
    }

    if (message.type === "simulation:stopped") {
      backendRunActiveRef.current = false;
      shouldResumeBackendRef.current = false;
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
      pendingStepSinceRef.current = 0;
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
      const errorMessage = `${payload.code ?? "ERROR"}: ${payload.message ?? "Ошибка симуляции"}`;
      addEvent("backend", errorMessage, "ERROR");
      setWsError(errorMessage);
      setStatus("error");
    }
  }

  function markerFor(id: string) {
    return devicePositions.find((marker) => marker.id === id);
  }

  function roomForPoint(point: Point) {
    return roomsForPlan.find((room) => point.x >= room.x && point.x <= room.x + room.w && point.y >= room.y && point.y <= room.y + room.h);
  }

  function resetFire() {
    setFireMode(false);
    setFirePoint(null);
    setFireActive(false);
    setIncidentPolygons((polygons) => polygons.filter((polygon) => polygon.kind !== "fire:spread"));
    sendSimulationTick([{ kind: "fire:spread", entityId: "fire", payload: { reset: true } }]);
  }

  function resetWater() {
    setWaterMode(false);
    setWaterPoint(null);
    setWaterActive(false);
    setIncidentPolygons((polygons) => polygons.filter((polygon) => polygon.kind !== "flood:spread"));
    sendSimulationTick([{ kind: "flood:spread", entityId: "flood", payload: { reset: true } }]);
  }

  function startFireAt(point: Point) {
    setFireMode(false);
    const room = roomForPoint(point);
    if (!room) {
      addEvent("fire", "Не удалось определить комнату для очага пожара", "ERROR");
      return;
    }

    const activation = buildIncidentActivation(floorSource, point, room.id);
    const inputs: SimEventInput[] = [{ kind: "fire:spread", entityId: "fire", payload: activation }];
    const markFireStarted = () => {
      setFirePoint(point);
      setFireActive(true);
      addEvent("fire", `Начало пожара: очаг в зоне "${room.title}"`, "WARNING");
    };
    if (sendSimulationTick(inputs)) {
      markFireStarted();
      return;
    }
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      pendingIncidentRef.current = { inputs, onSent: markFireStarted };
      onStart();
      return;
    }
    addEvent("fire", "Нет соединения с backend, очаг не создан", "ERROR");
  }

  function startWaterAt(point: Point) {
    setWaterMode(false);
    const room = roomForPoint(point);
    if (!room) {
      addEvent("flood", "Не удалось определить комнату для очага затопления", "ERROR");
      return;
    }

    const activation = buildIncidentActivation(floorSource, point, room.id);
    const inputs: SimEventInput[] = [{ kind: "flood:spread", entityId: "flood", payload: activation }];
    const markFloodStarted = () => {
      setWaterPoint(point);
      setWaterActive(true);
      addEvent("flood", `Начало потопа: вода появилась в зоне "${room.title}"`, "WARNING");
    };
    if (sendSimulationTick(inputs)) {
      markFloodStarted();
      return;
    }
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      pendingIncidentRef.current = { inputs, onSent: markFloodStarted };
      onStart();
      return;
    }
    addEvent("flood", "Нет соединения с backend, очаг не создан", "ERROR");
  }

  function onStart() {
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

    const startPayload = buildSimulationStartPayload({
        floorSource,
        rooms: roomsForPlan,
        markers: devicePositions,
        scenarios: selectedScenarios,
        deviceIds: placedDeviceIds,
        deviceTypes: Object.fromEntries(devicesForPlan.map((device) => [device.id, device.type])),
        speed,
        dependencies: planDependencies
      });
    lastStartPayloadRef.current = startPayload;
    const sentToBackend = sendWsMessage("simulation:start", startPayload);

    if (!sentToBackend) {
      failSimulationStart("Backend симуляции недоступен, запуск отменён");
      return;
    }

    addEvent("websocket", "Запрос на запуск отправлен, ждём подтверждение бэка", "INFO");
    wsStartAckTimerRef.current = setTimeout(() => {
      failSimulationStart("Backend не подтвердил запуск симуляции за 2 секунды");
    }, 2000);
  }

  function onPause() {
    setStatus((s) => (s === "running" ? "paused" : s));
  }

  function onStop() {
    const shouldStopBackend = backendRunActiveRef.current;
    clearStartAckTimer();
    backendRunActiveRef.current = false;
    shouldResumeBackendRef.current = false;
    pendingIncidentRef.current = null;
    if (shouldStopBackend) sendWsMessage("simulation:stop");
    setStatus("empty");
    setEvents([]);
    setLastEvent(null);
    setActiveNodes([]);
    setActiveEdges([]);
    setManualDeviceState({});
    setIncidentPolygons([]);
    setFirePoint(null);
    setFireActive(false);
    setWaterPoint(null);
    setWaterActive(false);
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
    setMotionActiveDeviceIds([]);
    activeMotionSensorsRef.current.clear();
    Object.values(motionTimersRef.current).forEach((timer) => clearTimeout(timer));
    motionTimersRef.current = {};
    clearMotionScenarioTimers();
    addEvent("plan", "Все устройства убраны с плана", "INFO");
  }

  useEffect(() => {
    if (status !== "running") {
      if (timerRef.current) clearInterval(timerRef.current);
      timerRef.current = null;
      return;
    }

    const delay = speedToDelay(speed);

    timerRef.current = setInterval(() => {
      if (runScenarios.length === 0) {
        sendSimulationTick();
        return;
      }

      if (runMode === "parallel") {
        const maxLen = Math.max(...runScenarios.map((s) => s.chain.length), 0);
        const next = runStepRef.current + 1;
        if (next >= maxLen) {
          sendSimulationTick();
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
          sendSimulationTick();
          return;
        }

        let nextStep = runSeqStepRef.current + 1;
        let nextScenarioIndex = runSeqIndexRef.current;

        if (nextStep >= currentScenario.chain.length) {
          nextScenarioIndex += 1;
          const nextScenario = runScenarios[nextScenarioIndex];
          if (!nextScenario) {
            sendSimulationTick();
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
    if (!url) {
      const disabledTimer = window.setTimeout(() => {
        if (disposed) return;
        setWsStatus("disabled");
        setWsError("Нет токена авторизации. Войдите в аккаунт и откройте симуляцию повторно.");
      }, 0);
      return () => {
        disposed = true;
        window.clearTimeout(disabledTimer);
      };
    }
    const wsUrl = url;

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

      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.addEventListener("open", () => {
        lastPongAtRef.current = Date.now();
        pendingStepSinceRef.current = 0;
        setWsStatus("connected");
        setWsError(null);
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
          setWsError("Соединение с backend симуляции разорвано. Выполняется переподключение.");
          scheduleReconnect();
        }
      });

      ws.addEventListener("error", () => {
        setWsStatus("error");
        setWsError("Не удалось подключиться к backend симуляции через API Gateway.");
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
    const timer = window.setInterval(() => {
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) return;

      const now = Date.now();
      if (status === "running" && backendRunActiveRef.current) {
        if (pendingStepSinceRef.current && now - pendingStepSinceRef.current > CONNECTION_STALE_MS) {
          ws.close();
        }
        return;
      }

      if (lastPongAtRef.current && now - lastPongAtRef.current > CONNECTION_STALE_MS) {
        ws.close();
        return;
      }
      sendWsMessage("ping", { sentAt: new Date(now).toISOString() });
    }, HEARTBEAT_INTERVAL_MS);

    return () => window.clearInterval(timer);
    // sendWsMessage uses the current socket ref; status selects tick/step or ping/pong health checks.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status]);

  useEffect(() => {
    return () => {
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
          {wsError && (
            <div className="simulation-error-banner" role="alert" data-testid="simulation-error">
              {wsError}
            </div>
          )}
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
                fireActive={fireActive}
                onToggleFireMode={() => setFireMode((value) => !value)}
                onPlaceFire={startFireAt}
                onResetFire={resetFire}
                waterMode={waterMode}
                waterPoint={waterPoint}
                waterActive={waterActive}
                incidentPolygons={incidentPolygons}
                onToggleWaterMode={() => setWaterMode((value) => !value)}
                onPlaceWater={startWaterAt}
                onResetWater={resetWater}
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
                  <strong data-testid="websocket-status">{wsStatusText}</strong>
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
