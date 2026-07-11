import type { DeviceMarker, Room } from "@/app/simulation/Mockdata";

export type Point = { x: number; y: number };

export type WallSegment =
  | { kind: "vertical"; x: number; y1: number; y2: number }
  | { kind: "horizontal"; y: number; x1: number; x2: number }
  | { kind: "segment"; from: Point; to: Point };

export type FloorPathLayer = {
  paths?: string[];
  stroke?: string;
  strokeWidth?: number;
  viewBox?: { width?: number; height?: number };
};

export type FloorPolygonLayer = {
  paths?: string[];
  fill?: string;
  stroke?: string;
  strokeWidth?: number;
  viewBox?: { width?: number; height?: number };
};

export type FloorZoneView = {
  id: string;
  roomId?: string;
  kind?: string;
  path: string;
  label?: Point;
  bounds?: { x: number; y: number; w: number; h: number };
};

export type FloorPlanView = {
  walls?: FloorPathLayer;
  doors?: FloorPathLayer;
  windows?: FloorPathLayer;
  furniture?: FloorPolygonLayer;
  zones?: FloorZoneView[];
  blockers?: WallSegment[];
};

export type AdaptedFloor = {
  source: "local" | "parser";
  rooms: Room[];
  floorPlan: FloorPlanView;
  markers: DeviceMarker[];
  placementMarkers: DeviceMarker[];
  warnings: string[];
};

type RawPoint = [number, number];
type RawPointLike = RawPoint | { x?: number; y?: number; X?: number; Y?: number };
type ParserWall = { id?: string; points?: RawPointLike[]; width?: number };
type ParserDoor = { id?: string; points?: RawPointLike[]; width?: number };
type ParserWindow = { id?: string; points?: RawPointLike[]; width?: number };
type ParserFurniture = { id?: string; category?: string; points?: RawPointLike[] };
type ParserRoom = { id?: string; name?: string; title?: string; area?: RawPointLike[]; x?: number; y?: number; w?: number; h?: number };
type ParserZone = {
  id?: string;
  room_id?: string;
  roomId?: string;
  type?: string;
  kind?: string;
  category?: string;
  name?: string;
  points?: RawPointLike[];
};
type ParserFloor = {
  schema_version?: string;
  meta?: { source?: string; source_ref?: string; units?: string };
  walls?: ParserWall[];
  doors?: ParserDoor[];
  windows?: ParserWindow[];
  furniture?: ParserFurniture[];
  rooms?: ParserRoom[];
  zones?: ParserZone[];
  warnings?: Array<{ code?: string; message?: string }>;
};

type LocalFloor = {
  rooms?: Room[];
  walls?: FloorPathLayer;
  doors?: FloorPathLayer;
  windows?: FloorPathLayer;
  furniture?: FloorPolygonLayer;
  zones?: ParserZone[];
  blockers?: WallSegment[];
  devices?: Array<{ id: string; x: number; y: number; roomId?: string }>;
};

type LayoutPlacement = {
  device?: { id?: string; type?: string };
  position?: { x?: number; y?: number };
};

type LayoutPayload = {
  placements?: Record<string, LayoutPlacement[]>;
};

const VIEWBOX = { width: 1000, height: 700 };
const PADDING = 0.04;

