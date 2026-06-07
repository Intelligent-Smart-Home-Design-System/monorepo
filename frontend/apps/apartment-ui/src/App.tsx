import { useCallback, useState } from 'react';
import { FloorPlanStage } from './components/FloorPlanStage';
import { DeviceSidebar } from './components/plan/DeviceSidebar';
import rawDevicesData from './devices.json';
import rawPlanData from './image_apartment_plan.json';
import type { FloorPlan, SmartDevice, Zone } from './types';
import { updateSmartDevicePosition } from './utils/devices';
import { updateZonePoint } from './utils/zones';
import rawZonesData from './zones.json';

const currentPlan = rawPlanData as unknown as FloorPlan;
const currentDevices = rawDevicesData as unknown as SmartDevice[];
const currentZones = rawZonesData as unknown as Zone[];

interface AppProps {
  plan?: FloorPlan;
  devices?: SmartDevice[];
  zones?: Zone[];
}

export default function App({
  plan = currentPlan,
  devices: initialDevices = currentDevices,
  zones = currentZones,
}: AppProps) {
  const [selectedDeviceId, setSelectedDeviceId] = useState<string | null>(null);
  const [selectedZoneId, setSelectedZoneId] = useState<string | null>(null);
  const [devices, setDevices] = useState<SmartDevice[]>(() => initialDevices);
  const [editableZones, setEditableZones] = useState<Zone[]>(() => zones);
  const selectedZone =
    editableZones.find((zone) => zone.id === selectedZoneId) ?? null;
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
      setDevices((currentDevicesState) =>
        updateSmartDevicePosition(currentDevicesState, deviceId, position),
      );
    },
    [],
  );
  const handleMoveZonePoint = useCallback(
    (zoneId: string, pointIndex: number, point: Zone['points'][number]) => {
      setEditableZones((currentZonesState) =>
        updateZonePoint(currentZonesState, zoneId, pointIndex, point, plan.rooms),
      );
    },
    [plan.rooms],
  );

  return (
    <div style={{ display: 'flex', width: '100vw', height: '100vh', overflow: 'hidden' }}>
      <div style={{ flex: 1, position: 'relative' }}>
        <FloorPlanStage
          plan={plan}
          devices={devices}
          zones={editableZones}
          selectedDeviceId={selectedDeviceId}
          selectedZoneId={selectedZoneId}
          onSelectDevice={handleSelectDevice}
          onSelectZone={handleSelectZone}
          onMoveDevice={handleMoveDevice}
          onMoveZonePoint={handleMoveZonePoint}
        />
      </div>

      <DeviceSidebar
        devices={devices}
        selectedZone={selectedZone}
        selectedDeviceId={selectedDeviceId}
        onSelectDevice={handleSelectDevice}
      />
    </div>
  );
}
