# Apartment UI

The application exposes an imperative renderer for integration into another web
interface.

```ts
import {
  renderApartmentPlan,
  type FloorPlan,
  type SmartDevice,
  type Zone,
} from 'smart-plan-demo';

const container = document.getElementById('apartment-plan');

if (container) {
  const renderer = renderApartmentPlan(
    container,
    plan as FloorPlan,
    devices as SmartDevice[],
    zones as Zone[],
  );

  renderer.addDevice(device);
  renderer.removeDevice(device.id);
  renderer.clearDevices();
  renderer.update(nextPlan, nextDevices, nextZones);
  renderer.destroy();
}
```

`addDevice` appends a new device. `removeDevice` deletes the device from the
renderer state and from the canvas. `clearDevices` removes all devices from the
renderer state and from the canvas. Device changes are not written back to JSON.

Zone data uses the same base shape as
`services/layout/internal/apartment/zone.go`. The flat frontend renderer also
accepts optional `room_id` to bind a zone to a room:

```ts
type Zone = {
  id: string;
  room_id?: string;
  points: Array<{ X: number; Y: number }>;
};
```

Rooms and zones are arbitrary polygons. Do not convert them to rectangles or
bounding boxes: preserve the ordered vertex list because the future zone editor
will update polygon vertices directly.

Zone vertices are draggable in the canvas. If `room_id` is provided, a zone
vertex can move only while the whole zone polygon remains inside that room
polygon. Edited zone coordinates are kept only in React state and are not
written back to JSON.

The target must be an empty HTML container with an explicit width and height.
The renderer creates and manages the Konva canvas inside that container.

All data arguments are optional. Calling `renderApartmentPlan(container)`
renders an empty viewport.

## Local verification

```bash
npm run lint
npm run build
npm run dev
```

Open the Vite URL to inspect the standalone application with the device
sidebar:

```text
http://127.0.0.1:5173/
```

Use `?embed=1` to verify the public `renderApartmentPlan(...)` function without
the sidebar:

```text
http://127.0.0.1:5173/?embed=1
```

Use `?polygon=1` to inspect the separate polygon test files:

```text
http://127.0.0.1:5173/?polygon=1
```

The polygon test data lives in:

- `src/polygon_plan.json`
- `src/polygon_zones.json`
- `src/polygon_devices.json`

Use `?empty=1` to verify the empty-data case:

```text
http://127.0.0.1:5173/?empty=1
```
