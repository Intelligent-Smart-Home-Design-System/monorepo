"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { ControlPanel } from "@/app/components/sim/ControlPanel";
import { ApartmentPlan } from "@/app/components/sim/ApartmentPlan";
import { EventConsole } from "@/app/components/sim/EventConsole";
import { Card } from "@/app/components/ui";

import {
  scenarios as MOCK_SCENARIOS,
  initialDevices,
  deviceMarkers,
  rooms as MOCK_ROOMS,
  type Scenario,
  type Device,
  type DeviceMarker,
  type LogEvent,
  type LogLevel,
} from "@/app/simulation/Mockdata";

type Status = "empty" | "loading" | "running" | "paused" | "error";
type Speed = number;
type Filter = "ALL" | LogLevel;
type RunMode = "parallel" | "sequence";

function speedToDelay(speed: Speed) {
  const s = Math.max(Number(speed) || 1, 0.1);
  return Math.round(700 / s);
}

export default function SimulationPage() {
  const scenarios = useMemo<Scenario[]>(() => MOCK_SCENARIOS, []);

  const [status, setStatus] = useState<Status>("empty");
  const [speed, setSpeed] = useState<Speed>(1);
  const [filter, setFilter] = useState<Filter>("ALL");
  const [search, setSearch] = useState("");
  const [autoscroll, setAutoscroll] = useState(true);

  const [selectedScenarioIds, setSelectedScenarioIds] = useState<string[]>([]);
  const [runMode, setRunMode] = useState<RunMode>("parallel");

  const selectedScenarios = useMemo(() => {
    return scenarios.filter((s) => selectedScenarioIds.includes(s.id));
  }, [scenarios, selectedScenarioIds]);

  const [events, setEvents] = useState<LogEvent[]>([]);
  const [activeNodes, setActiveNodes] = useState<string[]>([]);
  const [activeEdges, setActiveEdges] = useState<Array<[string, string]>>([]);
  const [lastEvent, setLastEvent] = useState<LogEvent | null>(null);
  const [runScenarios, setRunScenarios] = useState<Scenario[]>([]);
  const runStepRef = useRef(-1);
  const runSeqIndexRef = useRef(0);
  const runSeqStepRef = useRef(-1);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const [devicePositions, setDevicePositions] = useState<DeviceMarker[]>(deviceMarkers);

  const devicesForPlan = useMemo<Device[]>(() => {
    if (selectedScenarios.length === 0) return [];
    const ids = Array.from(new Set(selectedScenarios.flatMap((s) => s.chain)));

    return ids.map((id) => ({
      id,
      status: status === "running" && activeNodes.includes(id) ? "active" : "idle",
    }));
  }, [selectedScenarios, status, activeNodes]);

  function onMoveDevice(id: string, x: number, y: number) {
    setDevicePositions((prev) => {
      const idx = prev.findIndex((m) => m.id === id);
      if (idx === -1) return [...prev, { id, x, y }];
      const copy = prev.slice();
      copy[idx] = { ...copy[idx], x, y };
      return copy;
    });
  }

  const chainGroups = useMemo(() => {
    const palette = ["#38bdf8", "#f59e0b", "#22c55e", "#e879f9", "#f43f5e"];
    return selectedScenarios.map((s, i) => ({
      id: s.id,
      chain: s.chain,
      color: palette[i % palette.length],
    }));
  }, [selectedScenarios]);

  function nowTs() {
    return new Date().toLocaleTimeString("ru-RU", { hour12: false });
  }

  function onStart() {
    if (selectedScenarios.length === 0) return;

    setStatus("loading");
    setEvents([]);
    setLastEvent(null);
    setActiveNodes([]);
    setActiveEdges([]);
    setRunScenarios(selectedScenarios);
    runStepRef.current = -1;
    runSeqIndexRef.current = 0;
    runSeqStepRef.current = -1;

    const delay = speedToDelay(speed);

    setTimeout(() => {
      setStatus("running");
    }, delay);
  }

  function onPause() {
    setStatus((s) => (s === "running" ? "paused" : s));
  }

  function onStop() {
    setStatus("empty");
    setEvents([]);
    setLastEvent(null);
    setActiveNodes([]);
    setActiveEdges([]);
    setRunScenarios([]);
    runStepRef.current = -1;
    runSeqIndexRef.current = 0;
    runSeqStepRef.current = -1;
  }

  function onClear() {
    setEvents([]);
    setLastEvent(null);
  }

  useEffect(() => {
    if (status !== "running" || runScenarios.length === 0) {
      if (timerRef.current) clearInterval(timerRef.current);
      timerRef.current = null;
      return;
    }

    const delay = speedToDelay(speed);

    timerRef.current = setInterval(() => {
      if (runMode === "parallel") {
        const maxLen = Math.max(...runScenarios.map((s) => s.chain.length), 0);
        const next = runStepRef.current + 1;
        if (next >= maxLen) {
          setStatus("paused");
          return;
        }
        runStepRef.current = next;

        const nodes = runScenarios.map((s) => s.chain[next]).filter(Boolean);
        const edges = runScenarios
          .map((s) => (next > 0 ? [s.chain[next - 1], s.chain[next]] : null))
          .filter((e): e is [string, string] => !!e && !!e[0] && !!e[1])
          .map((e) => [e[0], e[1]] as [string, string]);

        setActiveNodes(nodes);
        setActiveEdges(edges);

        setEvents((prev: LogEvent[]) => {
          const appended = runScenarios.flatMap((s) =>
            s.chain[next]
              ? [
                  {
                    id: `${s.id}-${next}-${Date.now()}`,
                    ts: nowTs(),
                    level: "INFO" as const,
                    device: s.chain[next],
                    message: `Шаг ${next + 1}`,
                  } satisfies LogEvent,
                ]
              : []
          );
          const nextEvents = [...prev, ...appended];
          setLastEvent(appended[appended.length - 1] ?? prev[prev.length - 1] ?? null);
          return nextEvents;
        });
      } else {
        const currentScenario = runScenarios[runSeqIndexRef.current];
        if (!currentScenario) {
          setStatus("paused");
          return;
        }

        let nextStep = runSeqStepRef.current + 1;
        let nextScenarioIndex = runSeqIndexRef.current;

        if (nextStep >= currentScenario.chain.length) {
          nextScenarioIndex += 1;
          const nextScenario = runScenarios[nextScenarioIndex];
          if (!nextScenario) {
            setStatus("paused");
            return;
          }
          nextStep = 0;
        }

        const scenario = runScenarios[nextScenarioIndex];
        runSeqIndexRef.current = nextScenarioIndex;
        runSeqStepRef.current = nextStep;

        const node = scenario.chain[nextStep];
        const edge = nextStep > 0 ? [scenario.chain[nextStep - 1], scenario.chain[nextStep]] : null;

        setActiveNodes(node ? [node] : []);
        setActiveEdges(edge && edge[0] && edge[1] ? [[edge[0], edge[1]]] : []);

        if (node) {
          const ev: LogEvent = {
            id: `${scenario.id}-${nextStep}-${Date.now()}`,
            ts: nowTs(),
            level: "INFO",
            device: node,
            message: `Шаг ${nextStep + 1} • ${scenario.title}`,
          };
          setEvents((prev) => [...prev, ev]);
          setLastEvent(ev);
        }
      }
    }, delay);

    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
      timerRef.current = null;
    };
  }, [status, speed, runMode, runScenarios]);

  return (
    <main
      className="min-h-screen w-full text-neutral-100"
      style={{
        background:
          "radial-gradient(1200px 600px at 20% 0%, rgba(120,119,198,0.16), transparent 60%), radial-gradient(900px 500px at 80% 10%, rgba(56,189,248,0.10), transparent 60%), #0b0c0f",
      }}
    >
      <div className="mx-auto w-full max-w-6xl px-6 py-5">
        <Card className="p-4">
          <ControlPanel
            scenarios={scenarios}
            selectedScenarioIds={selectedScenarioIds}
            onSelectScenarios={setSelectedScenarioIds}
            runMode={runMode}
            onSetRunMode={setRunMode}
            status={status}
            speed={speed}
            filter={filter}
            search={search}
            autoscroll={autoscroll}
            onStart={onStart}
            onPause={onPause}
            onStop={onStop}
            onClear={onClear}
            onSetSpeed={setSpeed}
            onSetFilter={setFilter}
            onSetSearch={setSearch}
            onToggleAutoscroll={() => setAutoscroll((v) => !v)}
          />

          <div className="mt-2 grid grid-cols-1 gap-5 lg:grid-cols-[1fr_320px]">
            <div className="pl-[1cm]">
              <ApartmentPlan
                rooms={MOCK_ROOMS}
                markers={devicePositions}
                devices={devicesForPlan}
                chains={chainGroups}
                activeNodes={activeNodes}
                activeEdges={activeEdges}
                lastEvent={events.length ? events[events.length - 1] : null}
                onMoveDevice={onMoveDevice}
              />

              <div className="mt-5">
                <EventConsole title="Консоль событий" events={events} filter={filter} search={search} autoscroll={autoscroll} />
              </div>
            </div>

            <section className="glass-card p-5">
              <div className="mb-3 text-3xl font-semibold text-white">Устройства в симуляции</div>
              <div className="flex flex-col gap-4 pl-4">
                {devicesForPlan.map((d) => (
                  <div
                    key={d.id}
                    className="flex items-center justify-between rounded-xl border border-white/10 bg-black/20 px-4 py-3"
                  >
                    <div className="font-mono text-lg leading-6 text-white pl-3">{d.id}</div>
                    <div className="text-lg text-white/70">{d.status}</div>
                  </div>
                ))}
              </div>
            </section>
          </div>
        </Card>
      </div>
    </main>
  );
}
