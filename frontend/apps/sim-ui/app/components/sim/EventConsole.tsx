"use client";

import { useEffect, useMemo, useRef } from "react";
import type { LogEvent, LogLevel } from "@/app/simulation/Mockdata";

type Filter = "ALL" | LogLevel;

type Props = {
  title: string;
  events: LogEvent[];

  filter?: Filter;
  search?: string;
};

export function EventConsole({ title, events, filter = "ALL", search = "" }: Props) {
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
    const el = boxRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [filtered]);

  return (
    <section className="glass-card event-console">
      <div className="event-console-title">{title}</div>

      <div
        ref={boxRef}
        className="console-surface event-console-box"
      >
        {filtered.length === 0 ? (
          <div className="event-console-empty">Событий пока нет</div>
        ) : (
          <div className="event-log-list">
            {filtered.map((e) => (
              <div key={e.id} className={`event-log-row event-log-${e.level.toLowerCase()}`}>
                <span className="event-log-time">{e.ts}</span>
                <span className="event-log-level">[{e.level}]</span>
                <span className="event-log-device">{e.device}</span>
                <span className="event-log-message">{e.message}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
