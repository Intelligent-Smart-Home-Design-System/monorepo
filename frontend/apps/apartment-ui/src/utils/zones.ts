import type { LayoutPoint, Room, Zone } from '../types';
import {
  getConstrainedPointOnDragPath,
  isPolygonInsidePolygon,
  layoutPointToPoint,
} from './polygon';

export const getZoneRoomId = (zone: Zone): string | undefined =>
  zone.room_id ?? zone.roomId;

const getZoneWithUpdatedPoint = (
  zone: Zone,
  pointIndex: number,
  point: LayoutPoint,
): Zone => ({
  ...zone,
  points: zone.points.map((currentPoint, currentIndex) =>
    currentIndex === pointIndex ? point : currentPoint,
  ),
});

export const isZonePointMoveAllowed = (
  zone: Zone,
  pointIndex: number,
  point: LayoutPoint,
  roomsById: ReadonlyMap<string, Room>,
): boolean => {
  const roomId = getZoneRoomId(zone);

  if (!roomId) {
    return true;
  }

  const room = roomsById.get(roomId);

  if (!room) {
    return false;
  }

  const nextZone = getZoneWithUpdatedPoint(zone, pointIndex, point);

  return isPolygonInsidePolygon(
    nextZone.points.map(layoutPointToPoint),
    room.area,
  );
};

export const getConstrainedZonePoint = (
  zone: Zone,
  pointIndex: number,
  point: LayoutPoint,
  roomsById: ReadonlyMap<string, Room>,
  currentPoint = zone.points[pointIndex],
): LayoutPoint => {
  if (isZonePointMoveAllowed(zone, pointIndex, point, roomsById)) {
    return point;
  }

  const roomId = getZoneRoomId(zone);
  const room = roomId ? roomsById.get(roomId) : undefined;
  if (!room || !currentPoint) {
    return currentPoint ?? point;
  }

  const constrainedPoint = getConstrainedPointOnDragPath(
    layoutPointToPoint(currentPoint),
    layoutPointToPoint(point),
    room.area,
    (candidate) =>
      isZonePointMoveAllowed(
        zone,
        pointIndex,
        { X: candidate[0], Y: candidate[1] },
        roomsById,
      ),
  );

  return {
    X: constrainedPoint[0],
    Y: constrainedPoint[1],
  };
};

export const updateZonePoint = (
  zones: Zone[],
  zoneId: string,
  pointIndex: number,
  point: LayoutPoint,
  rooms?: Room[],
): Zone[] =>
  zones.map((zone) => {
    if (zone.id !== zoneId) {
      return zone;
    }

    if (rooms) {
      const roomsById = new Map(rooms.map((room) => [room.id, room]));
      return getZoneWithUpdatedPoint(
        zone,
        pointIndex,
        getConstrainedZonePoint(zone, pointIndex, point, roomsById),
      );
    }

    return getZoneWithUpdatedPoint(zone, pointIndex, point);
  });
