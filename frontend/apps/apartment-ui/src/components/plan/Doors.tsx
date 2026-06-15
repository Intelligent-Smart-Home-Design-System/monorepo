import { Group, Line } from 'react-konva';
import type { Door, FloorPlan } from '../../types';
import { getWallThickness } from '../../utils/geometry';

interface DoorsProps {
  doors: Door[];
  walls: FloorPlan['walls'];
  showHitboxes?: boolean;
}

const OPENING_MARKER_STROKE_WIDTH = 2;
const OPENING_EXTENSION = 5;
const HITBOX_STROKE_WIDTH = 1;

const getOpeningHitboxPoints = (
  p1: Door['points'][number],
  p2: Door['points'][number],
  openingWidth: number,
  hitboxThickness: number,
): number[] | null => {
  const dx = p2[0] - p1[0];
  const dy = p2[1] - p1[1];
  const len = Math.sqrt(dx * dx + dy * dy);

  if (len === 0) {
    return null;
  }

  const nx = dx / len;
  const ny = dy / len;
  const px = -ny;
  const py = nx;
  const centerX = (p1[0] + p2[0]) / 2;
  const centerY = (p1[1] + p2[1]) / 2;
  const halfLength = (openingWidth > 0 ? openingWidth : len) / 2 + OPENING_EXTENSION;
  const halfThickness = hitboxThickness / 2;
  const startX = centerX - nx * halfLength;
  const startY = centerY - ny * halfLength;
  const endX = centerX + nx * halfLength;
  const endY = centerY + ny * halfLength;

  return [
    startX - px * halfThickness,
    startY - py * halfThickness,
    startX + px * halfThickness,
    startY + py * halfThickness,
    endX + px * halfThickness,
    endY + py * halfThickness,
    endX - px * halfThickness,
    endY - py * halfThickness,
  ];
};

export function Doors({ doors, walls, showHitboxes = false }: DoorsProps) {
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
        const hitboxPoints = getOpeningHitboxPoints(
          p1,
          p2,
          door.width,
          wallWidth + 12,
        );

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
            {showHitboxes && hitboxPoints && (
              <Line
                points={hitboxPoints}
                stroke="#e74c3c"
                strokeWidth={HITBOX_STROKE_WIDTH}
                closed
                listening={false}
              />
            )}
          </Group>
        );
      })}
    </Group>
  );
}
