export const SIMULATION_STORAGE_KEYS = [
  "simulation-floor",
  "simulation-devices",
  "simulation-trigger-device-ids",
  "simulation-plan-layout",
  "simulation-plan-dependencies",
  "planner-floor-json",
  "parsed-floor",
  "floor-json",
  "sim-devices",
  "selectedDevices",
  "selected-devices",
] as const;

export function clearSimulationStorage() {
  if (typeof window === "undefined") return;

  SIMULATION_STORAGE_KEYS.forEach((key) => {
    try {
      window.localStorage.removeItem(key);
    } catch {
      // Storage can be unavailable in some browser privacy modes.
    }
  });
}
