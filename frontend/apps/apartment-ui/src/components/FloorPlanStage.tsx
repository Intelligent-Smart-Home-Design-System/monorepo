import { useEffect, useMemo, useRef, useState } from 'react';
import { Layer, Stage } from 'react-konva';
import type { KonvaEventObject } from 'konva/lib/Node';
import type { Stage as KonvaStage } from 'konva/lib/Stage';

import type { FloorPlan, SmartDevice } from '../types';
import { calculateInitialView, type InitialView, type ViewportSize } from '../utils/geometry';
import { Doors } from './plan/Doors';
import { Rooms } from './plan/Rooms';
import { Walls } from './plan/Walls';
import { Windows } from './plan/Windows';
import { SmartDevices } from './plan/SmartDevices';

interface FloorPlanStageProps {
  plan: FloorPlan;
  devices: SmartDevice[];
  selectedDeviceId: string | null;
  onSelectDevice: (id: string | null) => void;
}

// Отнимаем ширину сайдбара (320px) от ширины окна
const getViewportSize = (): ViewportSize => ({
  width: window.innerWidth - 320, 
  height: window.innerHeight,
});

export function FloorPlanStage({ plan, devices, selectedDeviceId, onSelectDevice }: FloorPlanStageProps) {
  const stageRef = useRef<KonvaStage | null>(null);
  const [viewportSize, setViewportSize] = useState<ViewportSize>(() => getViewportSize());
  const [view, setView] = useState<InitialView>(() =>
    calculateInitialView(plan, getViewportSize()),
  );

  const hatchPattern = useMemo<HTMLCanvasElement | null>(() => {
    const canvas = document.createElement('canvas');
    canvas.width = 40;
    canvas.height = 40;

    const ctx = canvas.getContext('2d');

    if (!ctx) {
      return null;
    }

    ctx.strokeStyle = '#aab7c4';
    ctx.lineWidth = 4;
    ctx.beginPath();
    ctx.moveTo(0, 0);
    ctx.lineTo(40, 40);
    ctx.stroke();

    return canvas;
  }, []);

  useEffect(() => {
    const handleResize = () => {
      const nextViewportSize = getViewportSize();

      setViewportSize(nextViewportSize);
      setView(calculateInitialView(plan, nextViewportSize));
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, [plan]);

  const clampPosition = (
    newX: number,
    newY: number,
    currentScale: number,
  ): { x: number; y: number } => {
    const margin = 150;
    const minAllowedX = margin - view.maxX * currentScale;
    const maxAllowedX = viewportSize.width - margin - view.minX * currentScale;
    const minAllowedY = margin - view.maxY * currentScale;
    const maxAllowedY = viewportSize.height - margin - view.minY * currentScale;

    return {
      x: Math.max(minAllowedX, Math.min(newX, maxAllowedX)),
      y: Math.max(minAllowedY, Math.min(newY, maxAllowedY)),
    };
  };

  const handleWheel = (event: KonvaEventObject<WheelEvent>) => {
    event.evt.preventDefault();

    const stage = stageRef.current;

    if (!stage) {
      return;
    }

    const pointer = stage.getPointerPosition();

    if (!pointer) {
      return;
    }

    const scaleBy = 1.1;
    const oldScale = stage.scaleX();
    const mousePointTo = {
      x: (pointer.x - stage.x()) / oldScale,
      y: (pointer.y - stage.y()) / oldScale,
    };
    const nextScale =
      event.evt.deltaY < 0 ? oldScale * scaleBy : oldScale / scaleBy;
    const newScale = Math.max(0.05, Math.min(nextScale, 3));
    const newX = pointer.x - mousePointTo.x * newScale;
    const newY = pointer.y - mousePointTo.y * newScale;
    const clampedPosition = clampPosition(newX, newY, newScale);

    setView((prev) => ({
      ...prev,
      scale: newScale,
      x: clampedPosition.x,
      y: clampedPosition.y,
    }));
  };


  return (
    <Stage
      width={viewportSize.width}
      height={viewportSize.height}
      onWheel={handleWheel}
      draggable
      ref={stageRef}
      scaleX={view.scale}
      scaleY={view.scale}
      x={view.x}
      y={view.y}
      style={{ cursor: 'grab', backgroundColor: '#f4f6f8' }}
      dragBoundFunc={(pos) => clampPosition(pos.x, pos.y, view.scale)}
      onDragEnd={(event) => {
        setView((prev) => ({ ...prev, x: event.target.x(), y: event.target.y() }));
      }}
      // Сброс выделения при клике на пустой фон или пол
      onClick={(e) => {
        if (e.target === e.target.getStage() || e.target.name() === 'room-floor') {
          onSelectDevice(null);
        }
      }}
    >
      <Layer>
        <Rooms rooms={plan.rooms} />
        <Walls walls={plan.walls} hatchPattern={hatchPattern} />
        <Doors doors={plan.doors} walls={plan.walls} />
        <Windows windows={plan.windows} walls={plan.walls} />
        {/* Передаем пропсы в устройства */}
        <SmartDevices 
          devices={devices} 
          selectedDeviceId={selectedDeviceId}
          onSelectDevice={onSelectDevice}
        />
      </Layer>
    </Stage>
  );
}
