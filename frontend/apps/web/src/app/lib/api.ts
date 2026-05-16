import type {
  ApiCreatePlanRequest,
  ApiCreatePlanResponse,
  ApiDeviceType,
  ApiEcosystem,
  ApiErrorResponse,
  ApiHomePlan,
  ApiPlanStatus,
  ApiPlanSummary,
  ApiPreset,
} from "./types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "") ?? "";

function buildUrl(path: string) {
  return `${API_BASE_URL}${path}`;
}

async function requestJson<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(buildUrl(path), {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    cache: "no-store",
  });

  if (!response.ok) {
    let error: ApiErrorResponse | null = null;
    try {
      error = (await response.json()) as ApiErrorResponse;
    } catch {
      error = null;
    }
    throw new Error(error?.message ?? `Request failed with status ${response.status}`);
  }

  return (await response.json()) as T;
}

export const api = {
  listPlans() {
    return requestJson<ApiPlanSummary[]>("/api/v1/plans");
  },
  listEcosystems() {
    return requestJson<ApiEcosystem[]>("/api/v1/ecosystems");
  },
  listPresets() {
    return requestJson<ApiPreset[]>("/api/v1/presets");
  },
  listDeviceTypes() {
    return requestJson<ApiDeviceType[]>("/api/v1/device-types");
  },
  createPlan(payload: ApiCreatePlanRequest) {
    return requestJson<ApiCreatePlanResponse>("/api/v1/plans", {
      method: "POST",
      body: JSON.stringify(payload),
    });
  },
  getPlanStatus(planId: number) {
    return requestJson<ApiPlanStatus>(`/api/v1/plans/${planId}/status`);
  },
  getPlan(planId: number) {
    return requestJson<ApiHomePlan>(`/api/v1/plans/${planId}`);
  },
};
