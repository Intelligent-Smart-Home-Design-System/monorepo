import { useCallback, useState } from 'react';

import type { FloorPlan, SmartDevice, Zone } from '../types';
import { updateSmartDevicePosition } from '../utils/devices';
import { updateZonePoint } from '../utils/zones';
import { FloorPlanStage } from './FloorPlanStage';

interface ApartmentPlanViewProps {
  plan: FloorPlan;
  devices: SmartDevice[];
  zones: Zone[];
  showOpeningHitboxes?: boolean;
  onDevicesChange?: (devices: SmartDevice[]) => void;
}

interface DeviceState {
  basis: SmartDevice[];
  current: SmartDevice[];
}

interface ZoneState {
  basis: Zone[];
  current: Zone[];
}

export function ApartmentPlanView({
  plan,
  devices,
  zones,
  showOpeningHitboxes = false,
  onDevicesChange,
}: ApartmentPlanViewProps) {
  const [selectedDeviceId, setSelectedDeviceId] = useState<string | null>(null);
  const [selectedZoneId, setSelectedZoneId] = useState<string | null>(null);
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
  const visibleSelectedZoneId =
    selectedZoneId && visibleZones.some((zone) => zone.id === selectedZoneId)
      ? selectedZoneId
      : null;
  const handleSelectDevice = useCallback((deviceId: string | null) => {
    setSelectedDeviceId(deviceId);

    if (deviceId) {
      setSelectedZoneId(null);
    }
  }, []);
  const handleSelectZone = useCallback((zoneId: string | null) => {
    setSelectedZoneId(zoneId);

    if (zoneId) {
      setSelectedDeviceId(null);
    }
  }, []);
  const handleMoveDevice = useCallback(
    (deviceId: string, position: SmartDevice['position']) => {
      const nextDevices = updateSmartDevicePosition(
        visibleDevices,
        deviceId,
        position,
      );

      setDeviceState({
        basis: devices,
        current: nextDevices,
      });
      onDevicesChange?.(nextDevices);
    },
    [devices, onDevicesChange, visibleDevices],
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
      showOpeningHitboxes={showOpeningHitboxes}
      selectedDeviceId={visibleSelectedDeviceId}
      selectedZoneId={visibleSelectedZoneId}
      onSelectDevice={handleSelectDevice}
      onSelectZone={handleSelectZone}
      onMoveDevice={handleMoveDevice}
      onMoveZonePoint={handleMoveZonePoint}
    />
  );
}
