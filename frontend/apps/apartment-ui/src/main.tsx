import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import App from './App';

import rawDevicesData from './devices.json';
import rawPlanData from './image_apartment_plan.json';
import rawPolygonDevicesData from './polygon_devices.json';
import rawPolygonPlanData from './polygon_plan.json';
import rawPolygonZonesData from './polygon_zones.json';
import rawResponsePlanData from './response.json';
import { renderApartmentPlan } from './renderApartmentPlan';
import type { FloorPlan, SmartDevice, Zone } from './types';
import rawZonesData from './zones.json';

const container = document.getElementById('root');

if (!container) {
  throw new Error('Root container was not found.');
}

const params = new URLSearchParams(window.location.search);
const shouldUseEmbedRenderer = params.has('embed') || params.has('empty');
const shouldUsePolygonExample = params.has('polygon');
const shouldUseResponseExample = params.has('response');
const activePlan = shouldUseResponseExample
  ? (rawResponsePlanData as unknown as FloorPlan)
  : shouldUsePolygonExample
  ? (rawPolygonPlanData as unknown as FloorPlan)
  : (rawPlanData as unknown as FloorPlan);
const activeDevices = shouldUseResponseExample
  ? []
  : shouldUsePolygonExample
  ? (rawPolygonDevicesData as unknown as SmartDevice[])
  : (rawDevicesData as unknown as SmartDevice[]);
const activeZones = shouldUseResponseExample
  ? []
  : shouldUsePolygonExample
  ? (rawPolygonZonesData as unknown as Zone[])
  : (rawZonesData as unknown as Zone[]);

if (shouldUseEmbedRenderer) {
  const renderer = params.has('empty')
    ? renderApartmentPlan(container)
    : renderApartmentPlan(
        container,
        activePlan,
        activeDevices,
        activeZones,
      );

  if (import.meta.hot) {
    import.meta.hot.dispose(() => {
      renderer.destroy();
    });
  }
} else {
  const root = createRoot(container);

  root.render(
    <StrictMode>
      <App
        key={
          shouldUseResponseExample
            ? 'response-example'
            : shouldUsePolygonExample
              ? 'polygon-example'
              : 'default-example'
        }
        plan={activePlan}
        devices={activeDevices}
        zones={activeZones}
      />
    </StrictMode>,
  );

  if (import.meta.hot) {
    import.meta.hot.dispose(() => {
      root.unmount();
    });
  }
}
