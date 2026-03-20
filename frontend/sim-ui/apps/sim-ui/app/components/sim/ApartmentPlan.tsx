"use client";

import { useEffect, useRef, useState } from "react";
import type { Device, DeviceMarker, LogEvent, Room } from "@/app/simulation/Mockdata";

type Props = {
  rooms: Room[];
  markers: DeviceMarker[];
  devices: Device[];
  chains: { id: string; chain: string[]; color: string }[];
  activeNodes: string[];
  activeEdges: Array<[string, string]>;
  lastEvent: LogEvent | null;
  onMoveDevice?: (id: string, x: number, y: number) => void;
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

export function ApartmentPlan({
  rooms,
  markers,
  devices,
  chains,
  activeNodes,
  activeEdges,
  lastEvent,
  onMoveDevice,
}: Props) {
  const deviceMap = new Map(devices.map((d) => [d.id, d.status]));
  const lastDevice = lastEvent?.device ?? null;
  const roomMap = new Map(rooms.map((r) => [r.id, r]));
  const markerMap = new Map(markers.map((m) => [m.id, m]));
  const chainSet = new Set(chains.flatMap((c) => c.chain));
  const activeSet = new Set(activeNodes);
  const surfaceRef = useRef<HTMLDivElement | null>(null);
  const dragRef = useRef<{ id: string } | null>(null);
  const lastValidRef = useRef<Record<string, { x: number; y: number }>>({});
  const [draggingType, setDraggingType] = useState<DeviceType | null>(null);
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
    const ring = isLast ? "ring-2 ring-sky-300/50" : "";
    const chain = isInChain ? "border-white/30" : "";
    const active = isActive ? "scale-[1.08] shadow-[0_0_25px_rgba(56,189,248,0.35)]" : "";

    if (st === "active") return `${base} px-4 py-2 text-lg font-medium ${glass} ${ring} ${chain} ${active}`;
    if (st === "error") return `${base} px-4 py-2 text-lg font-medium bg-red-600/30 border border-red-400/20 ${ring} text-red-200 ${active}`;
    return `${base} px-4 py-2 text-lg font-medium ${glass} ${ring} ${chain} ${active} text-white/80`;
  }

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
        <div
          ref={surfaceRef}
          className="plan-surface relative w-full aspect-[10/7] rounded-2xl overflow-hidden"
          style={{
            border: "1px solid rgba(0,0,0,0.12)",
            background: "linear-gradient(180deg, #3b3e41, #4d5053)",
            boxShadow: "inset 0 1px 0 rgba(255,255,255,0.2), 0 18px 40px rgba(0,0,0,0.26)",
            transform: "scale(0.7)",
            transformOrigin: "center",
          }}
        >
          <svg className="absolute inset-0 w-full h-full pointer-events-none" viewBox="0 0 1000 700" preserveAspectRatio="none">
            <rect x="0" y="0" width="1000" height="700" fill="#676a6e" />

            <path
              d="M 40 40 L 960 40 L 960 660 L 40 660 Z"
              fill="none"
              stroke="#2f343a"
              strokeWidth="8"
            />

            <path
              d="
                M 350 40 L 350 470
                M 40 300 L 350 300
                M 700 40 L 700 660
                M 700 420 L 960 420
                M 350 470 L 700 470
                M 40 470 L 350 470
              "
              fill="none"
              stroke="#2f343a"
              strokeWidth="8"
            />

            <path
              d="
                M 350 200 L 350 250
                M 350 350 L 350 390
                M 700 200 L 700 370
                M 700 550 L 700 580
                M 450 470 L 600 470
              "
              stroke="#676a6e"
              strokeWidth="14"
              strokeLinecap="round"
              fill="none"
            />
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
                  stroke="#38bdf8"
                  strokeOpacity={0.9}
                  strokeWidth={0.9}
                />
              );
            })}
          </svg>

          {rooms.map((r) => (
            <div
              key={r.id}
              className="absolute"
              style={{
                left: `${(r.labelX ?? r.x + r.w * 0.5) * 100}%`,
                top: `${(r.labelY ?? r.y + r.h * 0.5) * 100}%`,
                transform: "translate(-50%, -50%)",
                color: "#cccfd3",
                fontSize: "13px",
                fontWeight: 600,
                letterSpacing: "0.03em",
                textTransform: "uppercase",
                pointerEvents: "none",
              }}
            >
              {r.title}
            </div>
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

          {devices.map((d) => {
            const pos = positionForDevice(d.id);
            return (
              <div
                key={d.id}
                className={dotClass(d.id)}
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
              >
                <span className="marker-label">{d.id}</span>
              </div>
            );
          })}
        </div>

        <div className="mt-4 rounded-2xl border border-white/10 bg-black/20 px-4 py-3 font-mono text-base text-white/80">
          {chains.length ? chains.map((c) => c.chain.join(" → ")).join(" | ") : "—"}
        </div>
      </div>
    </section>
  );
}
