"use client";

import { useState } from "react";
import type { Scenario } from "@/app/simulation/Mockdata";
type Speed = number;
type DeviceType =
  | "pir"
  | "mmwave"
  | "door"
  | "leak"
  | "smoke"
  | "co"
  | "gas"
  | "temp"
  | "humidity"
  | "lux"
  | "noise"
  | "co2"
  | "voc"
  | "pm25"
  | "pressure"
  | "floor_temp"
  | "freeze"
  | "current"
  | "water_flow"
  | "camera"
  | "lock"
  | "other";

type Props = {
  scenarios: Scenario[];
  placedDeviceIds: string[];
  availableDeviceIds: string[];
  deviceNames: Record<string, string>;
  onPlaceDevice: (id: string) => void;
  manualPlacementOnly?: boolean;

  status: "empty" | "loading" | "running" | "paused" | "error";
  speed: Speed;

  onStart: () => void;
  onPause: () => void;
  onResume: () => void;
  onStop: () => void;
  onClearDevices: () => void;
  devicesLocked: boolean;

  onSetSpeed: (v: Speed) => void;
};

export function ControlPanel(props: Props) {
  const {
    scenarios,
    placedDeviceIds,
    availableDeviceIds,
    deviceNames,
    onPlaceDevice,
    manualPlacementOnly = false,
    status,
    speed,
    onStart,
    onPause,
    onResume,
    onStop,
    onClearDevices,
    devicesLocked,
    onSetSpeed,
  } = props;

  const [selectedDeviceTypes, setSelectedDeviceTypes] = useState<DeviceType[]>([]);

  const canStart = status === "empty" || status === "error";
  const canResume = status === "paused";
  const canPause = status === "running";
  const canStop = status === "running" || status === "paused" || status === "loading";
  const isBusy = status === "loading";

  function btnClass(active?: boolean) {
    return ["bubble apple-button", active ? "bubble-active" : ""].join(" ");
  }

  function pillClass(active?: boolean) {
    return ["bubble apple-pill pill", active ? "pill-active" : ""].join(" ");
  }

  const deviceTypeLabels: Record<DeviceType, string> = {
    pir: "Движение",
    mmwave: "Присутствие",
    door: "Открытие",
    leak: "Протечка",
    smoke: "Дым",
    co: "CO",
    gas: "Газ",
    temp: "Температура",
    humidity: "Влажность",
    lux: "Освещённость",
    noise: "Шум",
    co2: "CO₂",
    voc: "VOC",
    pm25: "PM2.5",
    pressure: "Давление",
    floor_temp: "Пол",
    freeze: "Замерзание",
    current: "Ток",
    water_flow: "Вода",
    camera: "Камера",
    lock: "Замок",
    other: "Другое",
  };

  function deviceTypeForId(id: string): DeviceType {
    const key = id.toLowerCase();
    if (key.includes("motion")) return "pir";
    if (key.includes("mmwave")) return "mmwave";
    if (key.includes("door") || key.includes("window")) return "door";
    if (key.includes("leak")) return "leak";
    if (key.includes("smoke")) return "smoke";
    if (key.includes("co2")) return "co2";
    if (key.includes("co")) return "co";
    if (key.includes("gas")) return "gas";
    if (key.includes("temp")) return "temp";
    if (key.includes("humidity")) return "humidity";
    if (key.includes("lux")) return "lux";
    if (key.includes("noise") || key.includes("sound")) return "noise";
    if (key.includes("voc")) return "voc";
    if (key.includes("pm25")) return "pm25";
    if (key.includes("pressure")) return "pressure";
    if (key.includes("floor")) return "floor_temp";
    if (key.includes("freeze")) return "freeze";
    if (key.includes("current") || key.includes("power")) return "current";
    if (key.includes("water") || key.includes("flow")) return "water_flow";
    if (key.includes("camera")) return "camera";
    if (key.includes("lock")) return "lock";
    return "other";
  }

  const availableDeviceTypes = (() => {
    const set = new Set<DeviceType>();
    scenarios.forEach((s) => s.chain.forEach((id) => set.add(deviceTypeForId(id))));
    availableDeviceIds.forEach((id) => set.add(deviceTypeForId(id)));
    return Array.from(set);
  })();

  const availableDevices = (() => {
    const byTypes =
      selectedDeviceTypes.length === 0
        ? scenarios
        : scenarios.filter((s) => s.chain.some((id) => selectedDeviceTypes.includes(deviceTypeForId(id))));
    const set = new Set<string>();
    byTypes.forEach((s) => s.chain.forEach((id) => set.add(id)));
    availableDeviceIds.forEach((id) => {
      if (selectedDeviceTypes.length === 0 || selectedDeviceTypes.includes(deviceTypeForId(id))) set.add(id);
    });
    return Array.from(set);
  })();

  const placedSet = new Set(placedDeviceIds);

  const statusLabel =
    status === "loading"
      ? "Загрузка"
      : status === "running"
      ? "Выполняется"
      : status === "paused"
      ? "Пауза"
      : status === "error"
      ? "Ошибка"
      : "Ожидает запуска";

  return (
    <section className="control-panel">
      <div className="control-header">
        <div>
          <div className="control-kicker">Smart Home</div>
          <h1>Симуляция</h1>
        </div>
        <div className={`status-chip status-${status}`}>{statusLabel}</div>
      </div>

      <div className="control-grid">
        <div className="control-section control-section-wide device-palette-section">
          <div className="section-title-row">
            <div className="section-label">Устройства</div>
            <button
              type="button"
              className="bubble apple-pill clear-devices-button"
              disabled={!placedDeviceIds.length || devicesLocked}
              onClick={onClearDevices}
            >
              Очистить план
            </button>
          </div>
          <div className="device-type-row no-scrollbar">
            <button type="button" className={pillClass(selectedDeviceTypes.length === 0)} onClick={() => setSelectedDeviceTypes([])}>
              Все типы
            </button>
            {availableDeviceTypes.map((t) => (
              <button
                key={t}
                type="button"
                className={pillClass(selectedDeviceTypes.includes(t))}
                onClick={() => {
                  const next = selectedDeviceTypes.includes(t)
                    ? selectedDeviceTypes.filter((x) => x !== t)
                    : [...selectedDeviceTypes, t];
                  setSelectedDeviceTypes(next);
                }}
              >
                {deviceTypeLabels[t]}
              </button>
            ))}
          </div>
          <div className="device-palette">
            {availableDevices.map((id) => {
              const placed = placedSet.has(id);
              const name = deviceNames[id] || id;
              return (
                <div
                  key={id}
                  role="button"
                  tabIndex={0}
                  draggable={!devicesLocked}
                  aria-disabled={devicesLocked}
                  className={`device-palette-item${placed ? " device-palette-item-placed" : ""}${devicesLocked ? " device-palette-item-locked" : ""}`}
                  onDragStart={(e) => {
                    if (devicesLocked) {
                      e.preventDefault();
                      return;
                    }
                    e.dataTransfer.setData("application/x-sim-device-id", id);
                    e.dataTransfer.setData("text/plain", id);
                    e.dataTransfer.effectAllowed = "copyMove";
                  }}
                  onClick={() => {
                    if (!devicesLocked && !placed && !manualPlacementOnly) onPlaceDevice(id);
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      if (!devicesLocked && !placed && !manualPlacementOnly) onPlaceDevice(id);
                    }
                  }}
                  title={
                    placed
                      ? `${name} (${id}) уже на плане`
                      : manualPlacementOnly
                      ? `${name} (${id}): перетащите устройство на план`
                      : `${name} (${id}): перетащи на план или кликни, чтобы поставить автоматически`
                  }
                >
                  <span>{name}</span>
                  <small>{placed ? "на плане" : manualPlacementOnly ? "перетащить на план" : "перетащить / клик"}</small>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div className="control-toolbar">
        <div className="action-cluster">
          <button
            type="button"
            className={btnClass(canStart || canResume)}
            disabled={(!canStart && !canResume) || isBusy}
            onClick={canResume ? onResume : onStart}
            data-testid={canResume ? "simulation-resume" : "simulation-start"}
          >
            {canResume ? "Продолжить" : "Запуск"}
          </button>
          <button type="button" className={btnClass(false)} disabled={!canPause || isBusy} onClick={onPause}>
            Пауза
          </button>
          <button type="button" className={btnClass(false)} disabled={!canStop} onClick={onStop}>
            Стоп
          </button>
        </div>

        <div className="speed-control">
          <input
            type="range"
            min={0.5}
            max={5}
            step={0.1}
            value={speed}
            onChange={(e) => onSetSpeed(parseFloat(e.target.value))}
            className="small-range"
          />
          <span>{Number(speed).toFixed(1)}x</span>
        </div>
      </div>
    </section>
  );
}
