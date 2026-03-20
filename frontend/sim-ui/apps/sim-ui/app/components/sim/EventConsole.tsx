"use client";

import { useEffect, useMemo, useRef } from "react";
import type { LogEvent, LogLevel } from "@/app/simulation/Mockdata";

type Filter = "ALL" | LogLevel;

type Props = {
  title: string;
  events: LogEvent[];

  filter?: Filter;
  search?: string;
  autoscroll?: boolean;
};

export function EventConsole({ title, events, filter = "ALL", search = "", autoscroll = true }: Props) {
  const boxRef = useRef<HTMLDivElement | null>(null);

  const filtered = useMemo(() => {
    const q = (search ?? "").trim().toLowerCase();

    return (events ?? []).filter((e) => {
      if (filter !== "ALL" && e.level !== filter) return false;
      if (!q) return true;
      return `${e.device} ${e.message}`.toLowerCase().includes(q);
    });
  }, [events, filter, search]);

  useEffect(() => {
    if (!autoscroll) return;
    const el = boxRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [filtered, autoscroll]);

  return (
    <section className="glass-card p-5">
      <div className="text-3xl font-semibold text-white mb-4">{title}</div>

      <div
        ref={boxRef}
        className={[
          "rounded-2xl border border-white/10 bg-black/30",
          "max-h-[260px] overflow-auto",
          "p-4",
          "font-mono text-lg",
        ].join(" ")}
      >
        {filtered.length === 0 ? (
          <div className="text-white/60 text-lg">—</div>
        ) : (
          <div className="flex flex-col gap-2">
            {filtered.map((e) => (
              <div key={e.id} className="flex gap-3 items-start">
                <span className="text-white/60 w-28">{e.ts}</span>
                <span className="text-white/80">[{e.level}]</span>
                <span className="text-white/90">{e.device}</span>
                <span className="text-white/50">—</span>
                <span className="text-white break-words">{e.message}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
