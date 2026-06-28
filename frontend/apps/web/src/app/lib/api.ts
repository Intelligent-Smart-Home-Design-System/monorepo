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
const USE_LOCAL_DEV_FALLBACK = process.env.NODE_ENV !== "production" && API_BASE_URL === "";

function buildUrl(path: string) {
  return `${API_BASE_URL}${path}`;
}

function shouldUseLocalDevFallback(error: unknown) {
  if (!USE_LOCAL_DEV_FALLBACK || !(error instanceof Error)) return false;
  return (
    error.message.includes("Не удалось найти нужные данные") ||
    error.message.includes("Не удалось подключиться к серверу")
  );
}

function localDevAuthResponse(email: string, name?: string): AuthResponse {
  const normalizedEmail = email.trim().toLowerCase();
  const suffix = Date.now().toString(36);
  return {
    access_token: `local-dev-access-${suffix}`,
    refresh_token: `local-dev-refresh-${suffix}`,
    token_type: "Bearer",
    user: {
      id: `local-dev-${suffix}`,
      email: normalizedEmail,
      name: name?.trim() || null,
    },
  };
}

const localDevEcosystems: ApiEcosystem[] = [
  {
    id: "yandex",
    name: "Яндекс Умный дом",
    description: "Основная экосистема для локальной проверки интерфейса.",
    may_be_main: true,
    image_url: "/brands/yandex-home.svg",
  },
  {
    id: "apple",
    name: "Apple Home",
    description: "Вариант экосистемы для демонстрации выбора.",
    may_be_main: true,
    image_url: "/brands/apple-home.svg",
  },
  {
    id: "google",
    name: "Google Home",
    description: "Вариант экосистемы для демонстрации выбора.",
    may_be_main: true,
    image_url: "/brands/google-home.svg",
  },
];

const localDevDeviceTypes: ApiDeviceType[] = [
  "water_leak_sensor",
  "gas_leak_sensor",
  "smart_doorbell",
  "smart_lock",
  "camera",
  "motion_sensor",
  "door_sensor",
  "window_sensor",
  "smart_siren",
  "smart_bulb",
  "wireless_button_switch",
  "smart_dimmer",
  "decorative_luminaire",
  "presence_sensor",
  "illumination_sensor",
  "curtains",
  "built_in_backlight",
  "temperature_sensor",
  "humidity_sensor",
  "co2_sensor",
  "air_purifier",
  "air_conditioner",
  "smart_humidifier",
  "smart_floor_thermostat",
  "smart_radiator_actuator",
  "floor_temperature_sensor",
  "smart_speaker",
  "smart_tv",
  "subwoofer",
  "robot_vacuum",
].map((id) => ({
  id,
  name: id
    .split("_")
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(" "),
  filters: [],
}));

function localDevParsedFloor(file: File) {
  return {
    source: "local-dev-fallback",
    file_name: file.name,
    file_size: file.size,
    rooms: [
      {
        id: "room_1",
        name: "Комната",
        type: "living",
        polygon: [
          { x: 0, y: 0 },
          { x: 6, y: 0 },
          { x: 6, y: 4 },
          { x: 0, y: 4 },
        ],
      },
    ],
    walls: [
      { from: { x: 0, y: 0 }, to: { x: 6, y: 0 } },
      { from: { x: 6, y: 0 }, to: { x: 6, y: 4 } },
      { from: { x: 6, y: 4 }, to: { x: 0, y: 4 } },
      { from: { x: 0, y: 4 }, to: { x: 0, y: 0 } },
    ],
    openings: [],
    metadata: {
      warning: "DXF parser backend is unavailable; using local dev fallback.",
    },
  };
}

