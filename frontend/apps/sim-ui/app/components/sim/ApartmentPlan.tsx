"use client";

import { useEffect, useRef, useState, type CSSProperties, type MouseEvent } from "react";
import type { Device, DeviceMarker, LogEvent, Room } from "@/app/simulation/Mockdata";
import type { FloorPlanView, FloorZoneView, WallSegment } from "@/app/simulation/floorAdapter";

type Point = { x: number; y: number };

type Props = {
  rooms: Room[];
  floorPlan?: FloorPlanView;
  markers: DeviceMarker[];
  devices: Device[];
  chains: { id: string; chain: string[]; color: string }[];
  activeNodes: string[];
  activeEdges: Array<[string, string]>;
  lastEvent: LogEvent | null;
  onMoveDevice?: (id: string, x: number, y: number) => void;
  onDropDevice?: (id: string, x: number, y: number) => void;
  onRemoveDevice?: (id: string) => void;
  fireMode?: boolean;
  firePoint?: Point | null;
  firePoints?: Point[];
  fireActive?: boolean;
  fireActiveDeviceIds?: string[];
  onToggleFireMode?: () => void;
  onPlaceFire?: (point: Point) => void;
  onResetFire?: () => void;
  waterMode?: boolean;
  waterPoint?: Point | null;
  waterPoints?: Point[];
  waterActive?: boolean;
  waterActiveDeviceIds?: string[];
  onToggleWaterMode?: () => void;
  onPlaceWater?: (point: Point) => void;
  onResetWater?: () => void;
  disasterMessage?: string | null;
  onPersonMove?: (point: Point, devicesPayload: string[]) => void;
  onMotionSensorTrigger?: (sensorId: string, point: Point) => void;
  onDeviceTrigger?: (deviceId: string) => void;
};

type DeviceType =
  | "pir"
  | "mmwave"
  | "door"
  | "leak"
  | "smoke"
  | "co"
  | "gas"
  | "temp"
  | "humidity"
  | "lux"
  | "noise"
  | "co2"
  | "voc"
  | "pm25"
  | "pressure"
  | "floor_temp"
  | "freeze"
  | "current"
  | "water_flow"
  | "camera"
  | "light"
  | "siren"
  | "lock"
  | "climate_device"
  | "media"
  | "other";

type ForbiddenZone = {
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;
  forbiddenFor: DeviceType[];
  reason: string;
};

const PERSON_STEP_MS = 340;
const ROUTE_STEP_GAP_MS = 35;
const PERSON_ROUTE_STEP = 0.035;
const MOTION_SENSOR_RADIUS = 0.13;
const ALL_DEVICE_TYPES: DeviceType[] = [
  "pir",
  "mmwave",
  "door",
  "leak",
  "smoke",
  "co",
  "gas",
  "temp",
  "humidity",
  "lux",
  "noise",
  "co2",
  "voc",
  "pm25",
  "pressure",
  "floor_temp",
  "freeze",
  "current",
  "water_flow",
  "camera",
  "light",
  "siren",
  "lock",
  "climate_device",
  "media",
  "other",
];

