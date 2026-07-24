"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { CircleAlert } from "lucide-react";
import { ControlPanel } from "@/app/components/sim/ControlPanel";
import { ApartmentPlan } from "@/app/components/sim/ApartmentPlan";
import type { IncidentPolygon } from "@/app/components/sim/ApartmentPlan";
import { EventConsole } from "@/app/components/sim/EventConsole";
import { ScenarioPanel } from "@/app/components/sim/ScenarioPanel";
import { Card } from "@/app/components/ui";
import floorPlanData from "@/app/simulation/floor.json";
import layoutDeviceConfig from "../../../../../services/layout/internal/configs/devices.json";
import { adaptFloorData, type FloorPlanView } from "@/app/simulation/floorAdapter";
import {
  buildIncidentActivation,
  buildDevicePercentInput,
  buildDeviceToggleInput,
  buildDeviceValueInput,
  buildHumanMoveInput,
  buildHumanRouteControlInput,
  buildHumanRouteInput,
  buildSimulationStartPayload,
  buildTickPayload,
  normalizeLogLevel,
  readDeviceActiveState,
  readDeviceLevelState,
  readDevicePercentState,
  readDeviceValueState,
  readHumanMoveState,
  readHumanRouteState,
  resolveSimulationWsUrl,
  deviceUsesPercentControl,
  deviceValueControl,
  SIMULATION_TICK_INTERVAL_MS,
  type IncidentKind,
  type IncidentStatePayload,
  type SimEventInput,
  type SimStateChange,
  type SimStepPayload,
  type WsEnvelope,
  type WsStatus,
} from "@/app/simulation/wsClient";

import {
  deviceMarkers,
  rooms as MOCK_ROOMS,
  type Scenario,
  type Device,
  type DeviceMarker,
  type LogEvent,
  type LogLevel,
  type Room,
} from "@/app/simulation/Mockdata";

interface PlacedDevice {
  id: string;
  x: number;
  y: number;
}
type Status = "empty" | "loading" | "running" | "paused" | "error";
type Speed = number;
type Filter = "ALL" | LogLevel;
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
type LayoutDeviceConfig = {
  types: Record<string, { description?: string; name?: string; tracks?: string[] }>;
  traits?: Record<string, unknown>;
};

const PLAN_STORAGE_KEY = "simulation-plan-layout";
const FLOOR_STORAGE_KEYS = ["simulation-floor", "planner-floor-json", "parsed-floor", "floor-json"];
const DEVICE_STORAGE_KEY = "simulation-devices";
const LEGACY_DEVICE_STORAGE_KEYS = ["sim-devices", "selectedDevices", "selected-devices"];
const HEARTBEAT_INTERVAL_MS = 25_000;
const CONNECTION_STALE_MS = 60_000;
const LAYOUT_DEVICES = layoutDeviceConfig as LayoutDeviceConfig;
const HUMAN_ID = "resident";

function pointInRoom(point: Point, room: Room) {
  const polygon = room.area;
  if (!polygon || polygon.length < 3) {
    return point.x >= room.x && point.x <= room.x + room.w && point.y >= room.y && point.y <= room.y + room.h;
  }

  let inside = false;
  for (let index = 0, previous = polygon.length - 1; index < polygon.length; previous = index, index += 1) {
    const currentPoint = polygon[index];
    const previousPoint = polygon[previous];
    const crossesRay =
      currentPoint.y > point.y !== previousPoint.y > point.y &&
      point.x <
        ((previousPoint.x - currentPoint.x) * (point.y - currentPoint.y)) /
          (previousPoint.y - currentPoint.y) +
          currentPoint.x;
    if (crossesRay) inside = !inside;
  }
  return inside;
}

function interiorPoint(room: Room): Point | null {
  const candidates: Point[] = [
    { x: room.labelX ?? room.x + room.w / 2, y: room.labelY ?? room.y + room.h / 2 },
    { x: room.x + room.w / 2, y: room.y + room.h / 2 },
  ];

  for (let row = 1; row < 10; row += 1) {
    for (let column = 1; column < 10; column += 1) {
      candidates.push({
        x: room.x + (room.w * column) / 10,
        y: room.y + (room.h * row) / 10,
      });
    }
  }
  return candidates.find((point) => pointInRoom(point, room)) ?? null;
}

function initialHumanPosition(rooms: Room[]): Point {
  const preferred = { x: 0.48, y: 0.72 };
  const preferredRoom = rooms.find((room) => pointInRoom(preferred, room));
  if (preferredRoom) return preferred;

  return rooms.map(interiorPoint).find((point): point is Point => point !== null) ?? preferred;
}

