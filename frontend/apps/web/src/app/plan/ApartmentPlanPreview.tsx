"use client";

import { useEffect, useMemo, useRef } from "react";
import type { ApartmentPlanRenderHandle, FloorPlan, Zone } from "smart-plan-demo";
import { renderApartmentPlan } from "smart-plan-demo";

type ApartmentPlanPreviewProps = {
  floor: unknown;
  zones?: unknown;
};

export function ApartmentPlanPreview({ floor, zones }: ApartmentPlanPreviewProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const handleRef = useRef<ApartmentPlanRenderHandle | null>(null);
  const plan = useMemo(() => normalizeFloorPlan(floor), [floor]);
  const normalizedZones = useMemo(() => normalizeZones(zones), [zones]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !plan) return;

    if (!handleRef.current) {
      handleRef.current = renderApartmentPlan(container, plan, [], normalizedZones);
      return;
    }

    handleRef.current.update(plan, [], normalizedZones);
  }, [normalizedZones, plan]);

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