function localDevPipelineResult(): ApiPipelineResult {
  const raw = typeof window !== "undefined" ? localStorage.getItem("planner-local-dev-pipeline") : null;
  const request = raw ? (JSON.parse(raw) as ApiStartPipelineRequest) : null;
  const requirements = request?.device_selection.requirements ?? [];
  const listings = requirements.map((requirement, index) => {
    const deviceType = requirement.device_type;
    const readableName = deviceType
      .split("_")
      .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
      .join(" ");
    return {
      id: 10_000 + index,
      name: `${readableName} Demo`,
      brand: request?.device_selection.main_ecosystem ?? "local-dev",
      model: `DEV-${index + 1}`,
      category: deviceType,
      device_type: deviceType,
      price: 2500 + index * 900,
      quantity: requirement.count,
      quality: 0.86,
      requirement_id: requirement.requirement_id,
      url: "#",
      connection: {
        ecosystem: request?.device_selection.main_ecosystem ?? "local-dev",
        protocol: "Wi-Fi",
        method: "Локальный демо-подбор без backend.",
      },
      device_attributes: {
        device_type: deviceType,
      },
    };
  });

  return {
    request_id: request?.request_id ?? "local-dev-request",
    parsed_floor_plan: request?.floor_plan ?? null,
    layout: {
      source: "local-dev-fallback",
      message: "Layout backend is unavailable; using local dev result.",
    },
    device_selection: {
      bundles: [
        {
          id: 1,
          total_cost: listings.reduce((sum, listing) => sum + listing.price * listing.quantity, 0),
          quality_score: 0.86,
          extra_ecosystems_used: 0,
          hubs_used: 0,
          is_recommended: true,
          ecosystems_used: [request?.device_selection.main_ecosystem ?? "local-dev"],
          listings,
        },
      ],
    },
    stages: [
      {
        key: "local-dev",
        title: "Локальный демо-результат",
        status: "completed",
        payload: {
          message: "Backend pipeline is unavailable, so the frontend generated a local demo result.",
        },
      },
    ],
  };
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
    return requestJson<AuthResponse>(
      AUTH_PATHS.login,
      {
        method: "POST",
        body: JSON.stringify(payload),
      },
      false
    ).catch((error: unknown) => {
      if (shouldUseLocalDevFallback(error)) return localDevAuthResponse(payload.email);
      throw error;
    });
  },
  register(payload: RegisterRequest) {
    return requestJson<AuthResponse>(
      AUTH_PATHS.register,
      {
        method: "POST",
        body: JSON.stringify(payload),
      },
      false
    ).catch((error: unknown) => {
      if (shouldUseLocalDevFallback(error)) return localDevAuthResponse(payload.email, payload.name);
      throw error;
    });
  },
  listPlans() {
    return requestJson<ApiPlanSummary[]>("/api/v1/plans").catch((error: unknown) => {
      if (shouldUseLocalDevFallback(error)) return [];
      throw error;
    });
  },
  listEcosystems() {
    return requestJson<ApiEcosystem[]>("/api/v1/ecosystems").catch((error: unknown) => {
      if (shouldUseLocalDevFallback(error)) return localDevEcosystems;
      throw error;
    });
  },
  listDeviceTypes() {
    return requestJson<ApiDeviceType[]>("/api/v1/device-types").catch((error: unknown) => {
      if (shouldUseLocalDevFallback(error)) return localDevDeviceTypes;
      throw error;
    });
  },
  startPipeline(payload: ApiStartPipelineRequest) {
    return requestJson<ApiStartPipelineResponse>("/start", {
      method: "POST",
      body: JSON.stringify(payload),
    }).catch((error: unknown) => {
      if (shouldUseLocalDevFallback(error)) {
        if (typeof window !== "undefined") {
          localStorage.setItem("planner-local-dev-pipeline", JSON.stringify(payload));
        }
        return {
          workflow_id: `local-dev-${Date.now().toString(36)}`,
          run_id: "local",
        };
      }
      throw error;
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
    ).catch((error: unknown) => {
      if (workflowId.startsWith("local-dev-") || shouldUseLocalDevFallback(error)) return localDevPipelineResult();
      throw error;
    });
  },
  parseFloorPlan(file: File) {
    const formData = new FormData();
    formData.set("file", file);
    return requestForm<unknown>("/api/v1/floor-parser/parse", formData).catch((error: unknown) => {
      if (shouldUseLocalDevFallback(error)) return localDevParsedFloor(file);
      throw error;
    });
  },
};
