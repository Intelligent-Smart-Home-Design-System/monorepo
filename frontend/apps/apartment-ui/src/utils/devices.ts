import type { Room, SmartDevice } from '../types';
import { getConstrainedPointOnDragPath, isPointInPolygon } from './polygon';

export const addSmartDevice = (
  devices: SmartDevice[],
  device: SmartDevice,
): SmartDevice[] => [...devices, device];

export const removeSmartDevice = (
  devices: SmartDevice[],
  deviceId: string,
): SmartDevice[] => devices.filter((device) => device.id !== deviceId);

export const clearSmartDevices = (): SmartDevice[] => [];

export const isSmartDevicePositionAllowed = (
  device: SmartDevice,
  position: SmartDevice['position'],
  roomsById: ReadonlyMap<string, Room>,
): boolean => {
  const room = roomsById.get(device.room_id);

  if (!room) {
    return true;
  }

  return isPointInPolygon([position.x, position.y], room.area);
};

export const getConstrainedSmartDevicePosition = (
  device: SmartDevice,
  position: SmartDevice['position'],
  currentPosition: SmartDevice['position'],
  roomsById: ReadonlyMap<string, Room>,
): SmartDevice['position'] => {
  if (isSmartDevicePositionAllowed(device, position, roomsById)) {
    return position;
  }

  const room = roomsById.get(device.room_id);

  if (!room) {
    return position;
  }

  const constrainedPosition = getConstrainedPointOnDragPath(
    [currentPosition.x, currentPosition.y],
    [position.x, position.y],
    room.area,
    (candidate) =>
      isPointInPolygon([candidate[0], candidate[1]], room.area),
  );

  return {
    x: constrainedPosition[0],
    y: constrainedPosition[1],
  };
};

export const updateSmartDevicePosition = (
  devices: SmartDevice[],
  deviceId: string,
  position: SmartDevice['position'],
): SmartDevice[] =>
  devices.map((device) =>
    device.id === deviceId ? ({ ...device, position } as SmartDevice) : device,
  );
