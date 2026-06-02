import { Group, Circle, Path } from 'react-konva';
import type { SmartDevice } from '../../types';

export const DEVICE_PATHS: Record<SmartDevice['type'], string> = {
  smart_lamp: "M11 19v-8H6q-.5 0-.8-.4t-.15-.9L7 3.4q.2-.625.725-1.013T8.9 2h6.2q.65 0 1.175.388T17 3.4l1.95 6.3q.15.5-.15.9t-.8.4h-5v8zm-3 3v-2h8v2z",
  smart_plug: "M9.5 21v-3L6 14.5V9q0-.825.588-1.412T8 7h1L8 8V3h2v4h4V3h2v5l-1-1h1q.825 0 1.413.588T18 9v5.5L14.5 18v3z",
  motion_sensor: "M4 22q-.825 0-1.412-.587T2 20v-4h2v4h4v2zM2 8V4q0-.825.588-1.412T4 2h4v2H4v4zm9 10.9q-2.3-.35-3.925-1.975T5.1 13h2q.3 1.475 1.363 2.537T11 16.9zM5.1 11q.35-2.3 1.975-3.937T11 5.1v2q-1.475.3-2.537 1.363T7.1 11zm6.9 3q-.825 0-1.412-.587T10 12q0-.85.588-1.425T12 10q.85 0 1.425.575T14 12q0 .825-.575 1.413T12 14m1 4.9v-2q1.475-.3 2.538-1.362T16.9 13h2q-.325 2.3-1.962 3.925T13 18.9m3.9-7.9q-.3-1.475-1.362-2.537T13 7.1v-2q2.3.35 3.938 1.975T18.9 11zM16 22v-2h4v-4h2v4q0 .825-.587 1.413T20 22zm4-14V4h-4V2h4q.825 0 1.413.588T22 4v4z",
  temperature_sensor: "M12 21q-2.075 0-3.537-1.463T7 16q0-1.2.525-2.238T9 12V6q0-1.25.875-2.125T12 3t2.125.875T15 6v6q.95.725 1.475 1.763T17 16q0 2.075-1.463 3.538T12 21m-1-11h2V6q0-.425-.288-.712T12 5t-.712.288T11 6z",
  water_leak_sensor: "M12.275 19q.3-.025.513-.238T13 18.25q0-.35-.225-.562T12.2 17.5q-1.025.075-2.175-.562t-1.45-2.313q-.05-.275-.262-.45T7.825 14q-.35 0-.575.263t-.15.612q.425 2.275 2 3.25t3.175.875m-5.987.65Q4 17.3 4 13.8q0-2.5 1.988-5.437T12 2q4.025 3.425 6.013 6.363T20 13.8q0 3.5-2.287 5.85T12 22t-5.712-2.35",
};

interface SmartDevicesProps {
  devices: SmartDevice[];
  selectedDeviceId: string | null;
  onSelectDevice: (id: string | null) => void;
}

export function SmartDevices({ devices, selectedDeviceId, onSelectDevice }: SmartDevicesProps) {
  return (
    <Group id="smart-devices-layer">
      {devices.map((device) => {
        const pathData = DEVICE_PATHS[device.type];
        
        // Логика цвета: Зеленый если выбран, иначе Серый
        const isSelected = device.id === selectedDeviceId;
        const color = isSelected ? '#2ecc71' : '#bdc3c7';
        
        const scale = 20; 
        
        return (
          <Group 
            key={device.id} 
            x={device.position.x} 
            y={device.position.y}
            // ОБРАБОТЧИКИ КЛИКОВ
            onClick={(e) => {
              e.cancelBubble = true; // Блокируем клик, чтобы он не ушел "в пол"
              onSelectDevice(device.id);
            }}
            onTap={(e) => {
              e.cancelBubble = true;
              onSelectDevice(device.id);
            }}
            // Меняем курсор при наведении
            onMouseEnter={(e) => {
              const container = e.target.getStage()?.container();
              if (container) container.style.cursor = 'pointer';
            }}
            onMouseLeave={(e) => {
              const container = e.target.getStage()?.container();
              if (container) container.style.cursor = 'grab';
            }}
          >
            <Circle 
              radius={300} 
              fill="#ffffff"
              shadowColor="#000000"
              shadowBlur={isSelected ? 20 : 10} // Выбранный датчик отбрасывает бОльшую тень
              shadowOpacity={0.2}
              stroke={isSelected ? '#2ecc71' : 'transparent'} // Добавляем зеленую обводку кругу при выборе
              strokeWidth={20}
            />
            {pathData && (
              <Path
                data={pathData}
                fill={color}
                scaleX={scale}
                scaleY={scale}
                offsetX={12} 
                offsetY={12}
              />
            )}
          </Group>
        );
      })}
    </Group>
  );
}