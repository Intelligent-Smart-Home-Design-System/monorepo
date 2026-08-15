"use client";

import { useEffect, useRef, useState, type CSSProperties, type MouseEvent } from "react";
import {
  Activity,
  AirVent,
  BellRing,
  Blinds,
  Bot,
  Camera,
  ChevronDown,
  ChevronUp,
  CircleGauge,
  Cloud,
  DoorOpen,
  Droplets,
  Fan,
  Flame,
  HousePlug,
  Lightbulb,
  LockKeyhole,
  PlugZap,
  Radio,
  ShieldAlert,
  SlidersHorizontal,
  Speaker,
  Thermometer,
  ToggleLeft,
  Tv,
  Waves,
  Wifi,
  type LucideIcon,
} from "lucide-react";
import type { Device, DeviceMarker, LogEvent, Room } from "@/app/simulation/Mockdata";
import type { FloorPlanView } from "@/app/simulation/floorAdapter";
import type { DeviceValueControl, IncidentKind } from "@/app/simulation/wsClient";

type Point = { x: number; y: number };
export type IncidentPolygon = {
  id: string;
  kind: IncidentKind;
  points: Point[];
};

type Props = {
  rooms: Room[];
  floorPlan?: FloorPlanView;
  markers: DeviceMarker[];
  devices: Device[];
  chains: { id: string; chain: string[]; color: string }[];
  highlightedEdges: Array<[string, string]>;
  activeEdges: Array<[string, string]>;
  lastEvent: LogEvent | null;
  onMoveDevice?: (id: string, x: number, y: number) => void;
  onDropDevice?: (id: string, x: number, y: number) => void;
  onRemoveDevice?: (id: string) => void;
  devicesLocked?: boolean;
  fireMode?: boolean;
  firePoint?: Point | null;
  fireActive?: boolean;
  onToggleFireMode?: () => void;
  onPlaceFire?: (point: Point) => void;
  onResetFire?: () => void;
  fireResetPending?: boolean;
  waterMode?: boolean;
  waterPoint?: Point | null;
  waterActive?: boolean;
  incidentPolygons?: IncidentPolygon[];
  onToggleWaterMode?: () => void;
  onPlaceWater?: (point: Point) => void;
  onResetWater?: () => void;
  waterResetPending?: boolean;
  smokeMode?: boolean;
  smokePoint?: Point | null;
  smokeActive?: boolean;
  onToggleSmokeMode?: () => void;
  onPlaceSmoke?: (point: Point) => void;
  onResetSmoke?: () => void;
  smokeResetPending?: boolean;
  motionSensorRadii: Record<string, Point>;
  personPosition: Point;
  personMovementEnabled?: boolean;
  onPersonMove?: (point: Point) => boolean;
  personRouteStatus?: string;
  onPersonRoute?: (points: Point[], speed: number) => boolean;
  onPersonRouteControl?: (action: "pause" | "resume" | "stop") => boolean;
  onDeviceTrigger?: (deviceId: string) => void;
  devicePercentLevels?: Record<string, number>;
  deviceValueControls?: Record<string, DeviceValueControl>;
  deviceLevelArcs?: Record<string, number>;
  onDevicePercentChange?: (deviceId: string, value: number) => void;
  onDeviceValueChange?: (deviceId: string, value: number) => void;
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
const PERSON_MOVE_STEP = 0.015;

export function ApartmentPlan({
  rooms,
  floorPlan,
  markers,
  devices,
  chains,
  highlightedEdges,
  activeEdges,
  lastEvent,
  onMoveDevice,
  onDropDevice,
  onRemoveDevice,
  devicesLocked = false,
  fireMode = false,
  firePoint = null,
  fireActive = false,
  onToggleFireMode,
  onPlaceFire,
  onResetFire,
  fireResetPending = false,
  waterMode = false,
  waterPoint = null,
  waterActive = false,
  incidentPolygons = [],
  onToggleWaterMode,
  onPlaceWater,
  onResetWater,
  waterResetPending = false,
  smokeMode = false,
  smokePoint = null,
  smokeActive = false,
  onToggleSmokeMode,
  onPlaceSmoke,
  onResetSmoke,
  smokeResetPending = false,
  motionSensorRadii,
  personPosition,
  personMovementEnabled = false,
  onPersonMove,
  personRouteStatus = "idle",
  onPersonRoute,
  onPersonRouteControl,
  onDeviceTrigger,
  devicePercentLevels = {},
  deviceValueControls = {},
  deviceLevelArcs = {},
  onDevicePercentChange,
  onDeviceValueChange,
}: Props) {
  const deviceMap = new Map(devices.map((d) => [d.id, d.status]));
  const lastDevice = lastEvent?.device ?? null;
  const roomMap = new Map(rooms.map((r) => [r.id, r]));
  const markerMap = new Map(markers.map((m) => [m.id, m]));
  const chainSet = new Set(chains.flatMap((c) => c.chain));
  const floorViewBox = floorPlan?.walls?.viewBox ?? floorPlan?.doors?.viewBox ?? { width: 1000, height: 700 };
  const surfaceRef = useRef<HTMLDivElement | null>(null);
  const dragRef = useRef<{ id: string } | null>(null);
  const personDragRef = useRef(false);
  const pendingPersonDragRef = useRef<Point | null>(null);
  const confirmedPersonPositionRef = useRef(personPosition);
  const lastValidRef = useRef<Record<string, { x: number; y: number }>>({});
  const walkTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const routeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [draggingType, setDraggingType] = useState<DeviceType | null>(null);
  const [personWalking, setPersonWalking] = useState(false);
  const [personDragging, setPersonDragging] = useState(false);
  const [walkCycle, setWalkCycle] = useState(0);
  const [routeMode, setRouteMode] = useState(false);
  const [routeError, setRouteError] = useState<string | null>(null);
  const [routePoints, setRoutePoints] = useState<Point[]>([]);
  const [routeSpeed, setRouteSpeed] = useState(1);
  const routeWalking = personRouteStatus === "running";
  const routePaused = personRouteStatus === "paused";
  const routeStatusError = personRouteStatus === "blocked" ? "Маршрут заблокирован стеной" : null;
  const personPos = personPosition;
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

  function iconForDevice(device: Pick<Device, "id" | "name" | "type">): LucideIcon {
    const key = `${device.id} ${device.name ?? ""} ${device.type ?? ""}`.toLowerCase();

    if (key.includes("curtain") || key.includes("blind") || key.includes("штор")) return Blinds;
    if (key.includes("dimmer") || key.includes("диммер")) return SlidersHorizontal;
    if (key.includes("button") || key.includes("switch") || key.includes("кноп") || key.includes("выключател")) return ToggleLeft;
    if (key.includes("lamp") || key.includes("light") || key.includes("led") || key.includes("свет")) return Lightbulb;
    if (key.includes("temperature") || key.includes("thermostat") || key.includes("radiator") || key.includes("heater") || key.includes("температур")) {
      return Thermometer;
    }
    if (key.includes("humidity") || key.includes("humidifier") || key.includes("влажност") || key.includes("увлажнител")) return Droplets;
    if (key.includes("leak") || key.includes("water") || key.includes("flood") || key.includes("протеч")) return Waves;
    if (key.includes("air purifier") || key.includes("air_purifier") || key.includes("ventilation") || key.includes("conditioner")) return AirVent;
    if (key.includes("fan") || key.includes("вентилятор")) return Fan;
    if (key.includes("speaker") || key.includes("колонк")) return Speaker;
    if (key.includes("tv") || key.includes("television") || key.includes("телевизор")) return Tv;
    if (key.includes("vacuum") || key.includes("robot") || key.includes("пылесос")) return Bot;
    if (key.includes("camera") || key.includes("камер")) return Camera;
    if (key.includes("lock") || key.includes("замок")) return LockKeyhole;
    if (key.includes("door") || key.includes("window") || key.includes("двер") || key.includes("окн")) return DoorOpen;
    if (key.includes("siren") || key.includes("alarm") || key.includes("сирен")) return BellRing;
    if (key.includes("smoke") || key.includes("fire") || key.includes("дым") || key.includes("пожар")) return Flame;
    if (key.includes("gas") || key.includes("co2") || key.includes("co sensor") || key.includes("газ")) return Cloud;
    if (key.includes("motion") || key.includes("presence") || key.includes("pir") || key.includes("движен")) return Activity;
    if (key.includes("plug") || key.includes("power") || key.includes("current") || key.includes("розет")) return PlugZap;
    if (key.includes("hub") || key.includes("gateway") || key.includes("controller") || key.includes("хаб")) return HousePlug;
    if (key.includes("wifi") || key.includes("wireless")) return Wifi;
    if (key.includes("radio")) return Radio;
    if (key.includes("sensor") || key.includes("датчик")) return ShieldAlert;
    if (key.includes("droplet")) return Droplets;

    return CircleGauge;
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

  function requestPersonPosition(next: Point) {
    if (!personMovementEnabled || !onPersonMove) return false;
    const clamped = {
      x: Math.min(0.955, Math.max(0.045, next.x)),
      y: Math.min(0.94, Math.max(0.06, next.y)),
    };
    return onPersonMove(clamped);
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

    return zones;
  })();

  const fireIncidentPolygons = incidentPolygons.filter((polygon) => polygon.kind === "fire:spread");
  const floodIncidentPolygons = incidentPolygons.filter((polygon) => polygon.kind === "flood:spread");
  const smokeIncidentPolygons = incidentPolygons.filter((polygon) => polygon.kind === "smoke:spread");

  function movePerson(dx: number, dy: number) {
    if (routeWalking || routePaused || !personMovementEnabled) return;
    const current = confirmedPersonPositionRef.current;
    requestPersonPosition({ x: current.x + dx, y: current.y + dy });
  }

  function stepDurationMs(speed = routeSpeed) {
    return PERSON_STEP_MS / speed;
  }

  function svgPolygonPoints(points: Point[]) {
    return points.map((point) => `${point.x * 100},${point.y * 100}`).join(" ");
  }

  function showRouteError(message: string) {
    setRouteError(message);
    if (routeTimerRef.current) clearTimeout(routeTimerRef.current);
    routeTimerRef.current = setTimeout(() => setRouteError(null), 1400);
  }

  function handlePlanClick(e: MouseEvent<HTMLDivElement>) {
    if (smokeMode) {
      if (!surfaceRef.current || routeWalking) return;
      const rect = surfaceRef.current.getBoundingClientRect();
      onPlaceSmoke?.({
        x: Math.min(0.955, Math.max(0.045, (e.clientX - rect.left) / rect.width)),
        y: Math.min(0.94, Math.max(0.06, (e.clientY - rect.top) / rect.height)),
      });
      return;
    }

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
    setRouteError(null);
    setRoutePoints((points) => [...points, next]);
  }

  function clearRoute() {
    if ((routeWalking || routePaused) && !onPersonRouteControl?.("stop")) return;
    setRouteError(null);
    setRoutePoints([]);
  }

  function pauseRoute() {
    if (!onPersonRouteControl?.("pause")) return;
  }

  function resumeRoute() {
    if (!onPersonRouteControl?.("resume")) return;
  }

  function startRoute() {
    if (!routePoints.length || routeWalking || routePaused || !personMovementEnabled || !onPersonRoute) return;
    if (!onPersonRoute(routePoints, routeSpeed)) {
      showRouteError("Маршрут не отправлен");
      return;
    }
    setRouteMode(false);
  }

  function dotClass(id: string, hasLevelArc: boolean) {
    const st = deviceMap.get(id) ?? "idle";
    const isInChain = chainSet.has(id);
    const isLast = lastDevice === id;
    const base = [
      "absolute -translate-x-1/2 -translate-y-1/2",
      "rounded-full",
      "inline-flex items-center justify-center",
      "transition-shadow transition-transform",
      "plan-marker",
      "device-marker",
    ].join(" ");

    const ring = isLast ? "ring-2 ring-[#0071e3]/50" : "";
    const chain = isInChain ? "border-white/30" : "";
    const active = st === "active" && !hasLevelArc ? "device-marker-active" : "";
    const level = hasLevelArc ? "device-marker-level" : "";

    if (st === "error") return `${base} device-marker-error ${ring}`;
    return `${base} ${ring} ${chain} ${active} ${level}`;
  }

  useEffect(() => {
    return () => {
      if (walkTimerRef.current) clearTimeout(walkTimerRef.current);
      if (routeTimerRef.current) clearTimeout(routeTimerRef.current);
    };
  }, []);

  useEffect(() => {
    const previous = confirmedPersonPositionRef.current;
    confirmedPersonPositionRef.current = personPosition;
    if (previous.x === personPosition.x && previous.y === personPosition.y) return;

    setPersonWalking(true);
    setWalkCycle((cycle) => cycle + 1);
    if (walkTimerRef.current) clearTimeout(walkTimerRef.current);
    walkTimerRef.current = setTimeout(() => setPersonWalking(false), stepDurationMs());
    // stepDurationMs depends only on the current route speed used for the visual animation.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [personPosition.x, personPosition.y]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (!personMovementEnabled || routeWalking || routePaused || event.altKey || event.ctrlKey || event.metaKey) return;
      const target = event.target;
      if (
        target instanceof HTMLInputElement ||
        target instanceof HTMLTextAreaElement ||
        target instanceof HTMLSelectElement ||
        (target instanceof HTMLElement && target.isContentEditable)
      ) {
        return;
      }

      const deltas: Record<string, Point> = {
        ArrowUp: { x: 0, y: -PERSON_MOVE_STEP },
        ArrowDown: { x: 0, y: PERSON_MOVE_STEP },
        ArrowLeft: { x: -PERSON_MOVE_STEP, y: 0 },
        ArrowRight: { x: PERSON_MOVE_STEP, y: 0 },
      };
      const delta = deltas[event.key];
      if (!delta) return;
      event.preventDefault();
      movePerson(delta.x, delta.y);
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
    // movePerson reads the ref containing the latest backend-confirmed position.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [personMovementEnabled, routeWalking, routePaused, onPersonMove]);

  useEffect(() => {
    function onMove(e: PointerEvent) {
      if (!personDragRef.current || routeWalking || routePaused || !personMovementEnabled) return;
      const next = pointFromPointer(e.clientX, e.clientY);
      if (!next) return;
      pendingPersonDragRef.current = next;
    }

    function onUp() {
      if (!personDragRef.current) return;
      personDragRef.current = false;
      setPersonDragging(false);
      const next = pendingPersonDragRef.current;
      pendingPersonDragRef.current = null;
      if (next) requestPersonPosition(next);
    }

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
    // requestPersonPosition reads the latest callback and confirmed position from the current render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [routeWalking, routePaused, personMovementEnabled, onPersonMove]);

  useEffect(() => {
    if (!onMoveDevice || devicesLocked) return;
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
  }, [devicesLocked, forbiddenZones, onMoveDevice]);

  return (
    <section className="simulation-plan-section">
      <div className="simulation-plan-frame">
        <div className="route-toolbar" onClick={(e) => e.stopPropagation()}>
          <button
            type="button"
            className={`route-button${routeMode ? " route-button-active" : ""}`}
            disabled={!personMovementEnabled || routeWalking || routePaused}
            onClick={() => setRouteMode((value) => !value)}
          >
            {routeMode ? "Ставь точки" : "Выбрать маршрут для человека"}
          </button>
          <button
            type="button"
            className="route-button"
            disabled={!personMovementEnabled || !routePoints.length || routeWalking || routePaused}
            onClick={startRoute}
          >
            Старт
          </button>
          <button
            type="button"
            className={`route-button ${routePaused ? "route-button-resume" : "route-button-stop"}`}
            disabled={!routeWalking && !routePaused}
            onClick={routePaused ? resumeRoute : pauseRoute}
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

        <div className="incident-toolbar">
          <div className="fire-toolbar" onClick={(e) => e.stopPropagation()}>
            <span className="tool-group-label">Пожар</span>
            <button
              type="button"
              className={`route-button fire-button${fireMode ? " fire-button-active" : ""}`}
              disabled={fireActive}
              onClick={onToggleFireMode}
              data-testid="fire-start"
            >
              {fireMode ? "Укажи очаг" : "Начать пожар"}
            </button>
            <button
              type="button"
              className="route-button"
              disabled={(!firePoint && !fireActive) || fireResetPending}
              onClick={onResetFire}
              data-testid="fire-reset"
            >
              {fireResetPending ? "Сбрасываем..." : "Сброс пожара"}
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
            <button
              type="button"
              className="route-button"
              disabled={(!waterPoint && !waterActive) || waterResetPending}
              onClick={onResetWater}
            >
              {waterResetPending ? "Сбрасываем..." : "Сброс потопа"}
            </button>
          </div>

          <div className="smoke-toolbar" onClick={(e) => e.stopPropagation()}>
            <span className="tool-group-label">Дым</span>
            <button
              type="button"
              className={`route-button smoke-button${smokeMode ? " smoke-button-active" : ""}`}
              disabled={smokeActive}
              onClick={onToggleSmokeMode}
              data-testid="smoke-start"
            >
              {smokeMode ? "Укажи очаг" : "Начать дым"}
            </button>
            <button
              type="button"
              className="route-button"
              disabled={(!smokePoint && !smokeActive) || smokeResetPending}
              onClick={onResetSmoke}
              data-testid="smoke-reset"
            >
              {smokeResetPending ? "Сбрасываем..." : "Сброс дыма"}
            </button>
          </div>
        </div>

        {(routeMode || routePoints.length > 0 || routeError || routeStatusError) && (
          <div className={`route-hint${routeError || routeStatusError ? " route-hint-error" : ""}`}>
            {routeError ??
              routeStatusError ??
              (routePaused
                ? "Маршрут на паузе"
                : routeMode
                ? "Кликни по плану, чтобы поставить точку"
                : `${routePoints.length} точек`)}
          </div>
        )}

        <div
          ref={surfaceRef}
          className="plan-surface relative w-full aspect-[10/7] rounded-2xl overflow-hidden"
          data-testid="plan-surface"
          onClick={handlePlanClick}
          onDragOver={(e) => {
            if (!onDropDevice || devicesLocked) return;
            e.preventDefault();
            e.dataTransfer.dropEffect = "copy";
          }}
          onDrop={(e) => {
            if (!onDropDevice || devicesLocked) return;
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
            cursor:
              (routeMode && !routeWalking && !routePaused) || fireMode || waterMode || smokeMode
                ? "crosshair"
                : "default",
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

            {highlightedEdges.map(([from, to], i) => {
              const a = positionForDevice(from);
              const b = positionForDevice(to);
              return (
                <line
                  key={`highlighted-${from}-${to}-${i}`}
                  x1={a.x * 100}
                  y1={a.y * 100}
                  x2={b.x * 100}
                  y2={b.y * 100}
                  stroke="#30d158"
                  strokeOpacity={1}
                  strokeWidth={1.1}
                  style={{ filter: "drop-shadow(0 0 1.5px rgba(48, 209, 88, 0.9))" }}
                />
              );
            })}

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
                const radius = motionSensorRadii[device.id];
                if (!radius) return null;
                const isActive = deviceMap.get(device.id) === "active";
                return (
                  <div
                    key={`motion-zone-${device.id}`}
                    className={`motion-zone${isActive ? " motion-zone-active" : ""}`}
                    style={{
                      left: `${pos.x * 100}%`,
                      top: `${pos.y * 100}%`,
                      width: `${radius.x * 200}%`,
                      height: `${radius.y * 200}%`,
                    }}
                    title={`Зона ${device.id}`}
                  />
                );
              })}
          </div>

          {incidentPolygons.length > 0 && (
            <svg className="incident-block-layer" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true" data-testid="incident-layer">
              <defs>
                <filter id="incident-soften" x="-8%" y="-8%" width="116%" height="116%">
                  <feGaussianBlur stdDeviation="0.25" />
                </filter>
              </defs>
              <g filter="url(#incident-soften)">
                {floodIncidentPolygons.map((polygon) => (
                  <polygon
                    key={polygon.id}
                    points={svgPolygonPoints(polygon.points)}
                    fill="rgba(10, 132, 255, 0.42)"
                    stroke="rgba(100, 210, 255, 0.74)"
                    strokeWidth="0.12"
                  />
                ))}
                {smokeIncidentPolygons.map((polygon) => (
                  <polygon
                    key={polygon.id}
                    points={svgPolygonPoints(polygon.points)}
                    fill="rgba(99, 99, 102, 0.28)"
                    stroke="rgba(99, 99, 102, 0.24)"
                    strokeWidth="0.08"
                  />
                ))}
                {fireIncidentPolygons.map((polygon) => (
                  <polygon
                    key={polygon.id}
                    points={svgPolygonPoints(polygon.points)}
                    fill="rgba(255, 95, 46, 0.54)"
                    stroke="rgba(255, 179, 64, 0.82)"
                    strokeWidth="0.12"
                  />
                ))}
              </g>
            </svg>
          )}

          {rooms.map((r) => (
            <div
              key={r.id}
              className="plan-room-label"
              style={{
                left: `${r.x * 100}%`,
                top: `${(r.labelY ?? r.y + r.h * 0.5) * 100}%`,
                width: `${r.w * 100}%`,
                fontSize: r.w < 0.08 || r.h < 0.06 ? "9px" : r.w < 0.14 ? "11px" : "13px",
              }}
              title={r.title}
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
              if (!personMovementEnabled || routeWalking || routePaused || e.target instanceof HTMLButtonElement) return;
              e.preventDefault();
              e.stopPropagation();
              personDragRef.current = true;
              setPersonDragging(true);
              const next = pointFromPointer(e.clientX, e.clientY);
              pendingPersonDragRef.current = next;
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
            <button
              type="button"
              className="person-arrow person-arrow-up"
              aria-label="Вверх"
              disabled={!personMovementEnabled || routeWalking || routePaused}
              onClick={() => movePerson(0, -PERSON_MOVE_STEP)}
            >
              ↑
            </button>
            <button
              type="button"
              className="person-arrow person-arrow-left"
              aria-label="Влево"
              disabled={!personMovementEnabled || routeWalking || routePaused}
              onClick={() => movePerson(-PERSON_MOVE_STEP, 0)}
            >
              ←
            </button>
            <button
              type="button"
              className="person-arrow person-arrow-right"
              aria-label="Вправо"
              disabled={!personMovementEnabled || routeWalking || routePaused}
              onClick={() => movePerson(PERSON_MOVE_STEP, 0)}
            >
              →
            </button>
            <button
              type="button"
              className="person-arrow person-arrow-down"
              aria-label="Вниз"
              disabled={!personMovementEnabled || routeWalking || routePaused}
              onClick={() => movePerson(0, PERSON_MOVE_STEP)}
            >
              ↓
            </button>
          </div>

          {devices.map((d) => {
            const pos = positionForDevice(d.id);
            const name = d.name || markerMap.get(d.id)?.label || d.id;
            const DeviceIcon = iconForDevice(d);
            const tooltipId = `device-tooltip-${d.id}`;
            const tooltipClass = pos.x > 0.72 ? "device-tooltip device-tooltip-left" : "device-tooltip";
            const hasPercentControl = Object.prototype.hasOwnProperty.call(devicePercentLevels, d.id);
            const percentLevel = devicePercentLevels[d.id] ?? 0;
            const valueControl = deviceValueControls[d.id];
            const hasValueControl = valueControl !== undefined;
            const numericControl = hasPercentControl
              ? { value: percentLevel, min: 0, max: 100, step: 1, unit: "%", label: "Уровень" }
              : valueControl;
            const hasLevelArc = Object.prototype.hasOwnProperty.call(deviceLevelArcs, d.id);
            const levelArc = Math.min(100, Math.max(0, deviceLevelArcs[d.id] ?? 0));
            return (
              <div
                key={d.id}
                className={dotClass(d.id, hasLevelArc)}
                data-testid={`device-${d.id}`}
                data-device-state={deviceMap.get(d.id) ?? "idle"}
                data-device-level={hasLevelArc ? levelArc : undefined}
                style={{ left: `${pos.x * 100}%`, top: `${pos.y * 100}%` }}
                role="button"
                tabIndex={0}
                aria-label={name}
                aria-describedby={tooltipId}
                onPointerDown={(e) => {
                  if (!onMoveDevice || devicesLocked) return;
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
                onKeyDown={(e) => {
                  if (e.key !== "Enter" && e.key !== " ") return;
                  e.preventDefault();
                  e.stopPropagation();
                  onDeviceTrigger?.(d.id);
                }}
              >
                {hasLevelArc && (
                  <svg
                    className={`device-level-ring${levelArc <= 0 ? " device-level-ring-empty" : ""}`}
                    viewBox="0 0 48 48"
                    aria-hidden="true"
                  >
                    <circle
                      cx="24"
                      cy="24"
                      r="21"
                      pathLength="100"
                      style={{ strokeDashoffset: 100 - levelArc } as CSSProperties}
                    />
                  </svg>
                )}
                <DeviceIcon className="device-marker-icon" size={22} strokeWidth={2} aria-hidden="true" />
                {numericControl ? (
                  <form
                    id={tooltipId}
                    className={`${tooltipClass} device-percent-control`}
                    aria-label={`${numericControl.label} устройства ${name}`}
                    onPointerDown={(e) => e.stopPropagation()}
                    onClick={(e) => e.stopPropagation()}
                    onSubmit={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      const form = new FormData(e.currentTarget);
                      const value = Number(form.get("value"));
                      if (!Number.isFinite(value)) return;
                      if (hasPercentControl) {
                        onDevicePercentChange?.(d.id, value);
                      } else if (hasValueControl) {
                        onDeviceValueChange?.(d.id, value);
                      }
                    }}
                  >
                    <span className="device-percent-name">{name}</span>
                    <div className="device-percent-row">
                      <div className="device-percent-input-wrap">
                        <input
                          className="device-percent-input"
                          type="number"
                          name="value"
                          min={numericControl.min}
                          max={numericControl.max}
                          step={numericControl.step}
                          defaultValue={numericControl.value}
                          key={numericControl.value}
                          aria-label={`${numericControl.label} устройства ${name}`}
                        />
                        <div className="device-percent-stepper">
                          <button
                            type="button"
                            className="device-percent-step"
                            aria-label={`Увеличить значение ${name}`}
                            onClick={(event) => {
                              const input = event.currentTarget
                                .closest(".device-percent-input-wrap")
                                ?.querySelector<HTMLInputElement>(".device-percent-input");
                              input?.stepUp();
                            }}
                          >
                            <ChevronUp size={14} strokeWidth={2.5} aria-hidden="true" />
                          </button>
                          <button
                            type="button"
                            className="device-percent-step"
                            aria-label={`Уменьшить значение ${name}`}
                            onClick={(event) => {
                              const input = event.currentTarget
                                .closest(".device-percent-input-wrap")
                                ?.querySelector<HTMLInputElement>(".device-percent-input");
                              input?.stepDown();
                            }}
                          >
                            <ChevronDown size={14} strokeWidth={2.5} aria-hidden="true" />
                          </button>
                        </div>
                      </div>
                      <span className="device-percent-unit">{numericControl.unit}</span>
                      <button className="device-percent-submit" type="submit">
                        ОК
                      </button>
                    </div>
                  </form>
                ) : (
                  <span id={tooltipId} className={tooltipClass} role="tooltip">
                    {name}
                  </span>
                )}
                {onRemoveDevice && !devicesLocked && (
                  <button
                    type="button"
                    className="device-remove-button"
                    aria-label={`Убрать ${name} с плана`}
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

        </div>
      </div>
    </section>
  );
}
