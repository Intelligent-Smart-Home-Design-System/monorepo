import type { LayoutPoint, Room, Zone } from '../types';
import { isPolygonInsidePolygon, layoutPointToPoint } from './polygon';

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

      if (!isZonePointMoveAllowed(zone, pointIndex, point, roomsById)) {
        return zone;
      }
    }

    return getZoneWithUpdatedPoint(zone, pointIndex, point);
  });
