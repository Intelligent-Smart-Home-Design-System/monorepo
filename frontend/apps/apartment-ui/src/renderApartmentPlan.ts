import { createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';

import { FloorPlanStage } from './components/FloorPlanStage';
import { DeviceSidebar } from './components/plan/DeviceSidebar';
import type { FloorPlan, SmartDevice, Zone } from './types';
import {
  addSmartDevice,
  clearSmartDevices,
  removeSmartDevice,
  updateSmartDevicePosition,
} from './utils/devices';
import { updateZonePoint } from './utils/zones';

export interface ApartmentPlanSidebarRenderHandle {
  destroy(): void;
}

export interface ApartmentPlanRenderHandle {
  update(
    plan?: FloorPlan | null,
    devices?: SmartDevice[] | null,
    zones?: Zone[] | null,
  ): void;
  addDevice(device: SmartDevice): void;
  removeDevice(deviceId: string): void;
  clearDevices(): void;
  renderSidebar(container: HTMLElement): ApartmentPlanSidebarRenderHandle;
  destroy(): void;
}

export interface ApartmentPlanRenderOptions {
  showOpeningHitboxes?: boolean;
}

interface RendererState {
  plan: FloorPlan;
  devices: SmartDevice[];
  zones: Zone[];
  selectedDeviceId: string | null;
  selectedZoneId: string | null;
}

const createEmptyPlan = (): FloorPlan => ({
  walls: [],
  doors: [],
  windows: [],
  rooms: [],
});

const assertContainer = (container: HTMLElement): void => {
  if (container instanceof HTMLCanvasElement) {
    throw new TypeError(
      'renderApartmentPlan expects an empty HTML container, not a canvas element.',
    );
  }
};

export function renderApartmentPlan(
  container: HTMLElement,
  plan?: FloorPlan | null,
  devices?: SmartDevice[] | null,
  zones?: Zone[] | null,
  options: ApartmentPlanRenderOptions = {},
): ApartmentPlanRenderHandle {
  assertContainer(container);

  const root = createRoot(container);
  const sidebarRoots = new Set<Root>();
  let destroyed = false;
  let state: RendererState = {
    plan: plan ?? createEmptyPlan(),
    devices: devices ?? [],
    zones: zones ?? [],
    selectedDeviceId: null,
    selectedZoneId: null,
  };

  const ensureActive = () => {
    if (destroyed) {
      throw new Error('Cannot update a destroyed apartment plan renderer.');
    }
  };

  const renderStage = () => {
    root.render(
      createElement(FloorPlanStage, {
        plan: state.plan,
        devices: state.devices,
        zones: state.zones,
        showOpeningHitboxes: options.showOpeningHitboxes ?? false,
        selectedDeviceId: state.selectedDeviceId,
        selectedZoneId: state.selectedZoneId,
        onSelectDevice: selectDevice,
        onSelectZone: selectZone,
        onMoveDevice: moveDevice,
        onMoveZonePoint: moveZonePoint,
      }),
    );
  };

  const renderSidebar = (sidebarRoot: Root) => {
    const selectedZone =
      state.zones.find((zone) => zone.id === state.selectedZoneId) ?? null;

    sidebarRoot.render(
      createElement(DeviceSidebar, {
        devices: state.devices,
        selectedZone,
        selectedDeviceId: state.selectedDeviceId,
        onSelectDevice: selectDevice,
      }),
    );
  };

  function renderAll() {
    ensureActive();
    renderStage();
    sidebarRoots.forEach(renderSidebar);
  }

  function selectDevice(deviceId: string | null) {
    state = {
      ...state,
      selectedDeviceId: deviceId,
      selectedZoneId: deviceId ? null : state.selectedZoneId,
    };
    renderAll();
  }

  function selectZone(zoneId: string | null) {
    state = {
      ...state,
      selectedZoneId: zoneId,
      selectedDeviceId: zoneId ? null : state.selectedDeviceId,
    };
    renderAll();
  }

  function moveDevice(deviceId: string, position: SmartDevice['position']) {
    state = {
      ...state,
      devices: updateSmartDevicePosition(state.devices, deviceId, position),
    };
    renderAll();
  }

  function moveZonePoint(
    zoneId: string,
    pointIndex: number,
    point: Zone['points'][number],
  ) {
    state = {
      ...state,
      zones: updateZonePoint(
        state.zones,
        zoneId,
        pointIndex,
        point,
        state.plan.rooms,
      ),
    };
    renderAll();
  }

  renderAll();

  return {
    update: (
      nextPlan?: FloorPlan | null,
      nextDevices?: SmartDevice[] | null,
      nextZones?: Zone[] | null,
    ) => {
      ensureActive();
      const nextDeviceList = nextDevices ?? [];
      const nextZoneList = nextZones ?? [];

      state = {
        plan: nextPlan ?? createEmptyPlan(),
        devices: nextDeviceList,
        zones: nextZoneList,
        selectedDeviceId:
          state.selectedDeviceId &&
          nextDeviceList.some((device) => device.id === state.selectedDeviceId)
            ? state.selectedDeviceId
            : null,
        selectedZoneId:
          state.selectedZoneId &&
          nextZoneList.some((zone) => zone.id === state.selectedZoneId)
            ? state.selectedZoneId
            : null,
      };
      renderAll();
    },
    addDevice: (device: SmartDevice) => {
      ensureActive();
      state = { ...state, devices: addSmartDevice(state.devices, device) };
      renderAll();
    },
    removeDevice: (deviceId: string) => {
      ensureActive();
      state = {
        ...state,
        devices: removeSmartDevice(state.devices, deviceId),
        selectedDeviceId:
          state.selectedDeviceId === deviceId ? null : state.selectedDeviceId,
      };
      renderAll();
    },
    clearDevices: () => {
      ensureActive();
      state = {
        ...state,
        devices: clearSmartDevices(),
        selectedDeviceId: null,
      };
      renderAll();
    },
    renderSidebar: (sidebarContainer: HTMLElement) => {
      ensureActive();
      assertContainer(sidebarContainer);

      const sidebarRoot = createRoot(sidebarContainer);
      sidebarRoots.add(sidebarRoot);
      renderSidebar(sidebarRoot);

      return {
        destroy: () => {
          if (!sidebarRoots.delete(sidebarRoot)) {
            return;
          }

          sidebarRoot.unmount();
        },
      };
    },
    destroy: () => {
      if (destroyed) {
        return;
      }

      sidebarRoots.forEach((sidebarRoot) => {
        sidebarRoot.unmount();
      });
      sidebarRoots.clear();
      root.unmount();
      destroyed = true;
    },
  };
}

export function renderApartmentPlanSidebar(
  container: HTMLElement,
  renderer: ApartmentPlanRenderHandle,
): ApartmentPlanSidebarRenderHandle {
  return renderer.renderSidebar(container);
}
