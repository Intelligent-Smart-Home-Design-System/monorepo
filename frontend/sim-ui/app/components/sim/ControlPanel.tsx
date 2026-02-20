"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import type { LogLevel, Scenario } from "@/app/simulation/Mockdata";

type Filter = "ALL" | LogLevel;
type Speed = number;
type RunMode = "parallel" | "sequence";
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
  selectedScenarioIds: string[];
  onSelectScenarios: (ids: string[]) => void;
  runMode: RunMode;
  onSetRunMode: (v: RunMode) => void;

  status: "empty" | "loading" | "running" | "paused" | "error";
  speed: Speed;
  filter: Filter;
  search: string;
  autoscroll: boolean;

  onStart: () => void;
  onPause: () => void;
  onStop: () => void;
  onClear: () => void;

  onSetSpeed: (v: Speed) => void;
  onSetFilter: (v: Filter) => void;
  onSetSearch: (v: string) => void;
  onToggleAutoscroll: () => void;
};

export function ControlPanel(props: Props) {
  const {
    scenarios,
    selectedScenarioIds,
    onSelectScenarios,
    runMode,
    onSetRunMode,
    status,
    speed,
    filter,
    search,
    autoscroll,
    onStart,
    onPause,
    onStop,
    onClear,
    onSetSpeed,
    onSetFilter,
    onSetSearch,
  } = props;

  const [scenarioQuery, setScenarioQuery] = useState("");
  const [modeOpen, setModeOpen] = useState(false);
  const [selectedDeviceTypes, setSelectedDeviceTypes] = useState<DeviceType[]>([]);
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);
  const modeRef = useRef<HTMLDivElement | null>(null);

  const selectedTitles = useMemo(() => {
    return scenarios.filter((s) => selectedScenarioIds.includes(s.id)).map((s) => s.title);
  }, [scenarios, selectedScenarioIds]);

  const filteredScenarios = useMemo(() => {
    const q = scenarioQuery.trim().toLowerCase();
    const byTypes =
      selectedDeviceTypes.length === 0
        ? scenarios
        : scenarios.filter((s) => s.chain.some((id) => selectedDeviceTypes.includes(deviceTypeForId(id))));
    const byDevices =
      selectedDevices.length === 0
        ? byTypes
        : byTypes.filter((s) => s.chain.some((id) => selectedDevices.includes(id)));
    if (!q) return byDevices.slice(0, 50);
    return byDevices.filter((s) => s.title.toLowerCase().includes(q)).slice(0, 50);
  }, [scenarios, scenarioQuery, selectedDeviceTypes, selectedDevices]);

  useEffect(() => {
    const onDocMouseDown = (e: MouseEvent) => {
      if (!modeRef.current) return;
      if (!modeRef.current.contains(e.target as Node)) setModeOpen(false);
    };

    document.addEventListener("mousedown", onDocMouseDown);
    return () => document.removeEventListener("mousedown", onDocMouseDown);
  }, []);

  const canStart = selectedScenarioIds.length > 0 && (status === "empty" || status === "paused" || status === "error");
  const canPause = status === "running";
  const canStop = status === "running" || status === "paused" || status === "loading";
  const isBusy = status === "loading";

  function btnClass(active?: boolean) {
    return ["bubble px-4 py-3 text-xl text-neutral-100", active ? "bubble-active" : ""].join(" ");
  }

  function pillClass(active?: boolean) {
    return ["bubble px-3 py-2 text-lg text-neutral-100 pill", active ? "pill-active" : ""].join(" ");
  }


  const modeLabels: Record<RunMode, string> = {
    parallel: "Одновременно",
    sequence: "По очереди",
  };

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

  const availableDeviceTypes = useMemo(() => {
    const set = new Set<DeviceType>();
    scenarios.forEach((s) => s.chain.forEach((id) => set.add(deviceTypeForId(id))));
    return Array.from(set);
  }, [scenarios]);

  const availableDevices = useMemo(() => {
    const byTypes =
      selectedDeviceTypes.length === 0
        ? scenarios
        : scenarios.filter((s) => s.chain.some((id) => selectedDeviceTypes.includes(deviceTypeForId(id))));
    const set = new Set<string>();
    byTypes.forEach((s) => s.chain.forEach((id) => set.add(id)));
    return Array.from(set);
  }, [scenarios, selectedDeviceTypes]);

  return (
    <section className="p-4 relative z-50">
      <div className="text-4xl font-semibold text-white mb-4">Симуляция</div>

      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-2 min-w-[150px] flex-1 w-full md:w-auto relative z-50">
          <div className="glass-card p-3 relative z-50">
              <div className="sticky top-0 z-10 -mx-3 -mt-3 px-3 pt-3 pb-2 bg-slate-900/80 backdrop-blur-lg rounded-t-2xl">
                <div className="flex flex-col gap-3">
                  <div className="flex items-center gap-2 overflow-x-auto no-scrollbar">
                    <button
                      type="button"
                      className={pillClass(false) + " shrink-0 text-base"}
                      onClick={() => {
                        setSelectedDeviceTypes([]);
                        setSelectedDevices([]);
                      }}
                    >
                      Сброс
                    </button>
                    <button
                      type="button"
                      className={pillClass(selectedDeviceTypes.length === 0) + " shrink-0 text-base"}
                      style={
                        selectedDeviceTypes.length === 0
                          ? { background: "rgba(255,255,255,0.22)", borderColor: "rgba(255,255,255,0.4)" }
                          : undefined
                      }
                      onClick={() => {
                        setSelectedDeviceTypes([]);
                        setSelectedDevices([]);
                      }}
                    >
                      Все типы
                    </button>
                    {availableDeviceTypes.map((t) => (
                      <button
                        key={t}
                        type="button"
                        className={[
                          pillClass(selectedDeviceTypes.includes(t)) + " shrink-0 text-base",
                          selectedDeviceTypes.includes(t) ? "pill-active" : "",
                        ].join(" ")}
                        style={
                          selectedDeviceTypes.includes(t)
                            ? { background: "rgba(255,255,255,0.22)", borderColor: "rgba(255,255,255,0.4)" }
                            : undefined
                        }
                        onClick={() => {
                          const next = selectedDeviceTypes.includes(t)
                            ? selectedDeviceTypes.filter((x) => x !== t)
                            : [...selectedDeviceTypes, t];
                          setSelectedDeviceTypes(next);
                          setSelectedDevices([]);
                        }}
                      >
                        {deviceTypeLabels[t]}
                      </button>
                    ))}
                  </div>

                  <div className="flex items-center gap-2 overflow-x-auto no-scrollbar">
                    <button
                      type="button"
                      className={pillClass(false) + " shrink-0 text-base"}
                      onClick={() => setSelectedDevices([])}
                    >
                      Сброс
                    </button>
                    <button
                      type="button"
                      className={pillClass(selectedDevices.length === 0) + " shrink-0 text-base"}
                      style={
                        selectedDevices.length === 0
                          ? { background: "rgba(255,255,255,0.22)", borderColor: "rgba(255,255,255,0.4)" }
                          : undefined
                      }
                      onClick={() => setSelectedDevices([])}
                    >
                      Все устройства
                    </button>
                    {availableDevices.map((id) => (
                      <button
                        key={id}
                        type="button"
                        className={[
                          pillClass(selectedDevices.includes(id)) + " shrink-0 text-base",
                          selectedDevices.includes(id) ? "pill-active" : "",
                        ].join(" ")}
                        style={
                          selectedDevices.includes(id)
                            ? { background: "rgba(255,255,255,0.22)", borderColor: "rgba(255,255,255,0.4)" }
                            : undefined
                        }
                        onClick={() => {
                          const next = selectedDevices.includes(id)
                            ? selectedDevices.filter((x) => x !== id)
                            : [...selectedDevices, id];
                          setSelectedDevices(next);
                        }}
                      >
                        {id}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              <div className="mt-3 flex items-center gap-2 overflow-x-auto no-scrollbar pb-2">
                <button
                  type="button"
                  className={pillClass(false) + " shrink-0 text-base"}
                  onClick={() => onSelectScenarios([])}
                >
                  Сброс
                </button>
                {filteredScenarios.map((s) => (
                  <button
                    type="button"
                    key={s.id}
                    className={[
                      pillClass(selectedScenarioIds.includes(s.id)) + " shrink-0 text-base",
                      selectedScenarioIds.includes(s.id) ? "pill-active" : "",
                    ].join(" ")}
                    style={
                      selectedScenarioIds.includes(s.id)
                        ? { background: "rgba(255,255,255,0.22)", borderColor: "rgba(255,255,255,0.4)" }
                        : undefined
                    }
                    onClick={() => {
                      const next = selectedScenarioIds.includes(s.id)
                        ? selectedScenarioIds.filter((id) => id !== s.id)
                        : [...selectedScenarioIds, s.id];
                      onSelectScenarios(next);
                    }}
                  >
                    {s.title}
                    {selectedScenarioIds.includes(s.id) ? ` #${selectedScenarioIds.indexOf(s.id) + 1}` : ""}
                  </button>
                ))}
              </div>
            </div>
        </div>

        <div className="flex items-center gap-3 flex-wrap">
          <button
            type="button"
            className={btnClass(canStart)}
            disabled={!canStart || isBusy}
            onClick={onStart}
          >
            Запуск
          </button>

          <button
            type="button"
            className={btnClass(false)}
            disabled={!canPause || isBusy}
            onClick={onPause}
          >
            Пауза
          </button>

          <button
            type="button"
            className={btnClass(false)}
            disabled={!canStop}
            onClick={onStop}
          >
            Стоп
          </button>
          <div className="flex items-center gap-2">
            <input
              type="range"
              min={0.5}
              max={5}
              step={0.1}
              value={speed}
              onChange={(e) => onSetSpeed(parseFloat(e.target.value))}
              className="small-range"
            />
            <div className="text-xl text-neutral-100">{Number(speed).toFixed(1)}×</div>
          </div>

          <button type="button" className={btnClass(false)} onClick={onClear}>
            Очистить
          </button>
        </div>
      </div>

      {status !== "empty" && (
        <div className="text-xl text-white/70 mt-3">
          <span className="text-white/90">
            {status === "loading"
              ? "загрузка"
              : status === "running"
              ? "выполняется"
              : status === "paused"
              ? "пауза"
              : "ошибка"}
          </span>
        </div>
      )}
    </section>
  );
}
