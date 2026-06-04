export type Point = [number, number];

export interface LayoutPoint {
  X: number;
  Y: number;
}

export interface Wall {
  id: string;
  points: Point[];
  width: number;
}

export interface Door {
  id: string;
  points: Point[];
  width: number;
}

export interface WindowOpening {
  id: string;
  points: Point[];
  width: number;
}

export interface Room {
  id: string;
  name: string;
  area: Point[];
}

export interface Zone {
  id: string;
  room_id?: string;
  roomId?: string;
  points: LayoutPoint[];
}

export interface FloorPlan {
  walls: Wall[];
  doors: Door[];
  windows: WindowOpening[];
  rooms: Room[];
}

export type SmartDeviceType =
  | 'smart_lamp'
  | 'smart_plug'
  | 'motion_sensor'
  | 'temperature_sensor'
  | 'water_leak_sensor';

export interface SmartDeviceStateByType {
  smart_lamp: { is_on: boolean };
  smart_plug: { is_on: boolean };
  motion_sensor: { detected: boolean };
  temperature_sensor: { value: number };
  water_leak_sensor: { leak_detected: boolean };
}

interface SmartDeviceBase<TType extends SmartDeviceType> {
  id: string;
  type: TType;
  room_id: string;
  position: { x: number; y: number };
}

export type SmartDevice = {
  [TType in SmartDeviceType]: SmartDeviceBase<TType> & {
    state: SmartDeviceStateByType[TType];
  };
}[SmartDeviceType];
