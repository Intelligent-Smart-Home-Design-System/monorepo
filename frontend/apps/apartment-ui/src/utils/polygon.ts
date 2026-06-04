import type { LayoutPoint, Point } from '../types';

const EPSILON = 1e-9;
const BOUNDARY_DISTANCE_TOLERANCE = 0.5;
const CONSTRAINT_SEARCH_STEPS = 32;

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

const getSquaredDistance = (first: Point, second: Point): number =>
  (first[0] - second[0]) * (first[0] - second[0]) +
  (first[1] - second[1]) * (first[1] - second[1]);

const getClosestPointOnSegment = (
  point: Point,
  start: Point,
  end: Point,
): Point => {
  const segmentX = end[0] - start[0];
  const segmentY = end[1] - start[1];
  const squaredLength = segmentX * segmentX + segmentY * segmentY;

  if (squaredLength < EPSILON) {
    return start;
  }

  const projection =
    ((point[0] - start[0]) * segmentX + (point[1] - start[1]) * segmentY) /
    squaredLength;
  const clampedProjection = Math.max(0, Math.min(1, projection));

  return [
    start[0] + clampedProjection * segmentX,
    start[1] + clampedProjection * segmentY,
  ];
};

const lerpPoint = (start: Point, end: Point, amount: number): Point => [
  start[0] + (end[0] - start[0]) * amount,
  start[1] + (end[1] - start[1]) * amount,
];

const getBoundaryEdgeIndexesForPoint = (
  point: Point,
  points: Point[],
): number[] => {
  const polygon = getOpenPolygonPoints(points);
  const toleranceSquared =
    BOUNDARY_DISTANCE_TOLERANCE * BOUNDARY_DISTANCE_TOLERANCE;

  return getPolygonEdges(polygon)
    .map(([start, end], index) => ({
      index,
      distance: getSquaredDistance(
        point,
        getClosestPointOnSegment(point, start, end),
      ),
    }))
    .filter((edge) => edge.distance <= toleranceSquared)
    .sort((first, second) => first.distance - second.distance)
    .map((edge) => edge.index);
};

const getProjectedPointsOnEdges = (
  point: Point,
  points: Point[],
  edgeIndexes: number[],
): Point[] => {
  const polygon = getOpenPolygonPoints(points);
  const edges = getPolygonEdges(polygon);

  return edgeIndexes
    .map((edgeIndex) => {
      const edge = edges[edgeIndex];

      if (!edge) {
        return null;
      }
      const projectedPoint = getClosestPointOnSegment(point, edge[0], edge[1]);

      return {
        point: projectedPoint,
        distance: getSquaredDistance(point, projectedPoint),
      };
    })
    .filter((candidate): candidate is { point: Point; distance: number } =>
      Boolean(candidate),
    )
    .sort((first, second) => first.distance - second.distance)
    .map((candidate) => candidate.point);
};

const getFarthestAllowedPointOnSegment = (
  start: Point,
  end: Point,
  isAllowed: (point: Point) => boolean,
): Point => {
  if (!isAllowed(start)) {
    return start;
  }

  let low = 0;
  let high = 1;
  let best = start;

  for (let step = 0; step < CONSTRAINT_SEARCH_STEPS; step += 1) {
    const middle = (low + high) / 2;
    const candidate = lerpPoint(start, end, middle);

    if (isAllowed(candidate)) {
      best = candidate;
      low = middle;
    } else {
      high = middle;
    }
  }

  return best;
};

export const getClosestPointsOnPolygonBoundary = (
  point: Point,
  points: Point[],
): Point[] => {
  const polygon = getOpenPolygonPoints(points);

  if (polygon.length < 2) {
    return [];
  }

  return getPolygonEdges(polygon)
    .map(([start, end]) => {
      const candidate = getClosestPointOnSegment(point, start, end);

      return {
        point: candidate,
        distance: getSquaredDistance(point, candidate),
      };
    })
    .sort((first, second) => first.distance - second.distance)
    .map((candidate) => candidate.point);
};

export const getClosestPointOnPolygonBoundary = (
  point: Point,
  points: Point[],
): Point | null => getClosestPointsOnPolygonBoundary(point, points)[0] ?? null;

export const getConstrainedPointOnDragPath = (
  currentPoint: Point,
  desiredPoint: Point,
  boundaryPoints: Point[],
  isAllowed: (point: Point) => boolean,
): Point => {
  if (isAllowed(desiredPoint)) {
    return desiredPoint;
  }

  const boundaryEdgeIndexes = getBoundaryEdgeIndexesForPoint(
    currentPoint,
    boundaryPoints,
  );
  const boundaryCandidate = getProjectedPointsOnEdges(
    desiredPoint,
    boundaryPoints,
    boundaryEdgeIndexes,
  ).find(isAllowed);

  if (boundaryCandidate) {
    return boundaryCandidate;
  }

  return getFarthestAllowedPointOnSegment(currentPoint, desiredPoint, isAllowed);
};

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
