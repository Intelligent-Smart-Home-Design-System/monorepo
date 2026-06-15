import type { AuthResponse, AuthTokens, AuthUser } from "./types";

const AUTH_STORAGE_KEY = "smart-home-auth";

const AUTH_LOGIN_PATH = process.env.NEXT_PUBLIC_AUTH_LOGIN_PATH ?? "/api/v1/auth/login";
const AUTH_REGISTER_PATH = process.env.NEXT_PUBLIC_AUTH_REGISTER_PATH ?? "/api/v1/auth/register";
const AUTH_REFRESH_PATH = process.env.NEXT_PUBLIC_AUTH_REFRESH_PATH ?? "/api/v1/auth/refresh";

type StoredAuthState = {
  tokens: AuthTokens;
  user?: AuthUser | null;
};

export function getAuthPaths() {
  return {
    login: AUTH_LOGIN_PATH,
    register: AUTH_REGISTER_PATH,
    refresh: AUTH_REFRESH_PATH,
  };
}

export function loadAuthState(): StoredAuthState | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(AUTH_STORAGE_KEY);
    return raw ? (JSON.parse(raw) as StoredAuthState) : null;
  } catch {
    return null;
  }
}

export function saveAuthState(state: StoredAuthState | null) {
  if (typeof window === "undefined") return;
  if (!state) {
    localStorage.removeItem(AUTH_STORAGE_KEY);
    return;
  }
  localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(state));
}

export function getAccessToken() {
  return loadAuthState()?.tokens.access_token ?? "";
}

export function getRefreshToken() {
  return loadAuthState()?.tokens.refresh_token ?? "";
}

export function clearAuthState() {
  saveAuthState(null);
}

export function normalizeAuthResponse(response: unknown): AuthResponse {
  if (!response || typeof response !== "object") {
    throw new Error("Сервер вернул неожиданный ответ. Попробуйте ещё раз.");
  }

  const value = response as Record<string, unknown>;
  const accessToken =
    pickString(value.access_token) ??
    pickString(value.accessToken) ??
    pickString((value.tokens as Record<string, unknown> | undefined)?.access_token) ??
    pickString((value.tokens as Record<string, unknown> | undefined)?.accessToken);
  const refreshToken =
    pickString(value.refresh_token) ??
    pickString(value.refreshToken) ??
    pickString((value.tokens as Record<string, unknown> | undefined)?.refresh_token) ??
    pickString((value.tokens as Record<string, unknown> | undefined)?.refreshToken);

  if (!accessToken || !refreshToken) {
    throw new Error("Не удалось завершить вход. Попробуйте ещё раз.");
  }

  const user = normalizeUser(value.user);
  return {
    access_token: accessToken,
    refresh_token: refreshToken,
    token_type: pickString(value.token_type) ?? "Bearer",
    user,
    message: pickString(value.message) ?? undefined,
  };
}

export function persistAuthResponse(response: AuthResponse, fallbackEmail?: string) {
  saveAuthState({
    tokens: {
      access_token: response.access_token,
      refresh_token: response.refresh_token,
      token_type: response.token_type ?? "Bearer",
    },
    user:
      response.user ??
      (fallbackEmail
        ? {
            email: fallbackEmail,
          }
        : null),
  });
}

function normalizeUser(value: unknown): AuthUser | null {
  if (!value || typeof value !== "object") return null;
  const user = value as Record<string, unknown>;
  const email = pickString(user.email);
  if (!email) return null;
  return {
    id: pickString(user.id) ?? (typeof user.id === "number" ? user.id : undefined),
    email,
    name: pickString(user.name) ?? pickString(user.full_name) ?? null,
  };
}

function pickString(value: unknown) {
  return typeof value === "string" && value.length > 0 ? value : null;
}
