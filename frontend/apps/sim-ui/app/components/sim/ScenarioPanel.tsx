"use client";

import { useMemo, useState } from "react";
import { Search } from "lucide-react";
import type { Scenario } from "@/app/simulation/Mockdata";

type Props = {
  scenarios: Scenario[];
  deviceNames: Record<string, string>;
  selectedScenarioId: string | null;
  onSelectScenario: (scenarioId: string) => void;
};

export function ScenarioPanel({ scenarios, deviceNames, selectedScenarioId, onSelectScenario }: Props) {
  const [query, setQuery] = useState("");

  const filteredScenarios = useMemo(() => {
    const normalizedQuery = query.trim().toLocaleLowerCase("ru");
    if (!normalizedQuery) return scenarios;

    return scenarios.filter((scenario) => {
      const searchableText = [
        scenario.title,
        scenario.description,
        scenario.id,
        ...scenario.chain,
        ...scenario.chain.map((deviceID) => deviceNames[deviceID]),
      ]
        .filter(Boolean)
        .join(" ")
        .toLocaleLowerCase("ru");
      return searchableText.includes(normalizedQuery);
    });
  }, [deviceNames, query, scenarios]);

  return (
    <section className="scenario-panel" aria-labelledby="scenario-panel-title">
      <div className="scenario-panel-header">
        <div>
          <h2 id="scenario-panel-title">Сценарии</h2>
          <span>{scenarios.length}</span>
        </div>
        <label className="scenario-panel-search">
          <Search size={19} strokeWidth={2} aria-hidden="true" />
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Поиск сценария"
            aria-label="Поиск сценария"
          />
        </label>
      </div>

      <div className="scenario-panel-list">
        {filteredScenarios.length ? (
          filteredScenarios.map((scenario) => (
            <button
              className={`scenario-panel-item${selectedScenarioId === scenario.id ? " scenario-panel-item-selected" : ""}`}
              key={scenario.id}
              type="button"
              aria-pressed={selectedScenarioId === scenario.id}
              onClick={() => onSelectScenario(scenario.id)}
            >
              <strong>{scenario.title}</strong>
              <span>{selectedScenarioId === scenario.id ? "выбрано" : "активно"}</span>
            </button>
          ))
        ) : (
          <div className="scenario-panel-empty">
            {scenarios.length ? "Сценарии по запросу не найдены" : "Модуль расстановки не вернул сценарии"}
          </div>
        )}
      </div>
    </section>
  );
}