const DEFAULT_BLOCKERS: WallSegment[] = [
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

function isPointTuple(value: unknown): value is RawPoint {
  return (
    Array.isArray(value) &&
    value.length >= 2 &&
    typeof value[0] === "number" &&
    Number.isFinite(value[0]) &&
    typeof value[1] === "number" &&
    Number.isFinite(value[1])
  );
}

function toRawPoint(value: unknown): RawPoint | null {
  if (isPointTuple(value)) return [value[0], value[1]];
  if (!value || typeof value !== "object") return null;

  const record = value as Record<string, unknown>;
  const x = typeof record.x === "number" ? record.x : typeof record.X === "number" ? record.X : undefined;
  const y = typeof record.y === "number" ? record.y : typeof record.Y === "number" ? record.Y : undefined;
  return typeof x === "number" && Number.isFinite(x) && typeof y === "number" && Number.isFinite(y) ? [x, y] : null;
}

function normalizeRawPoints(points: RawPointLike[] | undefined) {
  return points?.map(toRawPoint).filter((point): point is RawPoint => Boolean(point)) ?? [];
}

function isParserFloor(value: unknown): value is ParserFloor {
  if (!value || typeof value !== "object") return false;
  const candidate = value as ParserFloor;
  return Array.isArray(candidate.walls) && Boolean(candidate.schema_version || candidate.meta || Array.isArray(candidate.rooms));
}

function asLocalFloor(value: unknown): LocalFloor {
  return value && typeof value === "object" ? (value as LocalFloor) : {};
}

function unwrapFloorPayload(value: unknown): unknown {
  if (!value || typeof value !== "object") return value;
  const record = value as Record<string, unknown>;
  if (record.floor || record.floorJson || record.parsedFloor || record.apartment) {
    const floor = unwrapFloorPayload(record.floor ?? record.floorJson ?? record.parsedFloor ?? record.apartment);
    if (floor && typeof floor === "object") {
      const floorRecord = floor as Record<string, unknown>;
      return {
        ...floorRecord,
        zones: floorRecord.zones ?? record.zones,
        layout: floorRecord.layout ?? record.layout,
      };
    }
    return floor;
  }

  for (const key of ["floor", "floorJson", "parsedFloor", "apartment"]) {
    if (record[key]) return unwrapFloorPayload(record[key]);
  }
  return value;
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" ? (value as Record<string, unknown>) : null;
}

function normalizeZoneLike(value: unknown, fallbackKind?: string, roomId?: string): ParserZone | null {
  const record = asRecord(value);
  if (!record) return null;

  const points = Array.isArray(record.points) ? (record.points as RawPointLike[]) : undefined;
  if (!points?.length) return null;

  return {
    id: typeof record.id === "string" ? record.id : undefined,
    room_id: typeof record.room_id === "string" ? record.room_id : roomId,
    roomId: typeof record.roomId === "string" ? record.roomId : roomId,
    type: typeof record.type === "string" ? record.type : fallbackKind,
    kind: typeof record.kind === "string" ? record.kind : fallbackKind,
    category: typeof record.category === "string" ? record.category : fallbackKind,
    name: typeof record.name === "string" ? record.name : fallbackKind,
    points,
  };
}

function collectZonesFromPayload(value: unknown): ParserZone[] {
  const record = asRecord(value);
  if (!record) return [];

  const zones: ParserZone[] = [];
  const zoneKeys = [
    "zones",
    "no_wind_zones",
    "wet_zones",
    "gas_zones",
    "entry_doors_zones",
    "high_traffic_zones",
    "window_zones",
    "viewed_zones",
    "siren_zones",
    "pollution_zones",
    "cleaning_zones",
    "restricted_zones",
  ];

  zoneKeys.forEach((key) => {
    const valueForKey = record[key];
    const values = Array.isArray(valueForKey) ? valueForKey : valueForKey ? [valueForKey] : [];
    values.forEach((item) => {
      const zone = normalizeZoneLike(item, key);
      if (zone) zones.push(zone);
    });
  });

  const roomLists = [
    ...(Array.isArray(record.zoned_rooms) ? record.zoned_rooms : []),
    ...(Array.isArray(record.ZonedRooms) ? record.ZonedRooms : []),
    ...(Array.isArray(record.rooms) ? record.rooms : []),
  ];

  roomLists.forEach((room) => {
    const roomRecord = asRecord(room);
    if (!roomRecord) return;
    const rawRoom = asRecord(roomRecord.orig_room) ?? asRecord(roomRecord.OrigRoom) ?? roomRecord;
    const roomId =
      (typeof rawRoom?.id === "string" && rawRoom.id) ||
      (typeof roomRecord.room_id === "string" && roomRecord.room_id) ||
      (typeof roomRecord.roomId === "string" && roomRecord.roomId) ||
      undefined;

    zoneKeys
      .filter((key) => key !== "zones")
      .forEach((key) => {
        const valueForKey = roomRecord[key];
        const values = Array.isArray(valueForKey) ? valueForKey : valueForKey ? [valueForKey] : [];
        values.forEach((item) => {
          const zone = normalizeZoneLike(item, key, roomId);
          if (zone) zones.push(zone);
        });
      });
  });

  const seen = new Set<string>();
  return zones.filter((zone, index) => {
    const key = `${zone.kind ?? zone.type ?? zone.category ?? "zone"}:${zone.room_id ?? zone.roomId ?? ""}:${normalizeRawPoints(zone.points)
      .map(([x, y]) => `${x}:${y}`)
      .join("|")}`;
    if (seen.has(key)) return false;
    seen.add(key || `zone-${index}`);
    return true;
  });
}

function collectPoints(floor: ParserFloor) {
  const points: RawPoint[] = [];
  floor.walls?.forEach((item) => points.push(...normalizeRawPoints(item.points)));
  floor.doors?.forEach((item) => points.push(...normalizeRawPoints(item.points)));
  floor.windows?.forEach((item) => points.push(...normalizeRawPoints(item.points)));
  floor.furniture?.forEach((item) => points.push(...normalizeRawPoints(item.points)));
  floor.rooms?.forEach((item) => {
    points.push(...normalizeRawPoints(item.area));
    if (typeof item.x === "number" && typeof item.y === "number") points.push([item.x, item.y]);
    if (typeof item.x === "number" && typeof item.y === "number" && typeof item.w === "number" && typeof item.h === "number") {
      points.push([item.x + item.w, item.y + item.h]);
    }
  });
  collectZonesFromPayload(floor).forEach((item) => points.push(...normalizeRawPoints(item.points)));
  return points;
}

function collectLayoutPoints(raw: unknown) {
  const layout = extractLayout(raw);
  if (!layout?.placements) return [];

  return Object.values(layout.placements).flatMap((placements) =>
    placements.flatMap((placement) => {
      const x = placement.position?.x;
      const y = placement.position?.y;
      return typeof x === "number" && Number.isFinite(x) && typeof y === "number" && Number.isFinite(y) ? ([[x, y]] as RawPoint[]) : [];
    })
  );
}

function boundsFor(points: RawPoint[]) {
  const xs = points.map(([x]) => x);
  const ys = points.map(([, y]) => y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const width = Math.max(maxX - minX, 1);
  const height = Math.max(maxY - minY, 1);
  return { minX, minY, width, height };
}

function makeNormalizer(points: RawPoint[]) {
  const bounds = boundsFor(points);
  const scale = 1 - PADDING * 2;
  return ([x, y]: RawPoint): Point => ({
    x: PADDING + ((x - bounds.minX) / bounds.width) * scale,
    y: PADDING + ((y - bounds.minY) / bounds.height) * scale,
  });
}

function validateScale(points: RawPoint[], source: "local" | "parser") {
  if (!points.length) return [];
  const bounds = boundsFor(points);
  const warnings: string[] = [];
  const ratio = bounds.width / bounds.height;

  if (bounds.width < 1 || bounds.height < 1) {
    warnings.push("План имеет слишком маленький масштаб координат, проверь входной floor.json");
  }

  if (ratio > 4 || ratio < 0.25) {
    warnings.push("План выглядит слишком вытянутым, возможна ошибка масштаба или единиц измерения");
  }

  if (source === "parser" && points.length < 4) {
    warnings.push("В плане недостаточно стен для надежной проверки проходов");
  }

  return warnings;
}

function toPath(points: Point[], close = false) {
  if (!points.length) return "";
  const [first, ...rest] = points;
  const start = `M ${Math.round(first.x * VIEWBOX.width)} ${Math.round(first.y * VIEWBOX.height)}`;
  const tail = rest.map((point) => `L ${Math.round(point.x * VIEWBOX.width)} ${Math.round(point.y * VIEWBOX.height)}`).join(" ");
  return `${start}${tail ? ` ${tail}` : ""}${close ? " Z" : ""}`;
}

function toSegmentPaths(points: Point[], close = false) {
  if (points.length < 2) return [];
  if (close) return [toPath(points, true)];
  return points.slice(0, -1).flatMap((point, index) => {
    const next = points[index + 1];
    return next ? [toPath([point, next])] : [];
  });
}

function toBlocker(points: Point[]): WallSegment | null {
  if (points.length < 2) return null;
  const [a, b] = points;
  const dx = Math.abs(a.x - b.x);
  const dy = Math.abs(a.y - b.y);
  const tolerance = 0.012;

  if (dx <= tolerance) {
    return { kind: "vertical", x: (a.x + b.x) / 2, y1: Math.min(a.y, b.y), y2: Math.max(a.y, b.y) };
  }

  if (dy <= tolerance) {
    return { kind: "horizontal", y: (a.y + b.y) / 2, x1: Math.min(a.x, b.x), x2: Math.max(a.x, b.x) };
  }

  return { kind: "segment", from: a, to: b };
}

function toBlockers(points: Point[]) {
  return points.slice(0, -1).flatMap((point, index) => {
    const next = points[index + 1];
    const blocker = next ? toBlocker([point, next]) : null;
    return blocker ? [blocker] : [];
  });
}

function distance(a: Point | undefined, b: Point | undefined) {
  if (!a || !b) return Infinity;
  return Math.hypot(a.x - b.x, a.y - b.y);
}

function extractLayout(raw: unknown): LayoutPayload | null {
  if (!raw || typeof raw !== "object") return null;
  const record = raw as Record<string, unknown>;
  const layout = (record.layout ?? raw) as LayoutPayload;
  return layout && typeof layout === "object" && layout.placements ? layout : null;
}

function layoutMarkers(raw: unknown, normalize?: (point: RawPoint) => Point): DeviceMarker[] {
  const layout = extractLayout(raw);
  if (!layout?.placements) return [];

  return Object.entries(layout.placements).flatMap(([roomId, placements]) =>
    placements.flatMap((placement, index) => {
      const x = placement.position?.x;
      const y = placement.position?.y;
      if (typeof x !== "number" || !Number.isFinite(x) || typeof y !== "number" || !Number.isFinite(y)) return [];

      const point = normalize ? normalize([x, y]) : { x, y };
      if (point.x < 0 || point.x > 1 || point.y < 0 || point.y > 1) return [];

      const type = placement.device?.type || "device";
      const id = placement.device?.id || `${type}_${roomId}_${index + 1}`;
      return [{ id, x: point.x, y: point.y, label: type }];
    })
  );
}

function roomFromParser(room: ParserRoom, index: number, normalize: (point: RawPoint) => Point): Room | null {
  if (
    typeof room.x === "number" &&
    typeof room.y === "number" &&
    typeof room.w === "number" &&
    typeof room.h === "number" &&
    room.x >= 0 &&
    room.x <= 1 &&
    room.y >= 0 &&
    room.y <= 1
  ) {
    return {
      id: room.id || `room_${index + 1}`,
      title: room.title || room.name || room.id || `комната ${index + 1}`,
      x: room.x,
      y: room.y,
      w: Math.max(room.w, 0.04),
      h: Math.max(room.h, 0.04),
      labelX: room.x + room.w / 2,
      labelY: room.y + room.h / 2,
    };
  }

  const area = normalizeRawPoints(room.area).map(normalize);
  if (!area.length) return null;

  const minX = Math.min(...area.map((point) => point.x));
  const maxX = Math.max(...area.map((point) => point.x));
  const minY = Math.min(...area.map((point) => point.y));
  const maxY = Math.max(...area.map((point) => point.y));
  const center = area.reduce(
    (acc, point) => ({ x: acc.x + point.x / area.length, y: acc.y + point.y / area.length }),
    { x: 0, y: 0 }
  );

  return {
    id: room.id || `room_${index + 1}`,
    title: room.title || room.name || room.id || `комната ${index + 1}`,
    x: minX,
    y: minY,
    w: Math.max(maxX - minX, 0.04),
    h: Math.max(maxY - minY, 0.04),
    labelX: center.x,
    labelY: center.y,
  };
}

function zoneFromParser(zone: ParserZone, index: number, normalize: (point: RawPoint) => Point): FloorZoneView | null {
  const points = normalizeRawPoints(zone.points).map(normalize);
  if (points.length < 3) return null;
  const center = points.reduce(
    (acc, point) => ({ x: acc.x + point.x / points.length, y: acc.y + point.y / points.length }),
    { x: 0, y: 0 }
  );
  const minX = Math.min(...points.map((point) => point.x));
  const maxX = Math.max(...points.map((point) => point.x));
  const minY = Math.min(...points.map((point) => point.y));
  const maxY = Math.max(...points.map((point) => point.y));
  const kind = zone.kind ?? zone.type ?? zone.category ?? zone.name;

  return {
    id: zone.id || `zone_${index + 1}`,
    roomId: zone.room_id ?? zone.roomId,
    kind,
    path: toPath(points, true),
    label: center,
    bounds: { x: minX, y: minY, w: Math.max(maxX - minX, 0), h: Math.max(maxY - minY, 0) },
  };
}

function adaptParserFloor(raw: ParserFloor, fallbackRooms: Room[]): AdaptedFloor {
  const sourcePoints = [...collectPoints(raw), ...collectLayoutPoints(raw)];
  if (!sourcePoints.length) {
    return {
      source: "parser",
      rooms: fallbackRooms,
      markers: [],
      placementMarkers: [],
      warnings: ["Парсер прислал план без координат, использован локальный план"],
      floorPlan: { blockers: DEFAULT_BLOCKERS },
    };
  }

  const normalize = makeNormalizer(sourcePoints);
  const scaleWarnings = validateScale(sourcePoints, "parser");
  const rooms = raw.rooms?.map((room, index) => roomFromParser(room, index, normalize)).filter((room): room is Room => Boolean(room)) ?? [];
  const walls = raw.walls?.flatMap((wall) => {
    const points = normalizeRawPoints(wall.points).map(normalize);
    return toSegmentPaths(points, points.length > 2 && distance(points[0], points[points.length - 1]) < 0.002);
  });
  const doors = raw.doors?.flatMap((door) => {
    const points = normalizeRawPoints(door.points).map(normalize);
    return points.length >= 2 ? [toPath(points.slice(0, 2))] : [];
  });
  const windows = raw.windows?.flatMap((window) => {
    const points = normalizeRawPoints(window.points).map(normalize);
    return points.length >= 2 ? [toPath(points.slice(0, 2))] : [];
  });
  const furniture = raw.furniture?.flatMap((item) => {
    const points = normalizeRawPoints(item.points).map(normalize);
    return points.length >= 3 ? [toPath(points, true)] : [];
  });
  const zones = collectZonesFromPayload(raw).map((zone, index) => zoneFromParser(zone, index, normalize)).filter((zone): zone is FloorZoneView => Boolean(zone));
  const blockers =
    raw.walls
      ?.flatMap((wall) => toBlockers(normalizeRawPoints(wall.points).map(normalize))) ?? [];
  const placementMarkers = layoutMarkers(raw, normalize);

  return {
    source: "parser",
    rooms: rooms.length ? rooms : fallbackRooms,
    markers: placementMarkers,
    placementMarkers,
    warnings: [
      ...(raw.warnings?.map((warning) => warning.message || warning.code || "Предупреждение парсера") ?? []),
      ...scaleWarnings,
    ],
    floorPlan: {
      walls: { paths: walls, stroke: "#2f343b", strokeWidth: 8, viewBox: VIEWBOX },
      doors: { paths: doors, stroke: "#f5f5f7", strokeWidth: 16, viewBox: VIEWBOX },
      windows: { paths: windows, stroke: "#7cc7ff", strokeWidth: 9, viewBox: VIEWBOX },
      furniture: { paths: furniture, fill: "rgba(142,142,147,0.16)", stroke: "rgba(60,60,67,0.28)", strokeWidth: 2, viewBox: VIEWBOX },
      zones,
      blockers: blockers.length ? blockers : DEFAULT_BLOCKERS,
    },
  };
}

function adaptLocalFloor(raw: unknown, fallbackRooms: Room[], fallbackMarkers: DeviceMarker[]): AdaptedFloor {
  const floor = asLocalFloor(raw);
  const markersFromLayout = layoutMarkers(raw);
  const warnings = validateScale(
    [
      ...(floor.rooms?.flatMap((room) => [
        [room.x, room.y],
        [room.x + room.w, room.y + room.h],
      ]) as RawPoint[] | undefined ?? []),
      ...collectLayoutPoints(raw),
    ],
    "local"
  );

  return {
    source: "local",
    rooms: floor.rooms?.length ? floor.rooms : fallbackRooms,
    markers: floor.devices?.length
      ? floor.devices.map((device) => ({ id: device.id, x: device.x, y: device.y }))
      : markersFromLayout.length
      ? markersFromLayout
      : fallbackMarkers,
    placementMarkers: markersFromLayout,
    warnings,
    floorPlan: {
      walls: floor.walls,
      doors: floor.doors,
      windows: floor.windows,
      furniture: floor.furniture,
      zones: collectZonesFromPayload(floor).map((zone, index) => zoneFromParser(zone, index, ([x, y]) => ({ x, y }))).filter((zone): zone is FloorZoneView => Boolean(zone)),
      blockers: floor.blockers?.length ? floor.blockers : DEFAULT_BLOCKERS,
    },
  };
}

export function adaptFloorData(raw: unknown, fallbackRooms: Room[], fallbackMarkers: DeviceMarker[]): AdaptedFloor {
  const payload = unwrapFloorPayload(raw);
  if (isParserFloor(payload)) return adaptParserFloor(payload, fallbackRooms);
  return adaptLocalFloor(payload, fallbackRooms, fallbackMarkers);
}
