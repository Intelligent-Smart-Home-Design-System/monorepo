"use client";

/* eslint-disable react-hooks/set-state-in-effect */

import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { api } from "./api";
import {
  clearAuthState,
  loadAuthState,
  normalizeAuthResponse,
  persistAuthResponse,
} from "./auth";
import type { AuthUser, LoginRequest, RegisterRequest } from "./types";

type AuthContextValue = {
  isAuthenticated: boolean;
  user: AuthUser | null;
  loading: boolean;
  login: (payload: LoginRequest) => Promise<void>;
  register: (payload: RegisterRequest) => Promise<void>;
  logout: () => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider(props: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const state = loadAuthState();
    setUser(state?.user ?? null);
    setLoading(false);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      isAuthenticated: Boolean(user),
      user,
      loading,
      async login(payload) {
        const response = normalizeAuthResponse(await api.login(payload));
        persistAuthResponse(response, payload.email);
        setUser(response.user ?? { email: payload.email });
      },
      async register(payload) {
        const response = normalizeAuthResponse(await api.register(payload));
        persistAuthResponse(response, payload.email);
        setUser(response.user ?? { email: payload.email, name: payload.name ?? null });
      },
      logout() {
        clearAuthState();
        setUser(null);
      },
    }),
    [loading, user]
  );

  return <AuthContext.Provider value={value}>{props.children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return value;
}