function newSimulationReqId() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `sim-ui-${crypto.randomUUID()}`;
  }
  return `sim-ui-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

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

function normalizePlanDependencies(value: unknown): Record<string, string[]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) return {};

  const dependencies: Record<string, string[]> = {};
  Object.entries(value).forEach(([triggerID, targets]) => {
    if (!triggerID || !Array.isArray(targets)) return;
    dependencies[triggerID] = targets.filter(
      (targetID): targetID is string => typeof targetID === "string" && targetID.length > 0
    );
  });
  return dependencies;
}

function userFacingSimulationError(code?: string, message?: string) {
  const technicalMessage = `${code ?? ""} ${message ?? ""}`.toLocaleLowerCase("ru");

  if (technicalMessage.includes("circle dependencies")) {
    return "В сценариях обнаружена замкнутая цепочка: устройства запускают друг друга по кругу. Измените связи между устройствами и попробуйте снова.";
  }
  if (code === "START_FAILED") {
    return "Не удалось запустить симуляцию. Проверьте выбранные устройства и сценарии, затем попробуйте снова.";
  }
  if (code === "TICK_FAILED") {
    return "Backend отклонил одно из действий, поэтому симуляция остановлена. Проверьте журнал событий и запустите её повторно.";
  }
  if (message?.trim()) return message.trim();
  return "Во время работы симуляции произошла ошибка. Попробуйте запустить её повторно.";
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
  return Math.round(SIMULATION_TICK_INTERVAL_MS / s);
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

  const hasDevicesInUrl = new URLSearchParams(window.location.search).has("devices");
  const fromUrl = loadExternalDevicesFromUrl();
  if (hasDevicesInUrl) {
    writeStorage(DEVICE_STORAGE_KEY, JSON.stringify(fromUrl));
    LEGACY_DEVICE_STORAGE_KEYS.forEach(removeStorage);
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

  const canonicalDevices = readStorage(DEVICE_STORAGE_KEY);
  if (canonicalDevices !== null) {
    return parseStoredDevices(canonicalDevices);
  }

  for (const key of LEGACY_DEVICE_STORAGE_KEYS) {
    const raw = readStorage(key);
    if (!raw) continue;

    const devices = parseStoredDevices(raw);
    if (!devices.length) continue;

    writeStorage(DEVICE_STORAGE_KEY, JSON.stringify(devices));
    return devices;
  }

  return [];
}

function parseStoredDevices(raw: string): ExternalDevice[] {
  try {
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

export default function SimulationPage() {
  const [floorSource] = useState<unknown>(() => loadFloorSourceFromStorage());
  const adaptedFloor = useMemo(() => adaptFloorData(floorSource, MOCK_ROOMS, deviceMarkers), [floorSource]);
  const normalizeIncidentPoint = useMemo(() => makeIncidentPointNormalizer(floorSource), [floorSource]);
  const roomsForPlan = adaptedFloor.rooms;
  const floorPlanForView: FloorPlanView = adaptedFloor.floorPlan;
  const baseDeviceMarkers = adaptedFloor.markers;
  const placementMarkers = adaptedFloor.placementMarkers;
  const [externalDevices] = useState<ExternalDevice[]>(() => loadExternalDevicesFromStorage());
  const [savedPlanDevices] = useState<SavedPlanDevice[]>(() => loadSavedPlanDevices());
  const currentDeviceIds = new Set([
    ...placementMarkers.map((marker) => marker.id),
    ...externalDevices.map((device) => device.id),
  ]);
  const currentSavedPlanDevices = currentDeviceIds.size
    ? savedPlanDevices.filter((device) => currentDeviceIds.has(device.id))
    : savedPlanDevices;

  const [status, setStatus] = useState<Status>("empty");
  const [speed, setSpeed] = useState<Speed>(1);
  const [filter, setFilter] = useState<Filter>("ALL");
  const [search, setSearch] = useState("");

  const [humanPosition, setHumanPosition] = useState<Point>(() => initialHumanPosition(roomsForPlan));

  const [events, setEvents] = useState<LogEvent[]>([]);
  const [activeNodes, setActiveNodes] = useState<string[]>([]);
  const [activeEdges, setActiveEdges] = useState<Array<[string, string]>>([]);
  const [selectedScenarioId, setSelectedScenarioId] = useState<string | null>(null);
  const [backendDeviceState, setBackendDeviceState] = useState<Record<string, boolean>>({});
  const [backendDevicePercent, setBackendDevicePercent] = useState<Record<string, number>>({});
  const [backendDeviceValue, setBackendDeviceValue] = useState<Record<string, number>>({});
  const [backendDeviceLevelArc, setBackendDeviceLevelArc] = useState<Record<string, number>>({});
  const [lastEvent, setLastEvent] = useState<LogEvent | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const activeEdgesTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const wsReqIdRef = useRef(newSimulationReqId());
  const wsTickRef = useRef(0);
  const wsLastAppliedTickRef = useRef(0);
  const backendRunActiveRef = useRef(false);
  const pendingIncidentRef = useRef<{ inputs: SimEventInput[]; onQueued: () => void } | null>(null);
  const pendingTickInputsRef = useRef<SimEventInput[]>([]);
  const pendingDeviceStateRef = useRef<Record<string, boolean>>({});
  const pendingHumanMoveRef = useRef(false);
  const [humanRouteStatus, setHumanRouteStatus] = useState("idle");
  const [incidentResetPending, setIncidentResetPending] = useState<Record<IncidentKind, boolean>>({
    "fire:spread": false,
    "flood:spread": false,
    "smoke:spread": false,
  });
  const wsStartAckTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const wsReconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastPongAtRef = useRef(0);
  const pendingStepSinceRef = useRef(0);
  const floorWarningsLoggedRef = useRef(false);
  const devicesLocked = status === "loading" || status === "running" || status === "paused";

  const [devicePositions, setDevicePositions] = useState<DeviceMarker[]>(() => {
    const savedMarkers = currentSavedPlanDevices.map((device) => ({ id: device.id, x: device.x, y: device.y }));
    const knownMarkerIds = new Set([
      ...placementMarkers.map((marker) => marker.id),
      ...currentSavedPlanDevices.map((device) => device.id),
    ]);
    const externalMarkers = externalDevices
      .filter((device) => device.x !== undefined && device.y !== undefined && !knownMarkerIds.has(device.id))
      .map((device) => ({ id: device.id, x: device.x as number, y: device.y as number, label: device.name }));

    const markersByID = new Map<string, DeviceMarker>();
    [...baseDeviceMarkers, ...FIRE_DEVICE_MARKERS, ...WATER_DEVICE_MARKERS, ...externalMarkers, ...savedMarkers].forEach(
      (marker) => markersByID.set(marker.id, marker)
    );
    return Array.from(markersByID.values());
  });
  const [placedDeviceIds, setPlacedDeviceIds] = useState<string[]>(() => {
    const placementIds = placementMarkers.map((marker) => marker.id);
    const savedIds = currentSavedPlanDevices.map((device) => device.id);
    const externalPlacedIds = externalDevices.filter((device) => device.x !== undefined && device.y !== undefined).map((device) => device.id);
    return Array.from(new Set([...placementIds, ...savedIds, ...externalPlacedIds]));
  });
  const [fireMode, setFireMode] = useState(false);
  const [firePoint, setFirePoint] = useState<Point | null>(null);
  const [fireActive, setFireActive] = useState(false);
  const [waterMode, setWaterMode] = useState(false);
  const [waterPoint, setWaterPoint] = useState<Point | null>(null);
  const [waterActive, setWaterActive] = useState(false);
  const [smokeMode, setSmokeMode] = useState(false);
  const [smokePoint, setSmokePoint] = useState<Point | null>(null);
  const [smokeActive, setSmokeActive] = useState(false);
  const [incidentPolygons, setIncidentPolygons] = useState<IncidentPolygon[]>([]);
  const [wsStatus, setWsStatus] = useState<WsStatus>("connecting");
  const [wsError, setWsError] = useState<string | null>(null);

  const [planDependencies, setPlanDependencies] = useState<Record<string, string[]>>(() => {
    if (typeof window === "undefined") return {};
    try {
      const raw = window.localStorage.getItem("simulation-plan-dependencies");
      return raw ? normalizePlanDependencies(JSON.parse(raw)) : {};
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
        const normalizedDependencies = normalizePlanDependencies(dependencies);

        window.localStorage.setItem("simulation-plan-layout", JSON.stringify(layout));
        window.localStorage.setItem("simulation-plan-dependencies", JSON.stringify(normalizedDependencies));

        setTimeout(() => {
          setPlanDependencies(normalizedDependencies);

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
  const deviceNames = useMemo(() => {
    const names = new Map(
      devicePositions.flatMap((marker) => (marker.label ? [[marker.id, marker.label] as const] : []))
    );
    externalDevices.forEach((device) => {
      if (device.name) names.set(device.id, device.name);
    });
    return Object.fromEntries(names);
  }, [devicePositions, externalDevices]);
  const selectedScenarios = useMemo<Scenario[]>(() => {
    return Object.entries(planDependencies).flatMap(([triggerID, targetIDs]) =>
      targetIDs.map((targetID, index) => ({
        id: `layout_${triggerID}_to_${targetID}_${index}`,
        title: `${deviceNames[triggerID] ?? triggerID} → ${deviceNames[targetID] ?? targetID}`,
        description: "Получено от модуля расстановки",
        chain: [triggerID, targetID],
        category: "service",
      }))
    );
  }, [deviceNames, planDependencies]);
  const highlightedEdges = useMemo<Array<[string, string]>>(() => {
    const scenario = selectedScenarios.find((item) => item.id === selectedScenarioId);
    if (!scenario) return [];

    return scenario.chain.slice(0, -1).map((deviceID, index) => [deviceID, scenario.chain[index + 1]]);
  }, [selectedScenarioId, selectedScenarios]);
  const availableDeviceIds = useMemo(() => {
    const ids = new Set<string>();

    if (externalDevices.length) {
      externalDevices.forEach((device) => ids.add(device.id));
      placedDeviceIds.forEach((id) => ids.add(id));
      return Array.from(ids);
    }

    Object.keys(LAYOUT_DEVICES.types ?? {}).forEach((id) => ids.add(id));
    selectedScenarios.forEach((scenario) => scenario.chain.forEach((id) => ids.add(id)));
    return Array.from(ids);
  }, [externalDevices, placedDeviceIds, selectedScenarios]);

  const devicesForPlan = useMemo<Device[]>(() => {
    return placedDeviceIds.map((id) => ({
      id,
      name: deviceNames[id],
      type: externalDeviceMap.get(id)?.type,
      status: backendDeviceState[id] ? "active" : "idle",
    }));
  }, [placedDeviceIds, externalDeviceMap, deviceNames, backendDeviceState]);

  const devicePercentLevels = useMemo<Record<string, number>>(() => {
    const levels: Record<string, number> = {};
    placedDeviceIds.forEach((id) => {
      if (!deviceUsesPercentControl(id, externalDeviceMap.get(id)?.type)) return;
      levels[id] = backendDevicePercent[id] ?? 0;
    });
    return levels;
  }, [placedDeviceIds, externalDeviceMap, backendDevicePercent]);

  const deviceValueControls = useMemo(() => {
    const controls: Record<string, NonNullable<ReturnType<typeof deviceValueControl>> & { value: number }> = {};
    placedDeviceIds.forEach((id) => {
      const control = deviceValueControl(id, externalDeviceMap.get(id)?.type);
      if (!control) return;
      controls[id] = {
        ...control,
        value: backendDeviceValue[id] ?? control.defaultValue,
      };
    });
    return controls;
  }, [placedDeviceIds, externalDeviceMap, backendDeviceValue]);

  const displayedDeviceLevelArcs = useMemo<Record<string, number>>(() => {
    const arcs: Record<string, number> = {};
    Object.entries(backendDeviceLevelArc).forEach(([id, level]) => {
      if (deviceUsesPercentControl(id, externalDeviceMap.get(id)?.type)) return;
      if (deviceValueControl(id, externalDeviceMap.get(id)?.type)) return;
      arcs[id] = level;
    });
    return arcs;
  }, [backendDeviceLevelArc, externalDeviceMap]);

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
    if (devicesLocked) return;
    setPlacedDeviceIds((ids) => (ids.includes(id) ? ids : [...ids, id]));
    setDevicePositions((prev) => {
      let updated = false;
      const next = prev.flatMap((marker) => {
        if (marker.id !== id) return [marker];
        if (updated) return [];
        updated = true;
        return [{ ...marker, x, y }];
      });
      return updated ? next : [...next, { id, x, y }];
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
    if (devicesLocked) return;
    const marker = markerFor(id);
    const point = marker ?? suggestedDevicePosition(id);
    onDropDevice(id, point.x, point.y);
  }

  function onRemoveDevice(id: string) {
    if (devicesLocked) return;
    setPlacedDeviceIds((ids) => ids.filter((deviceId) => deviceId !== id));
    setActiveNodes((ids) => ids.filter((deviceId) => deviceId !== id));
    setActiveEdges((edges) => edges.filter(([from, to]) => from !== id && to !== id));
    setBackendDeviceState((state) => {
      const next = { ...state };
      delete next[id];
      return next;
    });
    setBackendDevicePercent((state) => {
      const next = { ...state };
      delete next[id];
      return next;
    });
    setBackendDeviceValue((state) => {
      const next = { ...state };
      delete next[id];
      return next;
    });
    setBackendDeviceLevelArc((state) => {
      const next = { ...state };
      delete next[id];
      return next;
    });
    delete pendingDeviceStateRef.current[id];
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

  function sendSimulationTick() {
    if (!backendRunActiveRef.current || pendingStepSinceRef.current) return false;
    const nextTick = wsTickRef.current + 1;
    const sent = sendWsMessage("simulation:tick", buildTickPayload(nextTick, pendingTickInputsRef.current));
    if (sent) {
      wsTickRef.current = nextTick;
      pendingTickInputsRef.current = [];
    }
    if (sent && !pendingStepSinceRef.current) pendingStepSinceRef.current = Date.now();
    return sent;
  }

  function queueSimulationInputs(inputs: SimEventInput[]) {
    if (!backendRunActiveRef.current) return false;
    pendingTickInputsRef.current.push(...inputs);
    return true;
  }

  function clearStartAckTimer() {
    if (!wsStartAckTimerRef.current) return;
    clearTimeout(wsStartAckTimerRef.current);
    wsStartAckTimerRef.current = null;
  }

  function failSimulationStart(message: string) {
    clearStartAckTimer();
    backendRunActiveRef.current = false;
    pendingIncidentRef.current = null;
    pendingTickInputsRef.current = [];
    pendingDeviceStateRef.current = {};
    pendingHumanMoveRef.current = false;
    setIncidentResetPending({
      "fire:spread": false,
      "flood:spread": false,
      "smoke:spread": false,
    });
    addEvent("websocket", message, "ERROR");
    setWsError(message);
    setStatus("error");
  }

  function triggerDeviceFromPlan(deviceId: string) {
    const currentState = pendingDeviceStateRef.current[deviceId] ?? backendDeviceState[deviceId] ?? false;
    const nextState = !currentState;
    const input = buildDeviceToggleInput(deviceId, externalDeviceMap.get(deviceId)?.type, nextState);
    if (!input) {
      addEvent(deviceId, "Состояние устройства определяется событиями симуляции и не переключается вручную", "WARNING");
      return;
    }
    if (!queueSimulationInputs([input])) {
      addEvent(deviceId, "Симуляция не запущена, команда не отправлена", "ERROR");
      return;
    }

    pendingDeviceStateRef.current[deviceId] = nextState;
    addEvent(deviceId, nextState ? "Команда включения поставлена в очередь" : "Команда выключения поставлена в очередь", "INFO");
  }

  function setDevicePercentFromPlan(deviceId: string, value: number) {
    const input = buildDevicePercentInput(deviceId, externalDeviceMap.get(deviceId)?.type, value);
    if (!input) {
      addEvent(deviceId, "Устройство не поддерживает процентное управление", "WARNING");
      return;
    }
    if (!queueSimulationInputs([input])) {
      addEvent(deviceId, "Симуляция не запущена, команда не отправлена", "ERROR");
      return;
    }

    const percents = input.payload.percents as number;
    addEvent(deviceId, `Команда установки уровня ${percents}% поставлена в очередь`, "INFO");
  }

  function setDeviceValueFromPlan(deviceId: string, value: number) {
    const input = buildDeviceValueInput(deviceId, externalDeviceMap.get(deviceId)?.type, value);
    if (!input) {
      addEvent(deviceId, "Устройство не поддерживает числовую настройку", "WARNING");
      return;
    }
    if (!queueSimulationInputs([input])) {
      addEvent(deviceId, "Симуляция не запущена, команда не отправлена", "ERROR");
      return;
    }

    addEvent(deviceId, "Команда изменения настройки поставлена в очередь", "INFO");
  }

  function requestHumanMove(point: Point) {
    if (pendingHumanMoveRef.current) return false;
    if (!queueSimulationInputs([buildHumanMoveInput(floorSource, HUMAN_ID, point)])) {
      addEvent(HUMAN_ID, "Симуляция не запущена, перемещение не отправлено", "ERROR");
      return false;
    }
    pendingHumanMoveRef.current = true;
    return true;
  }

  function requestHumanRoute(points: Point[], speed: number) {
    if (!queueSimulationInputs([buildHumanRouteInput(floorSource, HUMAN_ID, points, speed)])) {
      addEvent(HUMAN_ID, "Симуляция не запущена, маршрут не отправлен", "ERROR");
      return false;
    }
    return true;
  }

  function requestHumanRouteControl(action: "pause" | "resume" | "stop") {
    if (!queueSimulationInputs([buildHumanRouteControlInput(HUMAN_ID, action)])) {
      addEvent(HUMAN_ID, "Симуляция не запущена, команда маршрута не отправлена", "ERROR");
      return false;
    }
    return true;
  }

  function applyBackendStep(payload: SimStepPayload) {
    const changes = payload.stateChanges ?? [];
    const triggeredEdges = (payload.triggeredEdges ?? [])
      .filter((edge) => typeof edge.from === "string" && edge.from && typeof edge.to === "string" && edge.to)
      .map((edge) => [edge.from, edge.to] as [string, string]);
    if (triggeredEdges.length) {
      if (activeEdgesTimerRef.current) clearTimeout(activeEdgesTimerRef.current);
      setActiveEdges(triggeredEdges);
      activeEdgesTimerRef.current = setTimeout(() => {
        setActiveEdges([]);
        activeEdgesTimerRef.current = null;
      }, 500);
    }
    const latestHumanMove = [...changes]
      .reverse()
      .find((change) => getStateChangeEntityId(change) === HUMAN_ID && readHumanMoveState(change.payload, floorSource));
    if (latestHumanMove) {
      pendingHumanMoveRef.current = false;
      const humanState = readHumanMoveState(latestHumanMove.payload, floorSource);
      if (humanState) setHumanPosition(humanState.position);
    }
    const latestHumanRoute = [...changes]
      .reverse()
      .map((change) => (getStateChangeEntityId(change) === HUMAN_ID ? readHumanRouteState(change.payload) : null))
      .find((state) => state !== null);
    if (latestHumanRoute) setHumanRouteStatus(latestHumanRoute.status);
    if (latestHumanMove) {
      const humanState = readHumanMoveState(latestHumanMove.payload, floorSource);
      if (humanState?.status === "route completed") setHumanRouteStatus("completed");
    }

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
        setIncidentResetPending((pending) => ({ ...pending, "fire:spread": false }));
      }
      if (incidentSnapshot.kinds.has("flood:spread")) {
        const active = incidentSnapshot.polygons.some((polygon) => polygon.kind === "flood:spread");
        setWaterActive(active);
        if (!active) setWaterPoint(null);
        setIncidentResetPending((pending) => ({ ...pending, "flood:spread": false }));
      }
      if (incidentSnapshot.kinds.has("smoke:spread")) {
        const active = incidentSnapshot.polygons.some((polygon) => polygon.kind === "smoke:spread");
        setSmokeActive(active);
        if (!active) setSmokePoint(null);
        setIncidentResetPending((pending) => ({ ...pending, "smoke:spread": false }));
      }
    }

    setActiveNodes((current) => {
      const next = new Set(current);
      changes.forEach((change) => {
        const entityId = getStateChangeEntityId(change);
        const state = readDeviceActiveState(change.payload);
        if (!entityId || state === undefined) return;
        if (state) next.add(entityId);
        else next.delete(entityId);
      });
      return Array.from(next);
    });
    setBackendDeviceState((state) => {
      const next = { ...state };
      changes.forEach((change) => {
        const entityId = getStateChangeEntityId(change);
        if (!entityId) return;
        const active = readDeviceActiveState(change.payload);
        if (active === undefined) return;
        next[entityId] = active;
        delete pendingDeviceStateRef.current[entityId];
      });
      return next;
    });
    setBackendDevicePercent((state) => {
      const next = { ...state };
      changes.forEach((change) => {
        const entityId = getStateChangeEntityId(change);
        if (!entityId) return;
        const value = readDevicePercentState(change.payload);
        if (value !== undefined) next[entityId] = value;
      });
      return next;
    });
    setBackendDeviceValue((state) => {
      const next = { ...state };
      changes.forEach((change) => {
        const entityId = getStateChangeEntityId(change);
        if (!entityId) return;
        const value = readDeviceValueState(change.payload, entityId, externalDeviceMap.get(entityId)?.type);
        if (value !== undefined) next[entityId] = value;
      });
      return next;
    });
    setBackendDeviceLevelArc((state) => {
      const next = { ...state };
      changes.forEach((change) => {
        const entityId = getStateChangeEntityId(change);
        if (!entityId) return;
        const level = readDeviceLevelState(change.payload, entityId, externalDeviceMap.get(entityId)?.type);
        if (level !== undefined) next[entityId] = level.percent;
      });
      return next;
    });

    changes.forEach((change) => {
      const entityId = getStateChangeEntityId(change);
      if (!entityId) return;
      const humanState = entityId === HUMAN_ID ? readHumanMoveState(change.payload, floorSource) : null;
      if (humanState) {
        addEvent(
          entityId,
          humanState.status === "No move"
            ? "Положение не изменилось"
            : `Перемещение подтверждено${humanState.roomID ? `, комната: ${humanState.roomID}` : ""}`,
          "INFO"
        );
        return;
      }
      const state = readDeviceActiveState(change.payload);
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
    let decoded: unknown;
    try {
      decoded = JSON.parse(raw) as unknown;
    } catch {
      addEvent("websocket", "Бэк прислал некорректный JSON", "ERROR");
      return;
    }
    if (!decoded || typeof decoded !== "object" || Array.isArray(decoded) || typeof (decoded as { type?: unknown }).type !== "string") {
      addEvent("websocket", "Бэк прислал сообщение без корректного type", "ERROR");
      return;
    }
    const message = decoded as WsEnvelope;

    const sessionMessageTypes = new Set([
      "pong",
      "simulation:started",
      "simulation:step",
      "simulation:stopped",
      "error",
    ]);
    if (sessionMessageTypes.has(message.type) && message.reqId !== wsReqIdRef.current) {
      addEvent("websocket", `Ответ для другой сессии проигнорирован: ${message.type}`, "WARNING");
      return;
    }

    if (message.type === "hello:ack") {
      if (message.reqId && message.reqId !== wsReqIdRef.current) return;
      addEvent("websocket", "Соединение с симулятором установлено", "INFO");
      return;
    }

    if (message.type === "pong") {
      lastPongAtRef.current = Date.now();
      return;
    }

    if (message.type === "simulation:started") {
      clearStartAckTimer();
      backendRunActiveRef.current = true;
      setStatus("running");
      addEvent("websocket", "Бэкенд запустил симуляцию", "INFO");
      const pendingIncident = pendingIncidentRef.current;
      pendingIncidentRef.current = null;
      if (pendingIncident && queueSimulationInputs(pendingIncident.inputs)) pendingIncident.onQueued();
      return;
    }

    if (message.type === "simulation:stopped") {
      clearStartAckTimer();
      backendRunActiveRef.current = false;
      pendingDeviceStateRef.current = {};
      pendingHumanMoveRef.current = false;
      clearStoppedSimulationState();
      addEvent("websocket", "Бэкенд остановил симуляцию", "INFO");
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
      const payload = (message.payload ?? {}) as SimStepPayload;
      if (!Number.isInteger(payload.tick) || payload.tick <= wsLastAppliedTickRef.current || payload.tick > wsTickRef.current) {
        addEvent("websocket", `Некорректный или устаревший simulation:step с tick=${String(payload.tick)}`, "WARNING");
        return;
      }
      wsLastAppliedTickRef.current = payload.tick;
      if (payload.tick === wsTickRef.current) pendingStepSinceRef.current = 0;
      applyBackendStep(payload);
      return;
    }

    if (message.type === "device:state") {
      const payload = (message.payload ?? {}) as { id?: string; state?: string; turn_on?: boolean };
      if (!payload.id) return;
      const turnOn = typeof payload.turn_on === "boolean" ? payload.turn_on : payload.state === "active" || payload.state === "on";
      setBackendDeviceState((state) => ({ ...state, [payload.id as string]: turnOn }));
      delete pendingDeviceStateRef.current[payload.id];
      setActiveNodes((current) => {
        const next = new Set(current);
        if (turnOn) next.add(payload.id as string);
        else next.delete(payload.id as string);
        return Array.from(next);
      });
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
      pendingDeviceStateRef.current = {};
      pendingHumanMoveRef.current = false;
      const payload = (message.payload ?? {}) as { code?: string; message?: string };
      setIncidentResetPending({
        "fire:spread": false,
        "flood:spread": false,
        "smoke:spread": false,
      });
      const technicalError = `${payload.code ?? "ERROR"}: ${payload.message ?? "Ошибка симуляции"}`;
      addEvent("backend", technicalError, "ERROR");
      setWsError(userFacingSimulationError(payload.code, payload.message));
      setStatus("error");
    }
  }

  function markerFor(id: string) {
    return devicePositions.find((marker) => marker.id === id);
  }

  function roomForPoint(point: Point) {
    return roomsForPlan.find((room) => pointInRoom(point, room));
  }

  function resetFire() {
    setFireMode(false);
    if (incidentResetPending["fire:spread"]) return;
    if (queueSimulationInputs([{ kind: "fire:spread", entityId: "fire", payload: { reset: true } }])) {
      setIncidentResetPending((pending) => ({ ...pending, "fire:spread": true }));
      addEvent("fire", "Запрос на сброс пожара поставлен в очередь", "INFO");
    }
  }

  function resetWater() {
    setWaterMode(false);
    if (incidentResetPending["flood:spread"]) return;
    if (queueSimulationInputs([{ kind: "flood:spread", entityId: "flood", payload: { reset: true } }])) {
      setIncidentResetPending((pending) => ({ ...pending, "flood:spread": true }));
      addEvent("flood", "Запрос на сброс потопа поставлен в очередь", "INFO");
    }
  }

  function resetSmoke() {
    setSmokeMode(false);
    if (incidentResetPending["smoke:spread"]) return;
    if (queueSimulationInputs([{ kind: "smoke:spread", entityId: "smoke", payload: { reset: true } }])) {
      setIncidentResetPending((pending) => ({ ...pending, "smoke:spread": true }));
      addEvent("smoke", "Запрос на сброс дыма поставлен в очередь", "INFO");
    }
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
    if (queueSimulationInputs(inputs)) {
      markFireStarted();
      return;
    }
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      pendingIncidentRef.current = { inputs, onQueued: markFireStarted };
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
    if (queueSimulationInputs(inputs)) {
      markFloodStarted();
      return;
    }
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      pendingIncidentRef.current = { inputs, onQueued: markFloodStarted };
      onStart();
      return;
    }
    addEvent("flood", "Нет соединения с backend, очаг не создан", "ERROR");
  }

  function startSmokeAt(point: Point) {
    setSmokeMode(false);
    const room = roomForPoint(point);
    if (!room) {
      addEvent("smoke", "Не удалось определить комнату для источника задымления", "ERROR");
      return;
    }

    const activation = buildIncidentActivation(floorSource, point, room.id);
    const inputs: SimEventInput[] = [{ kind: "smoke:spread", entityId: "smoke", payload: activation }];
    const markSmokeStarted = () => {
      setSmokePoint(point);
      setSmokeActive(true);
      addEvent("smoke", `Начало задымления в зоне "${room.title}"`, "WARNING");
    };
    if (queueSimulationInputs(inputs)) {
      markSmokeStarted();
      return;
    }
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      pendingIncidentRef.current = { inputs, onQueued: markSmokeStarted };
      onStart();
      return;
    }
    addEvent("smoke", "Нет соединения с backend, источник дыма не создан", "ERROR");
  }

  function onStart() {
    const humanRoom = roomForPoint(humanPosition) ?? roomsForPlan[0];
    if (!humanRoom) {
      failSimulationStart("На плане нет комнаты для начальной позиции жителя");
      return;
    }

    setStatus("loading");
    setEvents([]);
    setLastEvent(null);
    setActiveNodes([]);
    setBackendDeviceState({});
    setBackendDevicePercent({});
    setBackendDeviceValue({});
    setBackendDeviceLevelArc({});
    setIncidentResetPending({
      "fire:spread": false,
      "flood:spread": false,
      "smoke:spread": false,
    });
    setActiveEdges([]);
    if (activeEdgesTimerRef.current) clearTimeout(activeEdgesTimerRef.current);
    activeEdgesTimerRef.current = null;

    wsTickRef.current = 0;
    wsLastAppliedTickRef.current = 0;
    backendRunActiveRef.current = false;
    pendingTickInputsRef.current = [];
    pendingDeviceStateRef.current = {};
    pendingHumanMoveRef.current = false;
    wsReqIdRef.current = newSimulationReqId();
    clearStartAckTimer();

    const startPayload = buildSimulationStartPayload({
      floorSource,
      rooms: roomsForPlan,
      markers: devicePositions,
      scenarios: selectedScenarios,
      deviceIds: placedDeviceIds,
      deviceTypes: Object.fromEntries(devicesForPlan.map((device) => [device.id, device.type])),
      human: {
        id: HUMAN_ID,
        position: humanPosition,
        roomID: humanRoom.id,
      },
      dependencies: planDependencies,
    });
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
    if (status !== "running") return;
    setStatus("paused");
    addEvent("simulation", "Симуляция поставлена на паузу", "INFO");
  }

  function onResume() {
    if (status !== "paused") return;
    const ws = wsRef.current;
    if (!backendRunActiveRef.current || !ws || ws.readyState !== WebSocket.OPEN) {
      const message = "Не удалось продолжить симуляцию: соединение с backend потеряно.";
      addEvent("websocket", message, "ERROR");
      setWsError(message);
      return;
    }

    pendingStepSinceRef.current = 0;
    setWsError(null);
    setStatus("running");
    addEvent("simulation", "Симуляция продолжена", "INFO");
  }

  function clearStoppedSimulationState() {
    wsTickRef.current = 0;
    wsLastAppliedTickRef.current = 0;
    pendingStepSinceRef.current = 0;
    setStatus("empty");
    setEvents([]);
    setLastEvent(null);
    setActiveNodes([]);
    setActiveEdges([]);
    setBackendDeviceState({});
    setBackendDevicePercent({});
    setBackendDeviceValue({});
    setBackendDeviceLevelArc({});
    setIncidentPolygons([]);
    setFirePoint(null);
    setFireActive(false);
    setWaterPoint(null);
    setWaterActive(false);
    setSmokePoint(null);
    setSmokeActive(false);
    setSmokeMode(false);
    setHumanRouteStatus("idle");
    setIncidentResetPending({
      "fire:spread": false,
      "flood:spread": false,
      "smoke:spread": false,
    });
    if (activeEdgesTimerRef.current) clearTimeout(activeEdgesTimerRef.current);
    activeEdgesTimerRef.current = null;
  }

  function onStop() {
    if (!backendRunActiveRef.current) {
      const message = "Нельзя остановить симуляцию: нет активной backend-сессии.";
      addEvent("websocket", message, "ERROR");
      setWsError(message);
      return;
    }
    clearStartAckTimer();
    backendRunActiveRef.current = false;
    pendingIncidentRef.current = null;
    pendingTickInputsRef.current = [];
    pendingDeviceStateRef.current = {};
    pendingHumanMoveRef.current = false;
    pendingStepSinceRef.current = 0;
    if (!sendWsMessage("simulation:stop")) {
      backendRunActiveRef.current = true;
      const message = "Не удалось отправить backend команду остановки.";
      addEvent("websocket", message, "ERROR");
      setWsError(message);
      return;
    }
    setStatus("loading");
    addEvent("simulation", "Ожидаем подтверждение остановки от backend", "INFO");
    wsStartAckTimerRef.current = setTimeout(() => {
      failSimulationStart("Backend не подтвердил остановку симуляции за 2 секунды");
    }, 2000);
  }

  function onClear() {
    setEvents([]);
    setLastEvent(null);
  }

  function onClearDevices() {
    removeStorage(PLAN_STORAGE_KEY);
    setPlacedDeviceIds([]);
    setSelectedScenarioId(null);
    setActiveNodes([]);
    setActiveEdges([]);
    if (activeEdgesTimerRef.current) clearTimeout(activeEdgesTimerRef.current);
    activeEdgesTimerRef.current = null;
    setBackendDeviceState({});
    setBackendDevicePercent({});
    setBackendDeviceValue({});
    setBackendDeviceLevelArc({});
    pendingDeviceStateRef.current = {};
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
      sendSimulationTick();
    }, delay);

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      timerRef.current = null;
    };
    // sendSimulationTick reads the current WebSocket ref and tick ref, so it is safe for this interval.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, speed]);

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
          const simulationWasActive = backendRunActiveRef.current || wsStartAckTimerRef.current !== null;
          clearStartAckTimer();
          wsRef.current = null;
          backendRunActiveRef.current = false;
          pendingHumanMoveRef.current = false;
          pendingTickInputsRef.current = [];
          pendingDeviceStateRef.current = {};
          setWsStatus("disconnected");
          setWsError(
            simulationWasActive
              ? "Соединение с backend разорвано. Симуляция остановлена; после подключения запустите её заново."
              : "Соединение с backend симуляции разорвано. Выполняется переподключение."
          );
          if (simulationWasActive) setStatus("error");
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
  }, [status]);

  useEffect(() => {
    return () => {
      clearStartAckTimer();
      if (wsReconnectTimerRef.current) clearTimeout(wsReconnectTimerRef.current);
      if (activeEdgesTimerRef.current) clearTimeout(activeEdgesTimerRef.current);
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
  const activeDeviceIds = activeNodes.filter((deviceId) => placedDeviceIds.includes(deviceId));
  const activeDeviceText = activeDeviceIds.length
    ? activeDeviceIds.map((deviceId) => deviceNames[deviceId] ?? deviceId).join(", ")
    : "—";
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
              <CircleAlert size={22} strokeWidth={2.2} aria-hidden="true" />
              <span>{wsError}</span>
            </div>
          )}
          <ControlPanel
            scenarios={selectedScenarios}
            placedDeviceIds={placedDeviceIds}
            availableDeviceIds={availableDeviceIds}
            deviceNames={deviceNames}
            onPlaceDevice={onPlaceDevice}
            status={status}
            speed={speed}
            onStart={onStart}
            onPause={onPause}
            onResume={onResume}
            onStop={onStop}
            onClearDevices={onClearDevices}
            devicesLocked={devicesLocked}
            onSetSpeed={setSpeed}
          />

          <div className="sim-workspace">
            <div className="sim-stage">
              <ApartmentPlan
                rooms={roomsForPlan}
                floorPlan={floorPlanForView}
                markers={devicePositions}
                devices={devicesForPlan}
                chains={chainGroups}
                highlightedEdges={highlightedEdges}
                activeEdges={activeEdges}
                lastEvent={lastEvent}
                onMoveDevice={onMoveDevice}
                onDropDevice={onDropDevice}
                onRemoveDevice={onRemoveDevice}
                devicesLocked={devicesLocked}
                fireMode={fireMode}
                firePoint={firePoint}
                fireActive={fireActive}
                onToggleFireMode={() => {
                  setFireMode((value) => !value);
                  setWaterMode(false);
                  setSmokeMode(false);
                }}
                onPlaceFire={startFireAt}
                onResetFire={resetFire}
                fireResetPending={incidentResetPending["fire:spread"]}
                waterMode={waterMode}
                waterPoint={waterPoint}
                waterActive={waterActive}
                incidentPolygons={incidentPolygons}
                onToggleWaterMode={() => {
                  setWaterMode((value) => !value);
                  setFireMode(false);
                  setSmokeMode(false);
                }}
                onPlaceWater={startWaterAt}
                onResetWater={resetWater}
                waterResetPending={incidentResetPending["flood:spread"]}
                smokeMode={smokeMode}
                smokePoint={smokePoint}
                smokeActive={smokeActive}
                onToggleSmokeMode={() => {
                  setSmokeMode((value) => !value);
                  setFireMode(false);
                  setWaterMode(false);
                }}
                onPlaceSmoke={startSmokeAt}
                onResetSmoke={resetSmoke}
                smokeResetPending={incidentResetPending["smoke:spread"]}
                personPosition={humanPosition}
                personMovementEnabled={status === "running"}
                onPersonMove={requestHumanMove}
                personRouteStatus={humanRouteStatus}
                onPersonRoute={requestHumanRoute}
                onPersonRouteControl={requestHumanRouteControl}
                onDeviceTrigger={triggerDeviceFromPlan}
                devicePercentLevels={devicePercentLevels}
                deviceValueControls={deviceValueControls}
                deviceLevelArcs={displayedDeviceLevelArcs}
                onDevicePercentChange={setDevicePercentFromPlan}
                onDeviceValueChange={setDeviceValueFromPlan}
              />

              <ScenarioPanel
                scenarios={selectedScenarios}
                deviceNames={deviceNames}
                selectedScenarioId={selectedScenarioId}
                onSelectScenario={(scenarioId) =>
                  setSelectedScenarioId((current) => (current === scenarioId ? null : scenarioId))
                }
              />

              <div className="console-wrap">
                <EventConsole
                  title="Консоль событий"
                  events={events}
                  deviceNames={deviceNames}
                  filter={filter}
                  search={search}
                  onClear={onClear}
                  onSetFilter={setFilter}
                  onSetSearch={setSearch}
                />
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
                  <strong>
                    {lastEvent ? `${deviceNames[lastEvent.device] ?? lastEvent.device}: ${lastEvent.message}` : "—"}
                  </strong>
                </div>
                <div className="activity-line">
                  <span>WebSocket</span>
                  <strong data-testid="websocket-status">{wsStatusText}</strong>
                </div>
              </section>

              <section className="rail-card">
                <div className="panel-title">Устройства</div>
                {devicesForPlan.length ? (
                  <div className="device-list">
                    {devicesForPlan.map((d) => (
                      <div key={d.id} className="device-row">
                        <div className="device-id" title={d.name ? d.id : undefined}>
                          {d.name ?? d.id}
                        </div>
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
                        <strong title={deviceNames[event.device] ? event.device : undefined}>
                          {deviceNames[event.device] ?? event.device}
                        </strong>
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
