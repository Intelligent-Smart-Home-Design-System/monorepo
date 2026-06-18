"use client";

import { useEffect, useMemo, useRef } from "react";
import type { ApartmentPlanRenderHandle, FloorPlan, SmartDevice, SmartDeviceType, Zone } from "smart-plan-demo";
import { renderApartmentPlan } from "smart-plan-demo";

type ApartmentPlanPreviewProps = {
  floor: unknown;
  devices?: unknown;
  zones?: unknown;
};

export function ApartmentPlanPreview({ floor, devices, zones }: ApartmentPlanPreviewProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const handleRef = useRef<ApartmentPlanRenderHandle | null>(null);
  const plan = useMemo(() => normalizeFloorPlan(floor), [floor]);
  const normalizedZones = useMemo(() => normalizeZones(zones), [zones]);
  const normalizedDevices = useMemo(() => normalizeDevices(devices, plan), [devices, plan]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !plan) return;

    if (!handleRef.current) {
      handleRef.current = renderApartmentPlan(container, plan, normalizedDevices, normalizedZones);
      return;
    }

    handleRef.current.update(plan, normalizedDevices, normalizedZones);
  }, [normalizedDevices, normalizedZones, plan]);

  useEffect(
    () => () => {
      handleRef.current?.destroy();
      handleRef.current = null;
    },
    []
  );

  if (!plan) {
    return null;
  }

  return (
    <div
      ref={containerRef}
      style={{
        width: "100%",
        height: "100%",
        minHeight: 0,
        overflow: "hidden",
      }}
    />
  );
}

function normalizeFloorPlan(value: unknown): FloorPlan | null {
  const candidate = findFloorCandidate(value);
  if (!candidate || typeof candidate !== "object") return null;

  const record = candidate as Record<string, unknown>;
  if (!Array.isArray(record.walls) || !Array.isArray(record.doors) || !Array.isArray(record.windows) || !Array.isArray(record.rooms)) {
    return null;
  }

  return {
    walls: record.walls as FloorPlan["walls"],
    doors: record.doors as FloorPlan["doors"],
    windows: record.windows as FloorPlan["windows"],
    rooms: record.rooms as FloorPlan["rooms"],
  };
}

function normalizeDevices(value: unknown, plan: FloorPlan | null): SmartDevice[] {
  if (!plan || !Array.isArray(value)) return [];

  return value.flatMap((item, index) => {
    const record = item && typeof item === "object" ? (item as Record<string, unknown>) : null;
    if (!record) return [];

    const id = toText(record.id ?? record.device_id ?? record.deviceId, `device_${index + 1}`);
    const deviceType = normalizeDeviceType(record.type ?? record.device_type ?? record.deviceType ?? record.name);
    const room = pickRoomForDevice(plan, record, index);
    const position = normalizeDevicePosition(record, room, index);
    const price = typeof record.price === "number" && Number.isFinite(record.price) ? record.price : undefined;
    const ecosystem = typeof record.ecosystem === "string" && record.ecosystem.trim() ? record.ecosystem.trim() : undefined;

    return [
      {
        id,
        type: deviceType,
        room_id: room.id,
        position,
        state: defaultDeviceState(deviceType),
        price,
        ecosystem,
      } as SmartDevice,
    ];
  });
}

function normalizeDeviceType(value: unknown): SmartDeviceType {
  const raw = toText(value, "").toLowerCase();
  if (raw.includes("motion") || raw.includes("presence") || raw.includes("pir")) return "motion_sensor";
  if (raw.includes("temperature") || raw.includes("thermostat") || raw.includes("climate") || raw.includes("co2")) return "temperature_sensor";
  if (raw.includes("leak") || raw.includes("water") || raw.includes("gas")) return "water_leak_sensor";
  if (raw.includes("plug") || raw.includes("switch") || raw.includes("button") || raw.includes("relay")) return "smart_plug";
  return "smart_lamp";
}

function defaultDeviceState(type: SmartDeviceType): SmartDevice["state"] {
  switch (type) {
    case "motion_sensor":
      return { detected: false };
    case "temperature_sensor":
      return { value: 23 };
    case "water_leak_sensor":
      return { leak_detected: false };
    case "smart_plug":
      return { is_on: false };
    case "smart_lamp":
    default:
      return { is_on: false };
  }
}

function pickRoomForDevice(plan: FloorPlan, record: Record<string, unknown>, index: number): FloorPlan["rooms"][number] {
  const roomId = toText(record.room_id ?? record.roomId, "");
  const byId = roomId ? plan.rooms.find((room) => room.id === roomId) : undefined;
  if (byId) return byId;

  const name = toText(record.name ?? record.type ?? record.device_type, "").toLowerCase();
  const matched =
    plan.rooms.find((room) => name.includes(room.id.toLowerCase()) || name.includes(room.name.toLowerCase())) ??
    plan.rooms[index % Math.max(plan.rooms.length, 1)];

  return matched ?? { id: "room", name: "Комната", area: [[0, 0], [1000, 0], [1000, 1000], [0, 1000]] };
}

function normalizeDevicePosition(record: Record<string, unknown>, room: FloorPlan["rooms"][number], index: number): SmartDevice["position"] {
  const rawPosition = record.position && typeof record.position === "object" ? (record.position as Record<string, unknown>) : null;
  const x = toNumber(record.x ?? rawPosition?.x, NaN);
  const y = toNumber(record.y ?? rawPosition?.y, NaN);
  if (Number.isFinite(x) && Number.isFinite(y)) return { x, y };

  const box = roomBounds(room);
  const col = index % 3;
  const row = Math.floor(index / 3) % 3;
  return {
    x: box.minX + box.width * (0.28 + col * 0.22),
    y: box.minY + box.height * (0.28 + row * 0.22),
  };
}

function roomBounds(room: FloorPlan["rooms"][number]) {
  const xs = room.area.map((point) => point[0]);
  const ys = room.area.map((point) => point[1]);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  return {
    minX,
    minY,
    width: Math.max(maxX - minX, 1),
    height: Math.max(maxY - minY, 1),
  };
}

function findFloorCandidate(value: unknown): unknown {
  if (!value || typeof value !== "object") return null;

  const record = value as Record<string, unknown>;
  if (Array.isArray(record.walls) && Array.isArray(record.rooms)) {
    return value;
  }

  for (const key of ["floor", "floorJson", "parsedFloor", "parsed_floor_plan", "floor_plan", "apartment", "plan"]) {
    const match = findFloorCandidate(record[key]);
    if (match) return match;
  }

  return null;
}

function normalizeZones(value: unknown): Zone[] {
  if (!Array.isArray(value)) return [];
  return value as Zone[];
}

function toText(value: unknown, fallback: string): string {
  return typeof value === "string" && value.trim() ? value.trim() : fallback;
}

function toNumber(value: unknown, fallback: number): number {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return fallback;
}
