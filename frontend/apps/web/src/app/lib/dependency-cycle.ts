export type DependencyGraph = Record<string, string[]>;

type DependencyRules = {
  triggers?: Record<string, { triggers?: string[] }>;
};

export function dependencyGraphForDeviceTypes(deviceTypes: string[], rules: DependencyRules): DependencyGraph {
  const selected = new Set(deviceTypes.filter(Boolean));
  const graph: DependencyGraph = {};

  Array.from(selected)
    .sort()
    .forEach((deviceType) => {
      graph[deviceType] = (rules.triggers?.[deviceType]?.triggers ?? []).filter((target) => selected.has(target));
    });

  return graph;
}

export function findDependencyCycle(graph: DependencyGraph | null | undefined): string[] {
  if (!graph) return [];

  const state = new Map<string, "visiting" | "visited">();
  const path: string[] = [];
  const pathIndex = new Map<string, number>();

  const visit = (node: string): string[] => {
    state.set(node, "visiting");
    pathIndex.set(node, path.length);
    path.push(node);

    const targets = [...(graph[node] ?? [])].sort();
    for (const target of targets) {
      if (state.get(target) === "visiting") {
        const start = pathIndex.get(target) ?? 0;
        return [...path.slice(start), target];
      }
      if (!state.has(target)) {
        const cycle = visit(target);
        if (cycle.length) return cycle;
      }
    }

    path.pop();
    pathIndex.delete(node);
    state.set(node, "visited");
    return [];
  };

  for (const node of Object.keys(graph).sort()) {
    if (!state.has(node)) {
      const cycle = visit(node);
      if (cycle.length) return cycle;
    }
  }

  return [];
}

export function dependencyCycleMessage(cycle: string[], names: Record<string, string> = {}) {
  if (!cycle.length) return "";
  const chain = cycle.map((id) => names[id] ?? id).join(" → ");
  return `Обнаружена циклическая зависимость между устройствами: ${chain}. Исправьте конфигурацию.`;
}
