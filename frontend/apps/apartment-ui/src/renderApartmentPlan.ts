import { createElement } from 'react';
import { createRoot } from 'react-dom/client';

import { ApartmentPlanView } from './components/ApartmentPlanView';
import type { FloorPlan, SmartDevice, Zone } from './types';

export interface ApartmentPlanRenderHandle {
  update(
    plan?: FloorPlan | null,
    devices?: SmartDevice[] | null,
    zones?: Zone[] | null,
  ): void;
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

  const update = (
    nextPlan?: FloorPlan | null,
    nextDevices?: SmartDevice[] | null,
    nextZones?: Zone[] | null,
  ) => {
    if (destroyed) {
      throw new Error('Cannot update a destroyed apartment plan renderer.');
    }

    root.render(
      createElement(ApartmentPlanView, {
        plan: nextPlan ?? createEmptyPlan(),
        devices: nextDevices ?? [],
        zones: nextZones ?? [],
      }),
    );
  };

  update(plan, devices, zones);

  return {
    update,
    destroy: () => {
      if (destroyed) {
        return;
      }

      root.unmount();
      destroyed = true;
    },
  };
}
