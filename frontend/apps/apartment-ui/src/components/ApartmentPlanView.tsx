import { useCallback, useState } from 'react';

import type { FloorPlan, SmartDevice, Zone } from '../types';
import { updateSmartDevicePosition } from '../utils/devices';
import { updateZonePoint } from '../utils/zones';
import { FloorPlanStage } from './FloorPlanStage';

interface ApartmentPlanViewProps {
  plan: FloorPlan;
  devices: SmartDevice[];
  zones: Zone[];
}

interface DeviceState {
  basis: SmartDevice[];
  current: SmartDevice[];
}

interface ZoneState {
  basis: Zone[];
  current: Zone[];
}

export function ApartmentPlanView({ plan, devices, zones }: ApartmentPlanViewProps) {
  const [selectedDeviceId, setSelectedDeviceId] = useState<string | null>(null);
  const [deviceState, setDeviceState] = useState<DeviceState>(() => ({
    basis: devices,
    current: devices,
  }));
  const [zoneState, setZoneState] = useState<ZoneState>(() => ({
    basis: zones,
    current: zones,
  }));
  const visibleDevices =
    deviceState.basis === devices ? deviceState.current : devices;
  const visibleZones = zoneState.basis === zones ? zoneState.current : zones;
  const visibleSelectedDeviceId =
    selectedDeviceId &&
    visibleDevices.some((device) => device.id === selectedDeviceId)
      ? selectedDeviceId
      : null;
  const handleMoveDevice = useCallback(
    (deviceId: string, position: SmartDevice['position']) => {
      setDeviceState({
        basis: devices,
        current: updateSmartDevicePosition(visibleDevices, deviceId, position),
      });
    },
    [devices, visibleDevices],
  );
  const handleMoveZonePoint = useCallback(
    (zoneId: string, pointIndex: number, point: Zone['points'][number]) => {
      setZoneState({
        basis: zones,
        current: updateZonePoint(visibleZones, zoneId, pointIndex, point, plan.rooms),
      });
    },
    [plan.rooms, visibleZones, zones],
  );

  return (
    <FloorPlanStage
      plan={plan}
      devices={visibleDevices}
      zones={visibleZones}
      selectedDeviceId={visibleSelectedDeviceId}
      onSelectDevice={setSelectedDeviceId}
      onMoveDevice={handleMoveDevice}
      onMoveZonePoint={handleMoveZonePoint}
    />
  );
}
