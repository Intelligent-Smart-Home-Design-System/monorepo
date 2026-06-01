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

export type FloorPlanView = {
  walls?: FloorPathLayer;
  doors?: FloorPathLayer;
  windows?: FloorPathLayer;
  furniture?: FloorPolygonLayer;
  blockers?: WallSegment[];
};

export type AdaptedFloor = {
  source: "local" | "parser";
  rooms: Room[];
  floorPlan: FloorPlanView;
  markers: DeviceMarker[];
  warnings: string[];
};

type RawPoint = [number, number];
type ParserWall = { id?: string; points?: RawPoint[]; width?: number };
type ParserDoor = { id?: string; points?: RawPoint[]; width?: number };
type ParserWindow = { id?: string; points?: RawPoint[]; width?: number };
type ParserFurniture = { id?: string; category?: string; points?: RawPoint[] };
type ParserRoom = { id?: string; name?: string; area?: RawPoint[] };
type ParserFloor = {
  schema_version?: string;
  meta?: { source?: string; source_ref?: string; units?: string };
  walls?: ParserWall[];
  doors?: ParserDoor[];
  windows?: ParserWindow[];
  furniture?: ParserFurniture[];
  rooms?: ParserRoom[];
  warnings?: Array<{ code?: string; message?: string }>;
};

type LocalFloor = {
  rooms?: Room[];
  walls?: FloorPathLayer;
  doors?: FloorPathLayer;
  windows?: FloorPathLayer;
  furniture?: FloorPolygonLayer;
  devices?: Array<{ id: string; x: number; y: number; roomId?: string }>;
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

function isParserFloor(value: unknown): value is ParserFloor {
  if (!value || typeof value !== "object") return false;
  const candidate = value as ParserFloor;
  return Array.isArray(candidate.walls) && Array.isArray(candidate.rooms) && Boolean(candidate.schema_version || candidate.meta);
}

function asLocalFloor(value: unknown): LocalFloor {
  return value && typeof value === "object" ? (value as LocalFloor) : {};
}

function collectPoints(floor: ParserFloor) {
  const points: RawPoint[] = [];
  floor.walls?.forEach((item) => item.points?.forEach((point) => isPointTuple(point) && points.push(point)));
  floor.doors?.forEach((item) => item.points?.forEach((point) => isPointTuple(point) && points.push(point)));
  floor.windows?.forEach((item) => item.points?.forEach((point) => isPointTuple(point) && points.push(point)));
  floor.furniture?.forEach((item) => item.points?.forEach((point) => isPointTuple(point) && points.push(point)));
  floor.rooms?.forEach((item) => item.area?.forEach((point) => isPointTuple(point) && points.push(point)));
  return points;
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

function toPath(points: Point[], close = false) {
  if (!points.length) return "";
  const [first, ...rest] = points;
  const start = `M ${Math.round(first.x * VIEWBOX.width)} ${Math.round(first.y * VIEWBOX.height)}`;
  const tail = rest.map((point) => `L ${Math.round(point.x * VIEWBOX.width)} ${Math.round(point.y * VIEWBOX.height)}`).join(" ");
  return `${start}${tail ? ` ${tail}` : ""}${close ? " Z" : ""}`;
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

function roomFromParser(room: ParserRoom, index: number, normalize: (point: RawPoint) => Point): Room | null {
  const area = room.area?.filter(isPointTuple).map(normalize) ?? [];
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
    title: room.name || room.id || `комната ${index + 1}`,
    x: minX,
    y: minY,
    w: Math.max(maxX - minX, 0.04),
    h: Math.max(maxY - minY, 0.04),
    labelX: center.x,
    labelY: center.y,
  };
}

function adaptParserFloor(raw: ParserFloor, fallbackRooms: Room[]): AdaptedFloor {
  const sourcePoints = collectPoints(raw);
  if (!sourcePoints.length) {
    return {
      source: "parser",
      rooms: fallbackRooms,
      markers: [],
      warnings: ["Парсер прислал план без координат, использован локальный план"],
      floorPlan: { blockers: DEFAULT_BLOCKERS },
    };
  }

  const normalize = makeNormalizer(sourcePoints);
  const rooms = raw.rooms?.map((room, index) => roomFromParser(room, index, normalize)).filter((room): room is Room => Boolean(room)) ?? [];
  const walls = raw.walls?.flatMap((wall) => {
    const points = wall.points?.filter(isPointTuple).map(normalize) ?? [];
    return points.length >= 2 ? [toPath(points.slice(0, 2))] : [];
  });
  const doors = raw.doors?.flatMap((door) => {
    const points = door.points?.filter(isPointTuple).map(normalize) ?? [];
    return points.length >= 2 ? [toPath(points.slice(0, 2))] : [];
  });
  const windows = raw.windows?.flatMap((window) => {
    const points = window.points?.filter(isPointTuple).map(normalize) ?? [];
    return points.length >= 2 ? [toPath(points.slice(0, 2))] : [];
  });
  const furniture = raw.furniture?.flatMap((item) => {
    const points = item.points?.filter(isPointTuple).map(normalize) ?? [];
    return points.length >= 3 ? [toPath(points, true)] : [];
  });
  const blockers =
    raw.walls
      ?.map((wall) => wall.points?.filter(isPointTuple).map(normalize) ?? [])
      .map(toBlocker)
      .filter((blocker): blocker is WallSegment => Boolean(blocker)) ?? [];

  return {
    source: "parser",
    rooms: rooms.length ? rooms : fallbackRooms,
    markers: [],
    warnings: raw.warnings?.map((warning) => warning.message || warning.code || "Предупреждение парсера") ?? [],
    floorPlan: {
      walls: { paths: walls, stroke: "#2f343b", strokeWidth: 8, viewBox: VIEWBOX },
      doors: { paths: doors, stroke: "#f5f5f7", strokeWidth: 16, viewBox: VIEWBOX },
      windows: { paths: windows, stroke: "#7cc7ff", strokeWidth: 9, viewBox: VIEWBOX },
      furniture: { paths: furniture, fill: "rgba(142,142,147,0.16)", stroke: "rgba(60,60,67,0.28)", strokeWidth: 2, viewBox: VIEWBOX },
      blockers: blockers.length ? blockers : DEFAULT_BLOCKERS,
    },
  };
}

function adaptLocalFloor(raw: unknown, fallbackRooms: Room[], fallbackMarkers: DeviceMarker[]): AdaptedFloor {
  const floor = asLocalFloor(raw);
  return {
    source: "local",
    rooms: floor.rooms?.length ? floor.rooms : fallbackRooms,
    markers: floor.devices?.length ? floor.devices.map((device) => ({ id: device.id, x: device.x, y: device.y })) : fallbackMarkers,
    warnings: [],
    floorPlan: {
      walls: floor.walls,
      doors: floor.doors,
      windows: floor.windows,
      furniture: floor.furniture,
      blockers: DEFAULT_BLOCKERS,
    },
  };
}

export function adaptFloorData(raw: unknown, fallbackRooms: Room[], fallbackMarkers: DeviceMarker[]): AdaptedFloor {
  if (isParserFloor(raw)) return adaptParserFloor(raw, fallbackRooms);
  return adaptLocalFloor(raw, fallbackRooms, fallbackMarkers);
}
