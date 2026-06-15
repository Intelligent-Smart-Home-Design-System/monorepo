import { Group, Line } from 'react-konva';
import type { WindowOpening, FloorPlan } from '../../types';
import { getWallThickness } from '../../utils/geometry';

interface WindowsProps {
  windows: WindowOpening[];
  walls: FloorPlan['walls'];
}

const OPENING_MARKER_STROKE_WIDTH = 2;

export function Windows({ windows, walls }: WindowsProps) {
  return (
    <Group id="windows-layer">
      {windows.map((win) => {
        const wallWidth = getWallThickness(win.points, walls);
        const p1 = win.points[0];
        const p2 = win.points[1];

        // Вычисляем вектор направления
        const dx = p2[0] - p1[0];
        const dy = p2[1] - p1[1];
        const len = Math.sqrt(dx * dx + dy * dy);
        const nx = dx / len;
        const ny = dy / len;
        
        // Удлиняем координаты на 5мм в обе стороны
        const ext = 5; 
        const extendedPoints = [
          p1[0] - nx * ext, p1[1] - ny * ext,
          p2[0] + nx * ext, p2[1] + ny * ext
        ];

        return (
          <Group key={win.id} id={`window-${win.id}`}>
            {/* Ластик */}
            <Line
              points={extendedPoints}
              stroke="#f4f6f8"
              strokeWidth={wallWidth + 12}
              lineCap="butt"
            />
            {/* Стекло */}
            <Line
              points={extendedPoints}
              stroke="#3498db"
              strokeWidth={OPENING_MARKER_STROKE_WIDTH}
              lineCap="butt"
            />
          </Group>
        );
      })}
    </Group>
  );
}
