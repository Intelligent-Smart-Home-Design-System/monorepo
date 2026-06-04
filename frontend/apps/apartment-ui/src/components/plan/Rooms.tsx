import { Group, Line, Text } from 'react-konva';

import type { Room } from '../../types';
import { flattenPolygonPoints, getPolygonCentroid } from '../../utils/polygon';

interface RoomsProps {
  rooms: Room[];
}

export function Rooms({ rooms }: RoomsProps) {
  return (
    <>
      {rooms.map((room) => {
        const [centerX, centerY] = getPolygonCentroid(room.area);
        const labelWidth = 2000;
        const labelHeight = 220;

        return (
          <Group key={room.id}>
            <Line
              name="room-floor"
              points={flattenPolygonPoints(room.area)}
              fill="#ffffff"
              closed
            />
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
