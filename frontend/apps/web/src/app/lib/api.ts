<<<<<<< HEAD
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
  AuthResponse,
  LoginRequest,
  RefreshTokenRequest,
  RegisterRequest,
} from "./types";
import {
  clearAuthState,
  getAccessToken,
  getAuthPaths,
  getRefreshToken,
  normalizeAuthResponse,
  persistAuthResponse,
} from "./auth";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "") ?? "";
const AUTH_PATHS = getAuthPaths();
let refreshPromise: Promise<string | null> | null = null;

function buildUrl(path: string) {
  return `${API_BASE_URL}${path}`;
}

async function requestJson<T>(path: string, init?: RequestInit, allowRefresh = true): Promise<T> {
  const headers = new Headers(init?.headers ?? {});
  if (!headers.has("Content-Type") && init?.body) {
    headers.set("Content-Type", "application/json");
  }

  const accessToken = getAccessToken();
  if (accessToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await fetch(buildUrl(path), {
    ...init,
    headers,
    cache: "no-store",
  });

  if (response.status === 401 && allowRefresh && path !== AUTH_PATHS.refresh) {
    const refreshedToken = await refreshAccessToken();
    if (refreshedToken) {
      return requestJson<T>(path, init, false);
    }
  }

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

async function refreshAccessToken() {
  const refreshToken = getRefreshToken();
  if (!refreshToken) {
    clearAuthState();
    return null;
  }

  if (!refreshPromise) {
    refreshPromise = requestJson<AuthResponse>(
      AUTH_PATHS.refresh,
      {
        method: "POST",
        body: JSON.stringify({
          refresh_token: refreshToken,
        } satisfies RefreshTokenRequest),
      },
      false
    )
      .then((response) => {
        const normalized = normalizeAuthResponse(response);
        persistAuthResponse(normalized);
        return normalized.access_token;
      })
      .catch(() => {
        clearAuthState();
        return null;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }

  return refreshPromise;
}

export const api = {
  login(payload: LoginRequest) {
    return requestJson<AuthResponse>(AUTH_PATHS.login, {
      method: "POST",
      body: JSON.stringify(payload),
    }, false);
  },
  register(payload: RegisterRequest) {
    return requestJson<AuthResponse>(AUTH_PATHS.register, {
      method: "POST",
      body: JSON.stringify(payload),
    }, false);
  },
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
=======
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
>>>>>>> 4bf54f8 (hz)
