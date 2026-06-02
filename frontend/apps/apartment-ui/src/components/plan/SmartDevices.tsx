import { Circle, Group, Path } from 'react-konva';
import type { SmartDevice } from '../../types';
import { DEVICE_PATHS } from './devicePaths';

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
        const isSelected = device.id === selectedDeviceId;
        const color = isSelected ? '#2ecc71' : '#bdc3c7';
        const scale = 20;

        return (
          <Group
            key={device.id}
            x={device.position.x}
            y={device.position.y}
            onClick={(e) => {
              e.cancelBubble = true;
              onSelectDevice(device.id);
            }}
            onTap={(e) => {
              e.cancelBubble = true;
              onSelectDevice(device.id);
            }}
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
              shadowBlur={isSelected ? 20 : 10}
              shadowOpacity={0.2}
              stroke={isSelected ? '#2ecc71' : 'transparent'}
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
