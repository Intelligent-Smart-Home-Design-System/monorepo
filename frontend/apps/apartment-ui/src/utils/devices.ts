import type { SmartDevice } from '../types';

export const updateSmartDevicePosition = (
  devices: SmartDevice[],
  deviceId: string,
  position: SmartDevice['position'],
): SmartDevice[] =>
  devices.map((device) =>
    device.id === deviceId ? ({ ...device, position } as SmartDevice) : device,
  );
