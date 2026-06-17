import type {
  ApiDeviceType,
  ApiEcosystem,
  ApiErrorResponse,
  ApiHomePlan,
  ApiPlanStatus,
  ApiPlanSummary,
  ApiPipelineResult,
  ApiStartPipelineRequest,
  ApiStartPipelineResponse,
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

  let response: Response;
  try {
    response = await fetch(buildUrl(path), {
      ...init,
      headers,
      cache: "no-store",
    });
  } catch {
    throw new Error("Не удалось подключиться к серверу. Проверьте, что проект запущен, и попробуйте ещё раз.");
  }

  if (response.status === 401 && allowRefresh && path !== AUTH_PATHS.refresh) {
    const refreshedToken = await refreshAccessToken();
    if (refreshedToken) {
      return requestJson<T>(path, init, false);
    }
  }

  if (!response.ok) {
    const backendMessage = await readErrorMessage(response);
    throw new Error(toUserErrorMessage(response.status, backendMessage));
  }

  try {
    return (await response.json()) as T;
  } catch {
    throw new Error("Сервер вернул неожиданный ответ. Попробуйте ещё раз.");
  }
}

async function requestForm<T>(path: string, formData: FormData, allowRefresh = true): Promise<T> {
  const headers = new Headers();
  const accessToken = getAccessToken();
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  let response: Response;
  try {
    response = await fetch(buildUrl(path), {
      method: "POST",
      body: formData,
      headers,
      cache: "no-store",
    });
  } catch {
    throw new Error("Не удалось подключиться к серверу. Проверьте, что проект запущен, и попробуйте ещё раз.");
  }

  if (response.status === 401 && allowRefresh) {
    const refreshedToken = await refreshAccessToken();
    if (refreshedToken) {
      return requestForm<T>(path, formData, false);
    }
  }

  if (!response.ok) {
    const backendMessage = await readErrorMessage(response);
    throw new Error(toUserErrorMessage(response.status, backendMessage));
  }

  try {
    return (await response.json()) as T;
  } catch {
    throw new Error("Сервер вернул неожиданный ответ. Попробуйте ещё раз.");
  }
}

async function readErrorMessage(response: Response) {
  const contentType = response.headers.get("Content-Type") ?? "";
  try {
    if (contentType.includes("application/json")) {
      const error = (await response.json()) as ApiErrorResponse & Record<string, unknown>;
      return pickErrorText(error.message) ?? pickErrorText(error.details) ?? pickErrorText(error.error);
    }
    return pickErrorText(await response.text());
  } catch {
    return null;
  }
}

function pickErrorText(value: unknown) {
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : null;
}

function toUserErrorMessage(status: number, backendMessage: string | null) {
  const message = backendMessage?.toLowerCase() ?? "";

  if (message.includes("valid email")) {
    return "Введите корректный email.";
  }
  if (message.includes("password") && message.includes("8")) {
    return "Пароль должен быть не короче 8 символов.";
  }
  if (message.includes("user already exists") || message.includes("duplicate") || message.includes("unique")) {
    return "Пользователь с таким email уже зарегистрирован.";
  }
  if (message.includes("invalid credentials")) {
    return "Неверный email или пароль.";
  }
  if (message.includes("refresh token") || message.includes("invalid token")) {
    clearAuthState();
    return "Сессия истекла. Войдите снова.";
  }
  if (message.includes("requirements") && message.includes("empty")) {
    return "Выберите хотя бы одно требование уровня.";
  }
  if (message.includes("unknown field")) {
    return "Форма отправила поле, которое сервер не принимает. Обновите страницу и попробуйте ещё раз.";
  }
  if (message.includes("no compatible") || message.includes("no devices") || message.includes("not found compatible")) {
    return "По выбранным требованиям не удалось подобрать устройства. Попробуйте другой уровень или измените бюджет.";
  }
  if (message.includes("budget")) {
    return "Проверьте бюджет: он должен быть положительным числом.";
  }

  switch (status) {
    case 400:
    case 422:
      return "Проверьте введённые данные.";
    case 401:
      clearAuthState();
      return "Нужно войти в аккаунт.";
    case 403:
      return "Недостаточно прав для этого действия.";
    case 404:
      return "Не удалось найти нужные данные.";
    case 409:
      return "Такая запись уже существует.";
    case 413:
      return "Файл слишком большой для загрузки. Попробуйте DXF меньшего размера или сожмите план.";
    case 429:
      return "Слишком много попыток. Попробуйте позже.";
    case 500:
      return "На сервере произошла ошибка. Попробуйте позже.";
    case 502:
    case 503:
    case 504:
      return "Сервис временно недоступен. Подождите немного и попробуйте ещё раз.";
    default:
      return "Что-то пошло не так. Попробуйте ещё раз.";
  }
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
  listDeviceTypes() {
    return requestJson<ApiDeviceType[]>("/api/v1/device-types");
  },
  startPipeline(payload: ApiStartPipelineRequest) {
    return requestJson<ApiStartPipelineResponse>("/start", {
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
  getPipelineResult(workflowId: string, runId?: string) {
    const params = new URLSearchParams();
    if (runId) params.set("run_id", runId);
    const suffix = params.toString() ? `?${params.toString()}` : "";
    return requestJson<ApiPipelineResult | { workflow_id: string; run_id?: string; status: string }>(
      `/result/${encodeURIComponent(workflowId)}${suffix}`
    );
  },
  parseFloorPlan(file: File) {
    const formData = new FormData();
    formData.set("file", file);
    return requestForm<unknown>("/api/v1/floor-parser/parse", formData);
  },
};
