"use client";

import { useEffect, useMemo, useRef } from "react";
import { Check, ChevronDown, Search, Trash2 } from "lucide-react";
import type { LogEvent, LogLevel } from "@/app/simulation/Mockdata";

type Filter = "ALL" | LogLevel;

type Props = {
  title: string;
  events: LogEvent[];
  deviceNames?: Record<string, string>;

  filter?: Filter;
  search?: string;
  onClear: () => void;
  onSetFilter: (filter: Filter) => void;
  onSetSearch: (search: string) => void;
};

const filterLabels: Record<Filter, string> = {
  ALL: "Все логи",
  INFO: "INFO",
  WARNING: "WARNING",
  ERROR: "ERROR",
};

export function EventConsole({
  title,
  events,
  deviceNames = {},
  filter = "ALL",
  search = "",
  onClear,
  onSetFilter,
  onSetSearch,
}: Props) {
  const boxRef = useRef<HTMLDivElement | null>(null);
  const filterMenuRef = useRef<HTMLDetailsElement | null>(null);

  const filtered = useMemo(() => {
    const q = (search ?? "").trim().toLowerCase();

    return (events ?? []).filter((e) => {
      if (filter !== "ALL" && e.level !== filter) return false;
      if (!q) return true;
      return `${deviceNames[e.device] ?? ""} ${e.device} ${e.message}`.toLowerCase().includes(q);
    });
  }, [deviceNames, events, filter, search]);

  useEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [filtered]);

  return (
    <section className="glass-card event-console">
      <div className="event-console-header">
        <div className="event-console-title">{title}</div>
        <div className="event-console-tools">
          <button type="button" className="event-console-clear" onClick={onClear}>
            <Trash2 size={17} strokeWidth={2} aria-hidden="true" />
            <span>Очистить</span>
          </button>

          <details
            ref={filterMenuRef}
            className="event-filter-menu"
            onBlur={(event) => {
              if (!event.currentTarget.contains(event.relatedTarget)) event.currentTarget.removeAttribute("open");
            }}
          >
            <summary>
              <span>{filterLabels[filter]}</span>
              <ChevronDown size={17} strokeWidth={2.2} aria-hidden="true" />
            </summary>
            <div className="event-filter-options">
              {(Object.keys(filterLabels) as Filter[]).map((option) => (
                <button
                  type="button"
                  key={option}
                  className={filter === option ? "event-filter-option-active" : ""}
                  onClick={() => {
                    onSetFilter(option);
                    filterMenuRef.current?.removeAttribute("open");
                  }}
                >
                  <span>{filterLabels[option]}</span>
                  {filter === option && <Check size={16} strokeWidth={2.4} aria-hidden="true" />}
                </button>
              ))}
            </div>
          </details>

          <label className="event-console-search">
            <Search size={18} strokeWidth={2} aria-hidden="true" />
            <input
              type="search"
              value={search}
              onChange={(event) => onSetSearch(event.target.value)}
              placeholder="Поиск по событиям"
              aria-label="Поиск по событиям"
            />
          </label>
        </div>
      </div>

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
                <span className="event-log-device" title={deviceNames[e.device] ? e.device : undefined}>
                  {deviceNames[e.device] ?? e.device}
                </span>
                <span className="event-log-message">{e.message}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
