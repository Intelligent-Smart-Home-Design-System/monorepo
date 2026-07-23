import { useEffect, useMemo, useRef, useState } from 'react';
import { Layer, Stage } from 'react-konva';
import type { KonvaEventObject } from 'konva/lib/Node';
import type { Stage as KonvaStage } from 'konva/lib/Stage';

import type { FloorPlan, SmartDevice, Zone } from '../types';
import { calculateInitialView, type InitialView, type ViewportSize } from '../utils/geometry';
import { Doors } from './plan/Doors';
import { Rooms } from './plan/Rooms';
import { SmartDevices } from './plan/SmartDevices';
import { Walls } from './plan/Walls';
import { Windows } from './plan/Windows';
import { Zones } from './plan/Zones';

interface FloorPlanStageProps {
  plan: FloorPlan;
  devices: SmartDevice[];
  zones?: Zone[];
  showOpeningHitboxes?: boolean;
  selectedDeviceId: string | null;
  selectedZoneId: string | null;
  onSelectDevice: (id: string | null) => void;
  onSelectZone: (id: string | null) => void;
  onMoveDevice: (id: string, position: SmartDevice['position']) => void;
  onMoveZonePoint: (zoneId: string, pointIndex: number, point: Zone['points'][number]) => void;
}

interface ViewState {
  basis: InitialView;
  current: InitialView;
}

const DEFAULT_VIEWPORT_SIZE: ViewportSize = {
  width: 1,
  height: 1,
};

const EMPTY_ZONES: Zone[] = [];

const getElementSize = (element: HTMLElement): ViewportSize => ({
  width: Math.max(element.clientWidth, 1),
  height: Math.max(element.clientHeight, 1),
});

const clampAxis = (value: number, min: number, max: number): number =>
  min <= max ? Math.max(min, Math.min(value, max)) : (min + max) / 2;

export function FloorPlanStage({
  plan,
  devices,
  zones = EMPTY_ZONES,
  showOpeningHitboxes = false,
  selectedDeviceId,
  selectedZoneId,
  onSelectDevice,
  onSelectZone,
  onMoveDevice,
  onMoveZonePoint,
}: FloorPlanStageProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const stageRef = useRef<KonvaStage | null>(null);
  const [viewportSize, setViewportSize] = useState<ViewportSize>(DEFAULT_VIEWPORT_SIZE);
  const initialView = useMemo(
    () => calculateInitialView(plan, viewportSize),
    [plan, viewportSize],
  );
  const [viewState, setViewState] = useState<ViewState>(() => ({
    basis: initialView,
    current: initialView,
  }));
  const view = viewState.basis === initialView ? viewState.current : initialView;

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
    const container = containerRef.current;

    if (!container) {
      return;
    }

    const updateViewportSize = () => {
      const nextViewportSize = getElementSize(container);

      setViewportSize((currentViewportSize) =>
        currentViewportSize.width === nextViewportSize.width &&
        currentViewportSize.height === nextViewportSize.height
          ? currentViewportSize
          : nextViewportSize,
      );
    };

    updateViewportSize();

    const observer = new ResizeObserver(updateViewportSize);
    observer.observe(container);

    return () => {
      observer.disconnect();
    };
  }, []);

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
      x: clampAxis(newX, minAllowedX, maxAllowedX),
      y: clampAxis(newY, minAllowedY, maxAllowedY),
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

    setViewState({
      basis: initialView,
      current: {
        ...view,
        scale: newScale,
        x: clampedPosition.x,
        y: clampedPosition.y,
      },
    });
  };

  return (
    <div
      ref={containerRef}
      style={{
        width: '100%',
        height: '100%',
        minWidth: 0,
        minHeight: 0,
        overflow: 'hidden',
        backgroundColor: '#f4f6f8',
      }}
    >
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
        style={{ cursor: 'grab' }}
        dragBoundFunc={(position) => clampPosition(position.x, position.y, view.scale)}
        onDragEnd={(event) => {
          setViewState({
            basis: initialView,
            current: {
              ...view,
              x: event.target.x(),
              y: event.target.y(),
            },
          });
        }}
        onClick={(event) => {
          if (
            event.target === event.target.getStage() ||
            event.target.name() === 'room-floor'
          ) {
            onSelectDevice(null);
            onSelectZone(null);
          }
        }}
      >
        <Layer>
          <Rooms rooms={plan.rooms} />
          <Zones
            zones={zones}
            rooms={plan.rooms}
            selectedZoneId={selectedZoneId}
            renderHandles={false}
            onSelectZone={onSelectZone}
            onMoveZonePoint={onMoveZonePoint}
          />
          <Walls walls={plan.walls} hatchPattern={hatchPattern} />
          <Doors
            doors={plan.doors}
            walls={plan.walls}
            showHitboxes={showOpeningHitboxes}
          />
          <Windows
            windows={plan.windows}
            walls={plan.walls}
            showHitboxes={showOpeningHitboxes}
          />
          <SmartDevices
            devices={devices}
            rooms={plan.rooms}
            selectedDeviceId={selectedDeviceId}
            onSelectDevice={onSelectDevice}
            onMoveDevice={onMoveDevice}
          />
          <Zones
            zones={zones}
            rooms={plan.rooms}
            selectedZoneId={selectedZoneId}
            renderPolygons={false}
            onSelectZone={onSelectZone}
            onMoveZonePoint={onMoveZonePoint}
          />
        </Layer>
      </Stage>
    </div>
  );
}
