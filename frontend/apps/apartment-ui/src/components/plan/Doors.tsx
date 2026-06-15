import { Group, Line } from 'react-konva';
import type { Door, FloorPlan } from '../../types';
import { getWallThickness } from '../../utils/geometry';

interface DoorsProps {
  doors: Door[];
  walls: FloorPlan['walls'];
}

const OPENING_MARKER_STROKE_WIDTH = 2;

export function Doors({ doors, walls }: DoorsProps) {
  return (
    <Group id="doors-layer">
      {doors.map((door) => {
        const wallWidth = getWallThickness(door.points, walls);
        const p1 = door.points[0];
        const p2 = door.points[1];

        // Вычисляем вектор направления
        const dx = p2[0] - p1[0];
        const dy = p2[1] - p1[1];
        const len = Math.sqrt(dx * dx + dy * dy);
        const nx = dx / len;
        const ny = dy / len;
        
        // Удлиняем координаты на 5мм в обе стороны для идеального стыка
        const ext = 5; 
        const extendedPoints = [
          p1[0] - nx * ext, p1[1] - ny * ext,
          p2[0] + nx * ext, p2[1] + ny * ext
        ];

        return (
          <Group key={door.id} id={`door-${door.id}`}>
            {/* Ластик: толще стены на 12px, чтобы с запасом перекрыть контур */}
            <Line
              points={extendedPoints}
              stroke="#f4f6f8"
              strokeWidth={wallWidth + 12}
              lineCap="butt"
            />
            {/* Дверь: использует те же удлиненные координаты */}
            <Line
              points={extendedPoints}
              stroke="#8e44ad"
              strokeWidth={OPENING_MARKER_STROKE_WIDTH}
              lineCap="butt"
            />
          </Group>
        );
      })}
    </Group>
  );
}
