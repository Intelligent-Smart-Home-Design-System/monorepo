import { useState } from 'react';
import { FloorPlanStage } from './components/FloorPlanStage';
import { DeviceSidebar } from './components/plan/DeviceSidebar';
import rawPlanData from './image_apartment_plan.json';
import rawDevicesData from './devices.json';
import type { FloorPlan, SmartDevice } from './types';

const currentPlan = rawPlanData as unknown as FloorPlan;
const currentDevices = rawDevicesData as unknown as SmartDevice[];

export default function App() {
  // Главное состояние приложения: какое устройство сейчас выбрано
  const [selectedDeviceId, setSelectedDeviceId] = useState<string | null>(null);

  return (
    <div style={{ display: 'flex', width: '100vw', height: '100vh', overflow: 'hidden' }}>
      
      {/* Левая часть: 2D движок с планом */}
      <div style={{ flex: 1, position: 'relative' }}>
        <FloorPlanStage 
          plan={currentPlan} 
          devices={currentDevices}
          selectedDeviceId={selectedDeviceId}
          onSelectDevice={setSelectedDeviceId}
        />
      </div>

      {/* Правая часть: HTML панель */}
      <DeviceSidebar 
        devices={currentDevices}
        selectedDeviceId={selectedDeviceId}
        onSelectDevice={setSelectedDeviceId}
      />
      
    </div>
  );
}