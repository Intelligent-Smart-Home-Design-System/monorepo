import { useMemo } from 'react';
import { Circle, Group, Line } from 'react-konva';

import type { LayoutPoint, Room, Zone } from '../../types';
import { flattenPolygonPoints, layoutPointToPoint } from '../../utils/polygon';
import { isZonePointMoveAllowed } from '../../utils/zones';

interface ZonesProps {
  zones: Zone[];
  rooms: Room[];
  onMoveZonePoint: (zoneId: string, pointIndex: number, point: LayoutPoint) => void;
}

const ZONE_POINT_RADIUS = 72;
const ZONE_POINT_STROKE_WIDTH = 22;

export function Zones({ zones, rooms, onMoveZonePoint }: ZonesProps) {
  const roomsById = useMemo(
    () => new Map(rooms.map((room) => [room.id, room])),
    [rooms],
  );

  const handleMovePoint = (
    zone: Zone,
    pointIndex: number,
    fallbackPoint: LayoutPoint,
    nextPoint: LayoutPoint,
    setNodePosition: (point: LayoutPoint) => void,
  ) => {
    if (!isZonePointMoveAllowed(zone, pointIndex, nextPoint, roomsById)) {
      setNodePosition(fallbackPoint);
      return;
    }

    onMoveZonePoint(zone.id, pointIndex, nextPoint);
  };

  return (
    <Group id="zones-layer">
      {zones.map((zone) => {
        const points = zone.points.map(layoutPointToPoint);

        return (
          <Group key={zone.id}>
            <Line
              points={flattenPolygonPoints(points)}
              fill="rgba(243, 156, 18, 0.2)"
              stroke="#d68910"
              strokeWidth={35}
              dash={[120, 80]}
              closed
              listening={false}
            />
            {zone.points.map((point, pointIndex) => (
              <Circle
                key={`${zone.id}-${pointIndex}`}
                x={point.X}
                y={point.Y}
                radius={ZONE_POINT_RADIUS}
                fill="#ffffff"
                stroke="#d68910"
                strokeWidth={ZONE_POINT_STROKE_WIDTH}
                shadowColor="#000000"
                shadowBlur={8}
                shadowOpacity={0.2}
                draggable
                dragDistance={0}
                onDragStart={(event) => {
                  event.cancelBubble = true;

                  const container = event.target.getStage()?.container();
                  if (container) container.style.cursor = 'grabbing';
                }}
                onDragMove={(event) => {
                  event.cancelBubble = true;
                  handleMovePoint(
                    zone,
                    pointIndex,
                    point,
                    {
                      X: event.target.x(),
                      Y: event.target.y(),
                    },
                    (nextPoint) => {
                      event.target.position({ x: nextPoint.X, y: nextPoint.Y });
                    },
                  );
                }}
                onDragEnd={(event) => {
                  event.cancelBubble = true;
                  handleMovePoint(
                    zone,
                    pointIndex,
                    point,
                    {
                      X: event.target.x(),
                      Y: event.target.y(),
                    },
                    (nextPoint) => {
                      event.target.position({ x: nextPoint.X, y: nextPoint.Y });
                    },
                  );

                  const container = event.target.getStage()?.container();
                  if (container) container.style.cursor = 'grab';
                }}
                onMouseEnter={(event) => {
                  const container = event.target.getStage()?.container();
                  if (container) container.style.cursor = 'grab';
                }}
                onMouseLeave={(event) => {
                  const container = event.target.getStage()?.container();
                  if (container) container.style.cursor = 'grab';
                }}
              />
            ))}
          </Group>
        );
      })}
    </Group>
  );
}
