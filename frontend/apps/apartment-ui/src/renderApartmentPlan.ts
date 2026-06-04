import { createElement } from 'react';
import { createRoot } from 'react-dom/client';

import { ApartmentPlanView } from './components/ApartmentPlanView';
import type { FloorPlan, SmartDevice, Zone } from './types';
import {
  addSmartDevice,
  clearSmartDevices,
  removeSmartDevice,
} from './utils/devices';

export interface ApartmentPlanRenderHandle {
  update(
    plan?: FloorPlan | null,
    devices?: SmartDevice[] | null,
    zones?: Zone[] | null,
  ): void;
  addDevice(device: SmartDevice): void;
  removeDevice(deviceId: string): void;
  clearDevices(): void;
  destroy(): void;
}

const createEmptyPlan = (): FloorPlan => ({
  walls: [],
  doors: [],
  windows: [],
  rooms: [],
});

export function renderApartmentPlan(
  container: HTMLElement,
  plan?: FloorPlan | null,
  devices?: SmartDevice[] | null,
  zones?: Zone[] | null,
): ApartmentPlanRenderHandle {
  if (container instanceof HTMLCanvasElement) {
    throw new TypeError(
      'renderApartmentPlan expects an empty HTML container, not a canvas element.',
    );
  }

  const root = createRoot(container);
  let destroyed = false;
  let currentPlan = plan ?? createEmptyPlan();
  let currentDevices = devices ?? [];
  let currentZones = zones ?? [];

  const renderCurrentState = () => {
    if (destroyed) {
      throw new Error('Cannot update a destroyed apartment plan renderer.');
    }

    root.render(
      createElement(ApartmentPlanView, {
        plan: currentPlan,
        devices: currentDevices,
        zones: currentZones,
        onDevicesChange: (nextDevices: SmartDevice[]) => {
          currentDevices = nextDevices;
        },
      }),
    );
  };

  const update = (
    nextPlan?: FloorPlan | null,
    nextDevices?: SmartDevice[] | null,
    nextZones?: Zone[] | null,
  ) => {
    if (destroyed) {
      throw new Error('Cannot update a destroyed apartment plan renderer.');
    }

    currentPlan = nextPlan ?? createEmptyPlan();
    currentDevices = nextDevices ?? [];
    currentZones = nextZones ?? [];
    renderCurrentState();
  };

  update(plan, devices, zones);

  return {
    update,
    addDevice: (device: SmartDevice) => {
      currentDevices = addSmartDevice(currentDevices, device);
      renderCurrentState();
    },
    removeDevice: (deviceId: string) => {
      currentDevices = removeSmartDevice(currentDevices, deviceId);
      renderCurrentState();
    },
    clearDevices: () => {
      currentDevices = clearSmartDevices();
      renderCurrentState();
    },
    destroy: () => {
      if (destroyed) {
        return;
      }

      root.unmount();
      destroyed = true;
    },
  };
}
