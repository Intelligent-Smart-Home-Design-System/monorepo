import { Group, Line, Text } from 'react-konva';

import type { Point, Room } from '../../types';

interface RoomsProps {
  rooms: Room[];
}

const getRoomCenter = (area: Point[]): Point => {
  if (area.length === 0) {
    return [0, 0];
  }

  const first = area[0];
  const last = area[area.length - 1];

  if (!first || !last) {
    return [0, 0];
  }

  const points =
    area.length > 1 && first[0] === last[0] && first[1] === last[1]
      ? area.slice(0, -1)
      : area;

  if (points.length === 0) {
    return [0, 0];
  }

  const totals = points.reduce(
    (acc, [x, y]) => ({
      x: acc.x + x,
      y: acc.y + y,
    }),
    { x: 0, y: 0 },
  );

  return [totals.x / points.length, totals.y / points.length];
};

export function Rooms({ rooms }: RoomsProps) {
  return (
    <>
      {rooms.map((room) => {
        const [centerX, centerY] = getRoomCenter(room.area);
        const labelWidth = 2000;
        const labelHeight = 220;

        return (
          <Group key={room.id}>
            <Line points={room.area.flat()} fill="#ffffff" closed />
            <Text
              x={centerX - labelWidth / 2}
              y={centerY - labelHeight / 2}
              width={labelWidth}
              height={labelHeight}
              text={room.name}
              fontSize={160}
              fontFamily="sans-serif"
              fontStyle="italic"
              fill="#7f8c8d"
              align="center"
              verticalAlign="middle"
            />
          </Group>
        );
      })}
    </>
  );
}
