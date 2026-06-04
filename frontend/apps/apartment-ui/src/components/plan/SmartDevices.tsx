import { useMemo, useRef } from 'react';
import { Circle, Group, Path } from 'react-konva';
import type { Room, SmartDevice } from '../../types';
import { isPointInPolygon } from '../../utils/polygon';
import { DEVICE_PATHS } from './devicePaths';

interface SmartDevicesProps {
  devices: SmartDevice[];
  rooms: Room[];
  selectedDeviceId: string | null;
  onSelectDevice: (id: string | null) => void;
  onMoveDevice: (id: string, position: SmartDevice['position']) => void;
}

export function SmartDevices({
  devices,
  rooms,
  selectedDeviceId,
  onSelectDevice,
  onMoveDevice,
}: SmartDevicesProps) {
  const draggedDeviceIds = useRef<Set<string>>(new Set());
  const lastValidPositions = useRef<Map<string, SmartDevice['position']>>(new Map());
  const roomsById = useMemo(
    () => new Map(rooms.map((room) => [room.id, room])),
    [rooms],
  );

  const isDevicePositionAllowed = (
    device: SmartDevice,
    position: SmartDevice['position'],
  ): boolean => {
    const room = roomsById.get(device.room_id);

    if (!room) {
      return true;
    }

    return isPointInPolygon([position.x, position.y], room.area);
  };

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
            draggable
            dragDistance={10}
            onDragStart={(event) => {
              event.cancelBubble = true;
              draggedDeviceIds.current.add(device.id);
              lastValidPositions.current.set(device.id, {
                x: event.target.x(),
                y: event.target.y(),
              });

              const container = event.target.getStage()?.container();
              if (container) container.style.cursor = 'grabbing';
            }}
            onDragMove={(event) => {
              event.cancelBubble = true;
              const nextPosition = {
                x: event.target.x(),
                y: event.target.y(),
              };

              if (isDevicePositionAllowed(device, nextPosition)) {
                lastValidPositions.current.set(device.id, nextPosition);
                return;
              }

              const lastValidPosition =
                lastValidPositions.current.get(device.id) ?? device.position;
              event.target.position(lastValidPosition);
            }}
            onDragEnd={(event) => {
              event.cancelBubble = true;
              const nextPosition = {
                x: event.target.x(),
                y: event.target.y(),
              };
              const finalPosition = isDevicePositionAllowed(device, nextPosition)
                ? nextPosition
                : lastValidPositions.current.get(device.id) ?? device.position;

              event.target.position(finalPosition);
              onMoveDevice(device.id, finalPosition);

              const container = event.target.getStage()?.container();
              if (container) container.style.cursor = 'grab';

              window.setTimeout(() => {
                draggedDeviceIds.current.delete(device.id);
                lastValidPositions.current.delete(device.id);
              }, 250);
            }}
            onClick={(event) => {
              event.cancelBubble = true;

              if (draggedDeviceIds.current.has(device.id)) {
                return;
              }

              onSelectDevice(isSelected ? null : device.id);
            }}
            onTap={(event) => {
              event.cancelBubble = true;

              if (draggedDeviceIds.current.has(device.id)) {
                return;
              }

              onSelectDevice(isSelected ? null : device.id);
            }}
            onMouseEnter={(event) => {
              const container = event.target.getStage()?.container();
              if (container) container.style.cursor = 'grab';
            }}
            onMouseLeave={(event) => {
              const container = event.target.getStage()?.container();
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
