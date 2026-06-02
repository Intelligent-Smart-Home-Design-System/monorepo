export type Point = [number, number];

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

export interface FloorPlan {
  walls: Wall[];
  doors: Door[];
  windows: WindowOpening[];
  rooms: Room[];
}

export interface SmartDevice {
  id: string;
  type: 'smart_lamp' | 'smart_plug' | 'motion_sensor' | 'temperature_sensor' | 'water_leak_sensor';
  room_id: string;
  position: { x: number; y: number };
  state: any; // Пока оставим any, так как у разных устройств разные стейты
}