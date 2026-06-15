import type { FloorPlan, Point, Wall } from '../types';

export const WALL_WIDTH_FALLBACK = 1;

export interface InitialView {
  scale: number;
  x: number;
  y: number;
  minX: number;
  maxX: number;
  minY: number;
  maxY: number;
}

export interface ViewportSize {
  width: number;
  height: number;
}

export const getRenderableWallWidth = (width: number): number =>
  width === 0 ? WALL_WIDTH_FALLBACK : width;

export const getWallThickness = (segPoints: Point[], walls: Wall[]): number => {
  if (segPoints.length < 2 || walls.length === 0) {
    return WALL_WIDTH_FALLBACK;
  }

  const p1 = segPoints[0];
  const p2 = segPoints[1];

  if (!p1 || !p2) {
    return WALL_WIDTH_FALLBACK;
  }

  const mx = (p1[0] + p2[0]) / 2;
  const my = (p1[1] + p2[1]) / 2;

  let minDist = Infinity;
  let matchedWidth = WALL_WIDTH_FALLBACK;

  walls.forEach((wall) => {
    for (let index = 0; index < wall.points.length - 1; index += 1) {
      const start = wall.points[index];
      const end = wall.points[index + 1];

      if (!start || !end) {
        continue;
      }

      const [x1, y1] = start;
      const [x2, y2] = end;
      const a = mx - x1;
      const b = my - y1;
      const c = x2 - x1;
      const d = y2 - y1;
      const dot = a * c + b * d;
      const lenSq = c * c + d * d;
      const param = lenSq === 0 ? -1 : dot / lenSq;

      let xx: number;
      let yy: number;

      if (param < 0) {
        xx = x1;
        yy = y1;
      } else if (param > 1) {
        xx = x2;
        yy = y2;
      } else {
        xx = x1 + param * c;
        yy = y1 + param * d;
      }

      const dx = mx - xx;
      const dy = my - yy;
      const dist = Math.sqrt(dx * dx + dy * dy);

      if (dist < minDist) {
        minDist = dist;
        matchedWidth = getRenderableWallWidth(wall.width);
      }
    }
  });

  return matchedWidth;
};

export const calculateInitialView = (
  plan: FloorPlan,
  viewport: ViewportSize,
  additionalPoints: Point[] = [],
): InitialView => {
  const points = [
    ...plan.walls.flatMap((wall) => wall.points),
    ...plan.doors.flatMap((door) => door.points),
    ...plan.windows.flatMap((windowOpening) => windowOpening.points),
    ...plan.rooms.flatMap((room) => room.area),
    ...additionalPoints,
  ];

  if (points.length === 0) {
    return {
      scale: 1,
      x: viewport.width / 2,
      y: viewport.height / 2,
      minX: 0,
      maxX: 0,
      minY: 0,
      maxY: 0,
    };
  }

  const xs = points.map(([x]) => x);
  const ys = points.map(([, y]) => y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const mapW = Math.max(maxX - minX, 1);
  const mapH = Math.max(maxY - minY, 1);
  const padding = 100;
  const availableWidth = Math.max(viewport.width - padding * 2, 1);
  const availableHeight = Math.max(viewport.height - padding * 2, 1);
  const scaleX = availableWidth / mapW;
  const scaleY = availableHeight / mapH;
  const scale = Math.min(scaleX, scaleY, 1.5);

  return {
    scale,
    x: (viewport.width - mapW * scale) / 2 - minX * scale,
    y: (viewport.height - mapH * scale) / 2 - minY * scale,
    minX,
    maxX,
    minY,
    maxY,
  };
};
