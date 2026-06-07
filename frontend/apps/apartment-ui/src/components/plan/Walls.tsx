import { Shape } from 'react-konva';
import type { Context } from 'konva/lib/Context';

import type { Point, Wall } from '../../types';
import { getRenderableWallWidth } from '../../utils/geometry';

interface WallsProps {
  walls: Wall[];
  hatchPattern: HTMLCanvasElement | null;
}

const buildWallPath = (ctx: Context, points: Point[]): void => {
  if (points.length < 2) {
    return;
  }

  const first = points[0];
  const last = points[points.length - 1];

  if (!first || !last) {
    return;
  }

  const isClosed = first[0] === last[0] && first[1] === last[1];
  const endIndex = isClosed ? points.length - 1 : points.length;

  ctx.moveTo(first[0], first[1]);

  for (let index = 1; index < endIndex; index += 1) {
    const point = points[index];

    if (!point) {
      continue;
    }

    ctx.lineTo(point[0], point[1]);
  }

  if (isClosed) {
    ctx.closePath();
  }
};

export function Walls({ walls, hatchPattern }: WallsProps) {
  return (
    <Shape
      sceneFunc={(ctx: Context) => {
        ctx.lineJoin = 'miter';
        ctx.lineCap = 'square';

        walls.forEach((wall) => {
          ctx.beginPath();
          buildWallPath(ctx, wall.points);
          ctx.lineWidth = getRenderableWallWidth(wall.width) + 8;
          ctx.strokeStyle = '#8395a7';
          ctx.stroke();
        });

        walls.forEach((wall) => {
          ctx.beginPath();
          buildWallPath(ctx, wall.points);
          ctx.lineWidth = getRenderableWallWidth(wall.width);
          ctx.strokeStyle = '#ffffff';
          ctx.stroke();
        });

        if (!hatchPattern) {
          return;
        }

        const pattern = ctx.createPattern(hatchPattern, 'repeat');

        if (!pattern) {
          return;
        }

        walls.forEach((wall) => {
          ctx.beginPath();
          buildWallPath(ctx, wall.points);
          ctx.lineWidth = getRenderableWallWidth(wall.width);
          ctx.strokeStyle = pattern;
          ctx.stroke();
        });
      }}
    />
  );
}