export function ApartmentPlan({
  rooms,
  floorPlan,
  markers,
  devices,
  chains,
  activeNodes,
  activeEdges,
  lastEvent,
  onMoveDevice,
  onDropDevice,
  onRemoveDevice,
  fireMode = false,
  firePoint = null,
  firePoints = [],
  fireActive = false,
  fireActiveDeviceIds = [],
  onToggleFireMode,
  onPlaceFire,
  onResetFire,
  waterMode = false,
  waterPoint = null,
  waterPoints = [],
  waterActive = false,
  waterActiveDeviceIds = [],
  onToggleWaterMode,
  onPlaceWater,
  onResetWater,
  disasterMessage = null,
  onPersonMove,
  onMotionSensorTrigger,
  onDeviceTrigger,
}: Props) {
  const deviceMap = new Map(devices.map((d) => [d.id, d.status]));
  const lastDevice = lastEvent?.device ?? null;
  const roomMap = new Map(rooms.map((r) => [r.id, r]));
  const markerMap = new Map(markers.map((m) => [m.id, m]));
  const chainSet = new Set(chains.flatMap((c) => c.chain));
  const activeSet = new Set(activeNodes);
  const floorViewBox = floorPlan?.walls?.viewBox ?? floorPlan?.doors?.viewBox ?? { width: 1000, height: 700 };
  const surfaceRef = useRef<HTMLDivElement | null>(null);
  const personMarkerRef = useRef<HTMLDivElement | null>(null);
  const dragRef = useRef<{ id: string } | null>(null);
  const personDragRef = useRef(false);
  const lastValidRef = useRef<Record<string, { x: number; y: number }>>({});
  const walkTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const routeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const routePathRef = useRef<Point[]>([]);
  const routeIndexRef = useRef(0);
  const pausedRouteRef = useRef<Point[]>([]);
  const triggeredMotionSensorsRef = useRef<Set<string>>(new Set());
  const [draggingType, setDraggingType] = useState<DeviceType | null>(null);
  const [personPos, setPersonPos] = useState({ x: 0.48, y: 0.72 });
  const [personWalking, setPersonWalking] = useState(false);
  const [personDragging, setPersonDragging] = useState(false);
  const [walkCycle, setWalkCycle] = useState(0);
  const [routeMode, setRouteMode] = useState(false);
  const [routeWalking, setRouteWalking] = useState(false);
  const [routePaused, setRoutePaused] = useState(false);
  const [routeError, setRouteError] = useState<string | null>(null);
  const [routePoints, setRoutePoints] = useState<Point[]>([]);
  const [routeSpeed, setRouteSpeed] = useState(1);
  const layoutMap = (() => {
    const byRoom = new Map<string, string[]>();
    devices.forEach((d) => {
      const roomId = roomForDevice(d.id);
      const list = byRoom.get(roomId) ?? [];
      list.push(d.id);
      byRoom.set(roomId, list);
    });

    for (const [roomId, list] of byRoom) {
      list.sort((a, b) => a.localeCompare(b));
      byRoom.set(roomId, list);
    }

    const pos = new Map<string, { id: string; x: number; y: number }>();

    for (const [roomId, list] of byRoom) {
      const room = roomMap.get(roomId) ?? rooms[0];
      if (!room) continue;
      const cols = room.w >= 0.28 ? 3 : 2;
      const rows = Math.max(1, Math.ceil(list.length / cols));
      const paddingX = 0.12;
      const paddingY = 0.18;

      list.forEach((id, i) => {
        const col = i % cols;
        const row = Math.floor(i / cols);
        const fx = (col + 0.5) / cols;
        const fy = rows === 1 ? 0.5 : (row + 0.5) / rows;
        const x = room.x + room.w * (paddingX + (1 - paddingX * 2) * fx);
        const y = room.y + room.h * (paddingY + (1 - paddingY * 2) * fy);
        pos.set(id, { id, x, y });
      });
    }

    return pos;
  })();

  function hashId(id: string) {
    let h = 0;
    for (let i = 0; i < id.length; i += 1) h = (h * 31 + id.charCodeAt(i)) % 9973;
    return h;
  }

  function roomForDevice(id: string) {
    const key = id.toLowerCase();
    if (key.includes("kitchen")) return "kitchen";
    if (key.includes("hall")) return "hall";
    if (key.includes("bed")) return "bedroom_1";
    if (key.includes("living")) return "living";
    if (key.includes("bath")) return "bath";
    if (key.includes("toilet")) return "toilet";
    if (key.includes("smoke")) return "kitchen";
    if (key.includes("temp")) return "living";
    if (key.includes("door")) return "hall";
    if (key.includes("motion")) return "hall";
    if (key.includes("hub")) return "hall";
    if (key.includes("gateway")) return "hall";
    if (key.includes("controller")) return "living";
    if (key.includes("siren")) return "hall";
    if (key.includes("lamp_kitchen")) return "kitchen";
    if (key.includes("lamp_hall")) return "hall";
    if (key.includes("lamp")) return "living";
    if (key.includes("heater")) return "living";
    return "living";
  }

  function deviceTypeForId(id: string): DeviceType {
    const key = id.toLowerCase();
    if (key.includes("motion")) return "pir";
    if (key.includes("mmwave")) return "mmwave";
    if (key.includes("door")) return "door";
    if (key.includes("leak")) return "leak";
    if (key.includes("smoke")) return "smoke";
    if (key.includes("co2")) return "co2";
    if (key.includes("co")) return "co";
    if (key.includes("gas")) return "gas";
    if (key.includes("temp")) return "temp";
    if (key.includes("humidity")) return "humidity";
    if (key.includes("lux")) return "lux";
    if (key.includes("noise") || key.includes("sound")) return "noise";
    if (key.includes("voc")) return "voc";
    if (key.includes("pm25")) return "pm25";
    if (key.includes("pressure")) return "pressure";
    if (key.includes("floor")) return "floor_temp";
    if (key.includes("freeze")) return "freeze";
    if (key.includes("current") || key.includes("power")) return "current";
    if (key.includes("water") || key.includes("flow")) return "water_flow";
    if (key.includes("lamp") || key.includes("bulb") || key.includes("backlight") || key.includes("luminaire") || key.includes("dimmer")) return "light";
    if (key.includes("siren")) return "siren";
    if (key.includes("lock")) return "lock";
    if (key.includes("conditioner") || key.includes("radiator") || key.includes("thermostat") || key.includes("purifier") || key.includes("humidifier")) {
      return "climate_device";
    }
    if (key.includes("tv") || key.includes("speaker") || key.includes("subwoofer")) return "media";
    if (key.includes("camera")) return "camera";
    return "other";
  }

  function isMotionSensor(device: Pick<Device, "id" | "name" | "type">) {
    const key = `${device.id} ${device.name ?? ""} ${device.type ?? ""}`.toLowerCase();
    return (
      key.includes("motion") ||
      key.includes("presence") ||
      key.includes("mmwave") ||
      key.includes("pir") ||
      key.includes("датчик движения") ||
      key.includes("движени") ||
      key.includes("присутств")
    );
  }

  function isLightDevice(id: string) {
    const key = id.toLowerCase();
    return key.includes("lamp") || key.includes("light") || key.includes("led");
  }

  function positionForDevice(id: string) {
    const manual = markerMap.get(id);
    if (manual) return manual;
    const placed = layoutMap.get(id);
    if (placed) return placed;
    const roomId = roomForDevice(id);
    const room = roomMap.get(roomId) ?? rooms[0];
    const seed = hashId(id);
    const fx = ((seed % 97) / 97) * 0.6 + 0.2;
    const fy = (((seed * 7) % 97) / 97) * 0.6 + 0.2;
    return {
      id,
      x: room.x + room.w * fx,
      y: room.y + room.h * fy,
    };
  }

  function handlePersonPosition(next: Point) {
    const activeMotionSensorIds = new Set<string>();
    devices.forEach((device) => {
      if (!isMotionSensor(device)) return;
      const pos = positionForDevice(device.id);
      const inZone = Math.hypot(next.x - pos.x, next.y - pos.y) <= MOTION_SENSOR_RADIUS;
      if (!inZone) return;
      if (hasWallBetween(next, pos)) return;

      activeMotionSensorIds.add(device.id);
      if (!triggeredMotionSensorsRef.current.has(device.id)) {
        triggeredMotionSensorsRef.current.add(device.id);
        onMotionSensorTrigger?.(device.id, next);
      }
    });

    for (const id of triggeredMotionSensorsRef.current) {
      if (!activeMotionSensorIds.has(id)) triggeredMotionSensorsRef.current.delete(id);
    }

    onPersonMove?.(next, Array.from(activeMotionSensorIds));
  }

  function pointFromPointer(clientX: number, clientY: number) {
    if (!surfaceRef.current) return null;
    const rect = surfaceRef.current.getBoundingClientRect();
    return {
      x: Math.min(0.955, Math.max(0.045, (clientX - rect.left) / rect.width)),
      y: Math.min(0.94, Math.max(0.06, (clientY - rect.top) / rect.height)),
    };
  }

  const forbiddenZones: ForbiddenZone[] = (() => {
    const zones: ForbiddenZone[] = [];

    function normalizedZoneKind(zone: FloorZoneView) {
      return `${zone.kind ?? zone.id}`.toLowerCase();
    }

    function addFloorZone(zone: FloorZoneView, forbiddenFor: DeviceType[], reason: string) {
      const bounds = zone.bounds;
      if (!bounds || bounds.w <= 0 || bounds.h <= 0) return;
      zones.push({
        id: zone.id,
        x: bounds.x,
        y: bounds.y,
        w: bounds.w,
        h: bounds.h,
        forbiddenFor,
        reason,
      });
    }

    function addRoomBand(
      roomId: string,
      band: { rx: number; ry: number; rw: number; rh: number },
      forbiddenFor: DeviceType[],
      reason: string
    ) {
      const room = roomMap.get(roomId);
      if (!room) return;
      zones.push({
        id: `${roomId}-${reason}`,
        x: room.x + room.w * band.rx,
        y: room.y + room.h * band.ry,
        w: room.w * band.rw,
        h: room.h * band.rh,
        forbiddenFor,
        reason,
      });
    }

    const windowForbid: DeviceType[] = ["pir", "mmwave", "temp", "humidity", "smoke", "gas", "co"];
    ["bedroom_1", "bedroom_2", "living", "kitchen"].forEach((id) =>
      addRoomBand(id, { rx: 0.05, ry: 0.02, rw: 0.9, rh: 0.12 }, windowForbid, "window")
    );

    const radiatorForbid: DeviceType[] = ["temp", "humidity", "pir"];
    ["bedroom_1", "bedroom_2", "living"].forEach((id) =>
      addRoomBand(id, { rx: 0.05, ry: 0.82, rw: 0.9, rh: 0.12 }, radiatorForbid, "radiator")
    );

    const kitchenForbid: DeviceType[] = ["smoke", "gas", "co"];
    addRoomBand("kitchen", { rx: 0.72, ry: 0.18, rw: 0.24, rh: 0.50 }, kitchenForbid, "stove");

    const bathroomForbid: DeviceType[] = ["camera"];
    addRoomBand("bath", { rx: 0.02, ry: 0.02, rw: 0.96, rh: 0.96 }, bathroomForbid, "privacy");

    floorPlan?.zones?.forEach((zone) => {
      const kind = normalizedZoneKind(zone);

      if (kind.includes("restricted")) {
        addFloorZone(zone, ALL_DEVICE_TYPES, "restricted");
        return;
      }

      if (kind.includes("window")) {
        addFloorZone(zone, windowForbid, "window");
        return;
      }

      if (kind.includes("no_wind")) {
        addFloorZone(zone, ["smoke", "gas", "co", "co2", "temp", "humidity", "climate_device"], "no_wind");
        return;
      }

      if (kind.includes("wet")) {
        addFloorZone(zone, ["pir", "mmwave", "camera", "smoke", "gas", "co", "co2", "temp", "humidity", "lux", "current", "light", "media"], "wet");
        return;
      }

      if (kind.includes("gas")) {
        addFloorZone(zone, ["leak", "water_flow", "camera", "humidity", "current", "media"], "gas");
      }
    });

    return zones;
  })();

  const fallbackBlockingWalls: WallSegment[] = [
    { kind: "vertical", x: 0.35, y1: 0.057, y2: 0.286 },
    { kind: "vertical", x: 0.35, y1: 0.357, y2: 0.5 },
    { kind: "vertical", x: 0.35, y1: 0.557, y2: 0.671 },
    { kind: "horizontal", y: 0.429, x1: 0.04, x2: 0.35 },
    { kind: "vertical", x: 0.7, y1: 0.057, y2: 0.286 },
    { kind: "vertical", x: 0.7, y1: 0.529, y2: 0.786 },
    { kind: "vertical", x: 0.7, y1: 0.829, y2: 0.943 },
    { kind: "horizontal", y: 0.6, x1: 0.7, x2: 0.96 },
    { kind: "horizontal", y: 0.671, x1: 0.04, x2: 0.45 },
    { kind: "horizontal", y: 0.671, x1: 0.6, x2: 0.7 },
  ];
  const blockingWalls = floorPlan?.blockers?.length ? floorPlan.blockers : fallbackBlockingWalls;

  function crossesWall(from: { x: number; y: number }, to: { x: number; y: number }, wall: WallSegment) {
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

  function movePerson(dx: number, dy: number) {
    if (routeWalking) return;
    setPersonPos((current) => {
      const next = {
        x: Math.min(0.955, Math.max(0.045, current.x + dx)),
        y: Math.min(0.94, Math.max(0.06, current.y + dy)),
      };

      if (blockingWalls.some((wall) => crossesWall(current, next, wall))) return current;
      setPersonWalking(true);
      setWalkCycle((cycle) => cycle + 1);
      if (walkTimerRef.current) clearTimeout(walkTimerRef.current);
      walkTimerRef.current = setTimeout(() => setPersonWalking(false), stepDurationMs());
      handlePersonPosition(next);
      return next;
    });
  }

  function stepDurationMs(speed = routeSpeed) {
    return PERSON_STEP_MS / speed;
  }

  function stepGapMs(speed = routeSpeed) {
    return ROUTE_STEP_GAP_MS / speed;
  }

  function hasWallBetween(from: Point, to: Point) {
    return blockingWalls.some((wall) => crossesWall(from, to, wall));
  }

  function currentVisualPersonPosition() {
    if (!surfaceRef.current || !personMarkerRef.current) return personPos;
    const surfaceRect = surfaceRef.current.getBoundingClientRect();
    const markerRect = personMarkerRef.current.getBoundingClientRect();
    return {
      x: Math.min(0.955, Math.max(0.045, (markerRect.left + markerRect.width / 2 - surfaceRect.left) / surfaceRect.width)),
      y: Math.min(0.94, Math.max(0.06, (markerRect.top + markerRect.height / 2 - surfaceRect.top) / surfaceRect.height)),
    };
  }

  function showRouteError(message: string) {
    setRouteError(message);
    if (routeTimerRef.current) clearTimeout(routeTimerRef.current);
    routeTimerRef.current = setTimeout(() => setRouteError(null), 1400);
  }

  function handlePlanClick(e: MouseEvent<HTMLDivElement>) {
    if (waterMode) {
      if (!surfaceRef.current || routeWalking) return;
      const rect = surfaceRef.current.getBoundingClientRect();
      onPlaceWater?.({
        x: Math.min(0.955, Math.max(0.045, (e.clientX - rect.left) / rect.width)),
        y: Math.min(0.94, Math.max(0.06, (e.clientY - rect.top) / rect.height)),
      });
      return;
    }

    if (fireMode) {
      if (!surfaceRef.current || routeWalking) return;
      const rect = surfaceRef.current.getBoundingClientRect();
      onPlaceFire?.({
        x: Math.min(0.955, Math.max(0.045, (e.clientX - rect.left) / rect.width)),
        y: Math.min(0.94, Math.max(0.06, (e.clientY - rect.top) / rect.height)),
      });
      return;
    }

    if (!routeMode || routeWalking || routePaused || !surfaceRef.current) return;

    const rect = surfaceRef.current.getBoundingClientRect();
    const next = {
      x: Math.min(0.955, Math.max(0.045, (e.clientX - rect.left) / rect.width)),
      y: Math.min(0.94, Math.max(0.06, (e.clientY - rect.top) / rect.height)),
    };
    const from = routePoints.length ? routePoints[routePoints.length - 1] : personPos;

    if (hasWallBetween(from, next)) {
      showRouteError("Точка за стеной");
      return;
    }

    setRouteError(null);
    setRoutePoints((points) => [...points, next]);
  }

  function clearRoute() {
    if (routeTimerRef.current) clearTimeout(routeTimerRef.current);
    if (walkTimerRef.current) clearTimeout(walkTimerRef.current);
    routePathRef.current = [];
    routeIndexRef.current = 0;
    pausedRouteRef.current = [];
    setRouteWalking(false);
    setRoutePaused(false);
    setPersonWalking(false);
    setRouteError(null);
    setRoutePoints([]);
  }

  function stopRoute() {
    if (routeTimerRef.current) clearTimeout(routeTimerRef.current);
    if (walkTimerRef.current) clearTimeout(walkTimerRef.current);

    const current = currentVisualPersonPosition();
    pausedRouteRef.current = routePathRef.current.slice(routeIndexRef.current);
    setPersonPos(current);

    setRouteWalking(false);
    setRoutePaused(true);
    setPersonWalking(false);
  }

  function resumeRoute() {
    const remaining = pausedRouteRef.current.length ? pausedRouteRef.current : buildWalkingPath(routePoints);
    if (!remaining.length || routeWalking) return;
    if (routeTimerRef.current) clearTimeout(routeTimerRef.current);
    setRoutePaused(false);
    setRouteWalking(true);
    routePathRef.current = remaining;
    routeIndexRef.current = 0;
    walkRoute(remaining, routeSpeed);
  }

  function walkRoute(points: Point[], speed: number, index = 0) {
    const next = points[index];
    if (!next) {
      routePathRef.current = [];
      routeIndexRef.current = 0;
      pausedRouteRef.current = [];
      setRouteWalking(false);
      setRoutePaused(false);
      setPersonWalking(false);
      return;
    }

    routePathRef.current = points;
    routeIndexRef.current = index;
    setPersonWalking(true);
    setWalkCycle((cycle) => cycle + 1);
    setPersonPos(next);
    handlePersonPosition(next);
    routeTimerRef.current = setTimeout(() => {
      walkRoute(points, speed, index + 1);
    }, stepDurationMs(speed) + stepGapMs(speed));
  }

  function buildWalkingPath(points: Point[]) {
    const path: Point[] = [];
    let from = personPos;

    points.forEach((to) => {
      const dx = to.x - from.x;
      const dy = to.y - from.y;
      const distance = Math.hypot(dx, dy);
      const steps = Math.max(1, Math.ceil(distance / PERSON_ROUTE_STEP));

      for (let step = 1; step <= steps; step += 1) {
        path.push({
          x: from.x + (dx * step) / steps,
          y: from.y + (dy * step) / steps,
        });
      }

      from = to;
    });

    return path;
  }

  function startRoute() {
    if (!routePoints.length || routeWalking || routePaused) return;
    if (hasWallBetween(personPos, routePoints[0])) {
      showRouteError("Первый шаг упирается в стену");
      return;
    }

    if (routeTimerRef.current) clearTimeout(routeTimerRef.current);
    pausedRouteRef.current = [];
    routePathRef.current = [];
    routeIndexRef.current = 0;
    setRouteWalking(true);
    setRoutePaused(false);
    setRouteMode(false);
    const path = buildWalkingPath(routePoints);
    routePathRef.current = path;
    walkRoute(path, routeSpeed);
  }

  function dotClass(id: string) {
    const st = deviceMap.get(id) ?? "idle";
    const isInChain = chainSet.has(id);
    const isActive = activeSet.has(id);
    const isLast = lastDevice === id;
    const base = [
      "absolute -translate-x-1/2 -translate-y-1/2",
      "rounded-full",
      "inline-flex items-center justify-center",
      "transition-shadow transition-transform",
      "plan-marker",
    ].join(" ");

    const glass = "text-white";
    const ring = isLast ? "ring-2 ring-[#0071e3]/50" : "";
    const chain = isInChain ? "border-white/30" : "";
    const active = isActive ? "scale-[1.08] shadow-[0_0_28px_rgba(0,113,227,0.42)]" : "";

    if (st === "active") return `${base} px-4 py-2 text-lg font-medium ${glass} ${ring} ${chain} ${active}`;
    if (st === "error") return `${base} px-4 py-2 text-lg font-medium bg-red-600/30 border border-red-400/20 ${ring} text-red-200 ${active}`;
    return `${base} px-4 py-2 text-lg font-medium ${glass} ${ring} ${chain} ${active} text-white/80`;
  }

  useEffect(() => {
    return () => {
      if (walkTimerRef.current) clearTimeout(walkTimerRef.current);
      if (routeTimerRef.current) clearTimeout(routeTimerRef.current);
    };
  }, []);

  useEffect(() => {
    function onMove(e: PointerEvent) {
      if (!personDragRef.current || routeWalking) return;
      const next = pointFromPointer(e.clientX, e.clientY);
      if (!next) return;
      setPersonWalking(false);
      setPersonPos(next);
      handlePersonPosition(next);
    }

    function onUp() {
      if (!personDragRef.current) return;
      personDragRef.current = false;
      setPersonDragging(false);
    }

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
    // handlePersonPosition is intentionally kept as the current render helper for this pointer subscription.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [routeWalking, devices, markers, onPersonMove, onMotionSensorTrigger]);

  useEffect(() => {
    if (!onMoveDevice) return;
    const moveDevice = onMoveDevice;

    function isForbidden(type: DeviceType, x: number, y: number) {
      return forbiddenZones.some(
        (zone) =>
          zone.forbiddenFor.includes(type) &&
          x >= zone.x &&
          x <= zone.x + zone.w &&
          y >= zone.y &&
          y <= zone.y + zone.h
      );
    }

    function onMove(e: PointerEvent) {
      const dragState = dragRef.current;
      if (!dragState || !surfaceRef.current) return;
      const rect = surfaceRef.current.getBoundingClientRect();
      const nx = (e.clientX - rect.left) / rect.width;
      const ny = (e.clientY - rect.top) / rect.height;
      const clampedX = Math.min(0.98, Math.max(0.02, nx));
      const clampedY = Math.min(0.98, Math.max(0.02, ny));
      moveDevice(dragState.id, clampedX, clampedY);

      if (!isForbidden(deviceTypeForId(dragState.id), clampedX, clampedY)) {
        lastValidRef.current[dragState.id] = { x: clampedX, y: clampedY };
      }
    }

    function onUp() {
      if (dragRef.current) {
        const id = dragRef.current.id;
        const last = lastValidRef.current[id];
        if (last) {
          moveDevice(id, last.x, last.y);
        }
      }
      dragRef.current = null;
      setDraggingType(null);
    }

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
  }, [forbiddenZones, onMoveDevice]);

  return (
    <section className="p-3">
      <div className="rounded-2xl border-0 bg-transparent p-4">
        <div className="route-toolbar" onClick={(e) => e.stopPropagation()}>
          <button
            type="button"
            className={`route-button${routeMode ? " route-button-active" : ""}`}
            disabled={routeWalking || routePaused}
            onClick={() => setRouteMode((value) => !value)}
          >
            {routeMode ? "Ставь точки" : "Выбрать маршрут"}
          </button>
          <button type="button" className="route-button" disabled={!routePoints.length || routeWalking || routePaused} onClick={startRoute}>
            Старт
          </button>
          <button
            type="button"
            className={`route-button ${routePaused ? "route-button-resume" : "route-button-stop"}`}
            disabled={!routeWalking && !routePaused}
            onClick={routePaused ? resumeRoute : stopRoute}
          >
            {routePaused ? "Продолжить" : "Стоп"}
          </button>
          <button type="button" className="route-button" disabled={!routePoints.length || routeWalking} onClick={clearRoute}>
            Очистить
          </button>
          <label className="route-speed" onClick={(e) => e.stopPropagation()}>
            <span>{routeSpeed.toFixed(1)}x</span>
            <input
              type="range"
              min="0.5"
              max="5"
              step="0.5"
              value={routeSpeed}
              disabled={routeWalking}
              onChange={(e) => setRouteSpeed(Number(e.target.value))}
            />
          </label>
        </div>

        <div className="fire-toolbar" onClick={(e) => e.stopPropagation()}>
          <span className="tool-group-label">Пожар</span>
          <button
            type="button"
            className={`route-button fire-button${fireMode ? " fire-button-active" : ""}`}
            disabled={fireActive}
            onClick={onToggleFireMode}
          >
            {fireMode ? "Укажи очаг" : "Начать пожар"}
          </button>
          <button type="button" className="route-button" disabled={!firePoint && !fireActive} onClick={onResetFire}>
            Сброс пожара
          </button>
        </div>

        <div className="water-toolbar" onClick={(e) => e.stopPropagation()}>
          <span className="tool-group-label">Потоп</span>
          <button
            type="button"
            className={`route-button water-button${waterMode ? " water-button-active" : ""}`}
            disabled={waterActive}
            onClick={onToggleWaterMode}
          >
            {waterMode ? "Укажи место" : "Начать потоп"}
          </button>
          <button type="button" className="route-button" disabled={!waterPoint && !waterActive} onClick={onResetWater}>
            Сброс потопа
          </button>
        </div>

        {(routeMode || routePoints.length > 0 || routeError || fireMode || fireActive || waterMode || waterActive) && (
          <div className={`route-hint${routeError ? " route-hint-error" : ""}`}>
            {routeError ??
              (fireMode
                ? "Кликни по плану, чтобы выбрать место возгорания"
                : fireActive
                ? "Идет симуляция пожара"
                : waterMode
                ? "Кликни по плану, чтобы выбрать место протечки"
                : waterActive
                ? "Идет симуляция потопа"
                : routePaused
                ? "Маршрут на паузе"
                : routeMode
                ? "Кликни по плану, чтобы поставить точку"
                : `${routePoints.length} точек`)}
          </div>
        )}

        <div
          ref={surfaceRef}
          className="plan-surface relative w-full aspect-[10/7] rounded-2xl overflow-hidden"
          onClick={handlePlanClick}
          onDragOver={(e) => {
            if (!onDropDevice) return;
            e.preventDefault();
            e.dataTransfer.dropEffect = "copy";
          }}
          onDrop={(e) => {
            if (!onDropDevice) return;
            const id = e.dataTransfer.getData("application/x-sim-device-id") || e.dataTransfer.getData("text/plain");
            if (!id || !surfaceRef.current) return;
            e.preventDefault();
            const rect = surfaceRef.current.getBoundingClientRect();
            onDropDevice(
              id,
              Math.min(0.98, Math.max(0.02, (e.clientX - rect.left) / rect.width)),
              Math.min(0.98, Math.max(0.02, (e.clientY - rect.top) / rect.height))
            );
          }}
          style={{
            border: "1px solid rgba(255,255,255,0.18)",
            background: "linear-gradient(180deg, #f5f5f7, #e8e8ed)",
            boxShadow: "inset 0 1px 0 rgba(255,255,255,0.82), 0 18px 44px rgba(0,0,0,0.20)",
            cursor: (routeMode && !routeWalking && !routePaused) || fireMode || waterMode ? "crosshair" : "default",
          }}
        >
          <svg
            className="absolute inset-0 w-full h-full pointer-events-none"
            viewBox={`0 0 ${floorViewBox.width ?? 1000} ${floorViewBox.height ?? 700}`}
            preserveAspectRatio="none"
          >
            <rect x="0" y="0" width="1000" height="700" fill="#f5f5f7" />

            {floorPlan?.furniture?.paths?.map((path, index) => (
              <path
                key={`furniture-${index}`}
                d={path}
                fill={floorPlan.furniture?.fill ?? "rgba(142,142,147,0.14)"}
                stroke={floorPlan.furniture?.stroke ?? "rgba(60,60,67,0.24)"}
                strokeWidth={floorPlan.furniture?.strokeWidth ?? 2}
              />
            ))}

            {floorPlan?.zones?.map((zone) => (
              <path
                key={`zone-${zone.id}`}
                d={zone.path}
                fill="rgba(0,113,227,0.08)"
                stroke="rgba(0,113,227,0.34)"
                strokeWidth={2}
                strokeDasharray="8 8"
              />
            ))}

            {(floorPlan?.walls?.paths?.length
              ? floorPlan.walls.paths
              : [
                  "M 40 40 L 960 40 L 960 660 L 40 660 Z",
                  "M 350 40 L 350 470",
                  "M 40 300 L 350 300",
                  "M 700 40 L 700 660",
                  "M 700 420 L 960 420",
                  "M 350 470 L 700 470",
                  "M 40 470 L 350 470",
                ]
            ).map((path, index) => (
              <path
                key={`wall-${index}`}
                d={path}
                fill="none"
                stroke={floorPlan?.walls?.stroke ?? "#86868b"}
                strokeWidth={floorPlan?.walls?.strokeWidth ?? 8}
                strokeLinecap="square"
              />
            ))}

            {floorPlan?.windows?.paths?.map((path, index) => (
              <path
                key={`window-${index}`}
                d={path}
                stroke={floorPlan.windows?.stroke ?? "#7cc7ff"}
                strokeWidth={floorPlan.windows?.strokeWidth ?? 9}
                strokeLinecap="round"
                fill="none"
              />
            ))}

            {(floorPlan?.doors?.paths?.length
              ? floorPlan.doors.paths
              : ["M 350 200 L 350 250", "M 350 350 L 350 390", "M 700 200 L 700 370", "M 700 550 L 700 580", "M 450 470 L 600 470"]
            ).map((path, index) => (
              <path
                key={`door-${index}`}
                d={path}
                stroke={floorPlan?.doors?.stroke ?? "#f5f5f7"}
                strokeWidth={floorPlan?.doors?.strokeWidth ?? 14}
                strokeLinecap="round"
                fill="none"
              />
            ))}
          </svg>

          <svg className="absolute inset-0 w-full h-full pointer-events-none" viewBox="0 0 100 100" preserveAspectRatio="none">
            {chains.map((c) =>
              c.chain.map((id, i) => {
                if (i === 0) return null;
                const prev = positionForDevice(c.chain[i - 1]);
                const curr = positionForDevice(id);
                return (
                  <line
                    key={`${c.id}-${i}`}
                    x1={prev.x * 100}
                    y1={prev.y * 100}
                    x2={curr.x * 100}
                    y2={curr.y * 100}
                    stroke={c.color}
                    strokeOpacity={0.35}
                    strokeWidth={0.5}
                    strokeDasharray="2 2"
                  />
                );
              })
            )}

            {activeEdges.map(([from, to], i) => {
              const a = positionForDevice(from);
              const b = positionForDevice(to);
              return (
                <line
                  key={`active-${from}-${to}-${i}`}
                  x1={a.x * 100}
                  y1={a.y * 100}
                  x2={b.x * 100}
                  y2={b.y * 100}
                  stroke="#0071e3"
                  strokeOpacity={0.9}
                  strokeWidth={0.9}
                />
              );
            })}

            {routePoints.length > 0 && (
              <polyline
                points={routePoints.map((p) => `${p.x * 100},${p.y * 100}`).join(" ")}
                fill="none"
                stroke="#0071e3"
                strokeOpacity={0.72}
                strokeWidth={0.7}
                strokeDasharray="2 1.4"
              />
            )}
          </svg>

          {routePoints.length > 0 && (
            <div className="route-points-layer">
              {routePoints.map((point, index) => (
                <div
                  key={`${point.x}-${point.y}-${index}`}
                  className="route-point"
                  style={{ left: `${point.x * 100}%`, top: `${point.y * 100}%` }}
                >
                  {index + 1}
                </div>
              ))}
            </div>
          )}

          <div className="motion-zones-layer">
            {devices
              .filter((device) => isMotionSensor(device))
              .map((device) => {
                const pos = positionForDevice(device.id);
                const isActive = activeSet.has(device.id) || deviceMap.get(device.id) === "active";
                return (
                  <div
                    key={`motion-zone-${device.id}`}
                    className={`motion-zone${isActive ? " motion-zone-active" : ""}`}
                    style={{
                      left: `${pos.x * 100}%`,
                      top: `${pos.y * 100}%`,
                      width: `${MOTION_SENSOR_RADIUS * 200}%`,
                      height: `${MOTION_SENSOR_RADIUS * 200}%`,
                    }}
                    title={`Зона ${device.id}`}
                  />
                );
              })}
          </div>

          {firePoints.length > 0 && (
            <svg className={`fire-spread-layer${fireActive ? " fire-spread-layer-active" : ""}`} viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
              <defs>
                <filter id="fire-soften" x="-20%" y="-20%" width="140%" height="140%">
                  <feGaussianBlur stdDeviation="1.8" />
                </filter>
                <radialGradient id="fire-pool" cx="50%" cy="50%" r="50%">
                  <stop offset="0%" stopColor="#fff5c2" stopOpacity="0.94" />
                  <stop offset="36%" stopColor="#ffb340" stopOpacity="0.72" />
                  <stop offset="70%" stopColor="#ff6a2a" stopOpacity="0.38" />
                  <stop offset="100%" stopColor="#ff3b1f" stopOpacity="0" />
                </radialGradient>
              </defs>
              <g filter="url(#fire-soften)">
                {firePoints.map((point, index) => (
                  <ellipse
                    key={`fire-${point.x}-${point.y}-${index}`}
                    cx={point.x * 100}
                    cy={point.y * 100}
                    rx={index === 0 ? 5.8 : 6.8}
                    ry={index === 0 ? 4.4 : 5.1}
                    fill="url(#fire-pool)"
                    opacity={index === 0 ? 1 : 0.82}
                  />
                ))}
              </g>
            </svg>
          )}

          {waterPoints.length > 0 && (
            <svg className="water-spread-layer" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
              <defs>
                <filter id="water-soften" x="-20%" y="-20%" width="140%" height="140%">
                  <feGaussianBlur stdDeviation="1.8" />
                </filter>
                <radialGradient id="water-pool" cx="50%" cy="50%" r="50%">
                  <stop offset="0%" stopColor="#f0fbff" stopOpacity="0.92" />
                  <stop offset="38%" stopColor="#64d2ff" stopOpacity="0.68" />
                  <stop offset="72%" stopColor="#0a84ff" stopOpacity="0.34" />
                  <stop offset="100%" stopColor="#0a84ff" stopOpacity="0" />
                </radialGradient>
              </defs>
              <g filter="url(#water-soften)">
                {waterPoints.map((point, index) => (
                  <ellipse
                    key={`water-${point.x}-${point.y}-${index}`}
                    cx={point.x * 100}
                    cy={point.y * 100}
                    rx={index === 0 ? 6.2 : 7.4}
                    ry={index === 0 ? 4.6 : 5.4}
                    fill="url(#water-pool)"
                    opacity={index === 0 ? 1 : 0.82}
                  />
                ))}
              </g>
            </svg>
          )}

          {rooms.map((r) => (
            <div
              key={r.id}
              className="absolute"
              style={{
                left: `${(r.labelX ?? r.x + r.w * 0.5) * 100}%`,
                top: `${(r.labelY ?? r.y + r.h * 0.5) * 100}%`,
                transform: "translate(-50%, -50%)",
                color: "#6e6e73",
                fontSize: "13px",
                fontWeight: 600,
                letterSpacing: 0,
                textTransform: "uppercase",
                pointerEvents: "none",
              }}
            >
              {r.title}
            </div>
          ))}

          {floorPlan?.zones?.map((zone) => (
            zone.label ? (
              <div
                key={`zone-label-${zone.id}`}
                className="absolute"
                style={{
                  left: `${zone.label.x * 100}%`,
                  top: `${zone.label.y * 100}%`,
                  transform: "translate(-50%, -50%)",
                  color: "#0071e3",
                  fontSize: "11px",
                  fontWeight: 700,
                  letterSpacing: 0,
                  textTransform: "uppercase",
                  pointerEvents: "none",
                  opacity: 0.72,
                }}
              >
                {zone.roomId ? `зона ${zone.roomId}` : "зона"}
              </div>
            ) : null
          ))}

          {draggingType &&
            forbiddenZones
              .filter((z) => z.forbiddenFor.includes(draggingType))
              .map((z) => (
                <div
                  key={z.id}
                  className="absolute"
                  style={{
                    left: `${z.x * 100}%`,
                    top: `${z.y * 100}%`,
                    width: `${z.w * 100}%`,
                    height: `${z.h * 100}%`,
                    background: "rgba(239,68,68,0.25)",
                    border: "1px dashed rgba(239,68,68,0.6)",
                    borderRadius: "10px",
                    pointerEvents: "none",
                  }}
                  title={z.reason}
                />
              ))}

          <div
            ref={personMarkerRef}
            className={`person-marker${personWalking ? " person-marker-walking" : ""}${personDragging ? " person-marker-dragging" : ""}`}
            style={
              {
                left: `${personPos.x * 100}%`,
                top: `${personPos.y * 100}%`,
                "--person-step-duration": `${stepDurationMs()}ms`,
              } as CSSProperties
            }
            title="Житель"
            onPointerDown={(e) => {
              if (routeWalking || e.target instanceof HTMLButtonElement) return;
              e.preventDefault();
              e.stopPropagation();
              personDragRef.current = true;
              setPersonDragging(true);
              const next = pointFromPointer(e.clientX, e.clientY);
              if (next) {
                setPersonPos(next);
                handlePersonPosition(next);
              }
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <svg key={walkCycle} className="person-figure" viewBox="0 0 32 44" aria-hidden="true">
              <circle className="person-head" cx="16" cy="6" r="5" />
              <path className="person-body" d="M16 13 L16 27" />
              <path className="person-arm person-arm-left" d="M16 15 L7 23" />
              <path className="person-arm person-arm-right" d="M16 15 L25 23" />
              <path className="person-leg person-leg-left" d="M16 27 L11 39" />
              <path className="person-leg person-leg-right" d="M16 27 L21 39" />
            </svg>
            <button type="button" className="person-arrow person-arrow-up" aria-label="Вверх" disabled={routeWalking} onClick={() => movePerson(0, -0.045)}>
              ↑
            </button>
            <button type="button" className="person-arrow person-arrow-left" aria-label="Влево" disabled={routeWalking} onClick={() => movePerson(-0.045, 0)}>
              ←
            </button>
            <button type="button" className="person-arrow person-arrow-right" aria-label="Вправо" disabled={routeWalking} onClick={() => movePerson(0.045, 0)}>
              →
            </button>
            <button type="button" className="person-arrow person-arrow-down" aria-label="Вниз" disabled={routeWalking} onClick={() => movePerson(0, 0.045)}>
              ↓
            </button>
          </div>

          {devices.map((d) => {
            const pos = positionForDevice(d.id);
            const isLightActive = isLightDevice(d.id) && (deviceMap.get(d.id) === "active" || activeSet.has(d.id));
            return (
              <div
                key={d.id}
                className={`${dotClass(d.id)}${fireActiveDeviceIds.includes(d.id) ? " fire-device-active" : ""}${
                  waterActiveDeviceIds.includes(d.id) ? " water-device-active" : ""
                }${isLightActive ? " light-device-active" : ""}`}
                style={{ left: `${pos.x * 100}%`, top: `${pos.y * 100}%` }}
                title={d.id}
                onPointerDown={(e) => {
                  if (!onMoveDevice) return;
                  e.preventDefault();
                  dragRef.current = { id: d.id };
                  const p = positionForDevice(d.id);
                  lastValidRef.current[d.id] = { x: p.x, y: p.y };
                  setDraggingType(deviceTypeForId(d.id));
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  onDeviceTrigger?.(d.id);
                }}
              >
                <span className="marker-label">{d.id}</span>
                {onRemoveDevice && (
                  <button
                    type="button"
                    className="device-remove-button"
                    aria-label={`Убрать ${d.id} с плана`}
                    title="Убрать с плана"
                    onPointerDown={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                    }}
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      onRemoveDevice(d.id);
                    }}
                  >
                    ×
                  </button>
                )}
              </div>
            );
          })}

          {disasterMessage && (
            <div className="disaster-overlay" role="status" aria-live="assertive">
              <div className="disaster-title">{disasterMessage}</div>
              <div className="disaster-subtitle">Система не успела обнаружить угрозу</div>
            </div>
          )}

        </div>

        <div className="console-surface mt-4 rounded-2xl px-4 py-3 font-mono text-base text-white/80">
          {chains.length ? chains.map((c) => c.chain.join(" → ")).join(" | ") : "—"}
        </div>
      </div>
    </section>
  );
}
