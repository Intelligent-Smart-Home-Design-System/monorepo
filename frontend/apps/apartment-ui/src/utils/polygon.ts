import type { LayoutPoint, Point } from '../types';

const EPSILON = 1e-9;

export const layoutPointToPoint = (point: LayoutPoint): Point => [point.X, point.Y];

const isSamePoint = (first: Point, second: Point): boolean =>
  first[0] === second[0] && first[1] === second[1];

export const getOpenPolygonPoints = (points: Point[]): Point[] => {
  if (points.length < 2) {
    return points;
  }

  const first = points[0];
  const last = points[points.length - 1];

  if (!first || !last) {
    return points;
  }

  return isSamePoint(first, last) ? points.slice(0, -1) : points;
};

export const flattenPolygonPoints = (points: Point[]): number[] =>
  getOpenPolygonPoints(points).flat();

const getAveragePoint = (points: Point[]): Point => {
  if (points.length === 0) {
    return [0, 0];
  }

  const totals = points.reduce(
    (acc, [x, y]) => ({ x: acc.x + x, y: acc.y + y }),
    { x: 0, y: 0 },
  );

  return [totals.x / points.length, totals.y / points.length];
};

export const getPolygonCentroid = (points: Point[]): Point => {
  const polygon = getOpenPolygonPoints(points);

  if (polygon.length < 3) {
    return getAveragePoint(polygon);
  }

  let twiceArea = 0;
  let centerX = 0;
  let centerY = 0;

  polygon.forEach((point, index) => {
    const nextPoint = polygon[(index + 1) % polygon.length];

    if (!nextPoint) {
      return;
    }

    const cross = point[0] * nextPoint[1] - nextPoint[0] * point[1];
    twiceArea += cross;
    centerX += (point[0] + nextPoint[0]) * cross;
    centerY += (point[1] + nextPoint[1]) * cross;
  });

  if (Math.abs(twiceArea) < EPSILON) {
    return getAveragePoint(polygon);
  }

  return [centerX / (3 * twiceArea), centerY / (3 * twiceArea)];
};

const isPointOnSegment = (point: Point, start: Point, end: Point): boolean => {
  const cross =
    (point[1] - start[1]) * (end[0] - start[0]) -
    (point[0] - start[0]) * (end[1] - start[1]);

  if (Math.abs(cross) > EPSILON) {
    return false;
  }

  const dot =
    (point[0] - start[0]) * (end[0] - start[0]) +
    (point[1] - start[1]) * (end[1] - start[1]);

  if (dot < -EPSILON) {
    return false;
  }

  const squaredLength =
    (end[0] - start[0]) * (end[0] - start[0]) +
    (end[1] - start[1]) * (end[1] - start[1]);

  return dot <= squaredLength + EPSILON;
};

const getSegmentDirection = (start: Point, end: Point, point: Point): number =>
  (point[0] - start[0]) * (end[1] - start[1]) -
  (point[1] - start[1]) * (end[0] - start[0]);

const getDirectionSign = (value: number): -1 | 0 | 1 => {
  if (value > EPSILON) {
    return 1;
  }

  if (value < -EPSILON) {
    return -1;
  }

  return 0;
};

const doSegmentsProperlyIntersect = (
  firstStart: Point,
  firstEnd: Point,
  secondStart: Point,
  secondEnd: Point,
): boolean => {
  const firstDirectionStart = getDirectionSign(
    getSegmentDirection(firstStart, firstEnd, secondStart),
  );
  const firstDirectionEnd = getDirectionSign(
    getSegmentDirection(firstStart, firstEnd, secondEnd),
  );
  const secondDirectionStart = getDirectionSign(
    getSegmentDirection(secondStart, secondEnd, firstStart),
  );
  const secondDirectionEnd = getDirectionSign(
    getSegmentDirection(secondStart, secondEnd, firstEnd),
  );

  return (
    firstDirectionStart * firstDirectionEnd < 0 &&
    secondDirectionStart * secondDirectionEnd < 0
  );
};

const getPolygonEdges = (points: Point[]): Array<[Point, Point]> =>
  points.map((point, index) => [point, points[(index + 1) % points.length] ?? point]);

const getMidPoint = (first: Point, second: Point): Point => [
  (first[0] + second[0]) / 2,
  (first[1] + second[1]) / 2,
];

export const isPointInPolygon = (point: Point, points: Point[]): boolean => {
  const polygon = getOpenPolygonPoints(points);

  if (polygon.length < 3) {
    return false;
  }

  let isInside = false;
  let previousIndex = polygon.length - 1;

  for (let currentIndex = 0; currentIndex < polygon.length; currentIndex += 1) {
    const current = polygon[currentIndex];
    const previous = polygon[previousIndex];

    if (!current || !previous) {
      previousIndex = currentIndex;
      continue;
    }

    if (isPointOnSegment(point, current, previous)) {
      return true;
    }

    if ((current[1] > point[1]) !== (previous[1] > point[1])) {
      const intersectX =
        ((previous[0] - current[0]) * (point[1] - current[1])) /
          (previous[1] - current[1]) +
        current[0];

      if (point[0] < intersectX) {
        isInside = !isInside;
      }
    }

    previousIndex = currentIndex;
  }

  return isInside;
};

export const isPolygonInsidePolygon = (
  innerPoints: Point[],
  outerPoints: Point[],
): boolean => {
  const innerPolygon = getOpenPolygonPoints(innerPoints);
  const outerPolygon = getOpenPolygonPoints(outerPoints);

  if (innerPolygon.length < 3 || outerPolygon.length < 3) {
    return false;
  }

  if (!innerPolygon.every((point) => isPointInPolygon(point, outerPolygon))) {
    return false;
  }

  const outerEdges = getPolygonEdges(outerPolygon);

  return getPolygonEdges(innerPolygon).every(([innerStart, innerEnd]) => {
    if (!isPointInPolygon(getMidPoint(innerStart, innerEnd), outerPolygon)) {
      return false;
    }

    return outerEdges.every(
      ([outerStart, outerEnd]) =>
        !doSegmentsProperlyIntersect(innerStart, innerEnd, outerStart, outerEnd),
    );
  });
};
