"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import Link from "next/link";
import { useAuth } from "../lib/auth-context";

export default function LoginPage() {
  const auth = useAuth();
  const router = useRouter();
  const next = getNextPath();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const submit = async () => {
    const trimmedEmail = email.trim();

    if (!trimmedEmail.includes("@")) {
      setError("Введите корректный email.");
      return;
    }

    setSubmitting(true);
    setError("");
    try {
      await auth.login({ email: trimmedEmail, password });
      router.push(next);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Не удалось выполнить вход.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthPageShell
      title="Вход"
      subtitle="Введите email и пароль, чтобы продолжить создание плана."
    >
      <Stack spacing={2}>
        {error && <Alert severity="error">{error}</Alert>}
        <TextField label="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} fullWidth />
        <TextField
          label="Пароль"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          fullWidth
        />
        <Button variant="contained" size="large" disabled={!email || !password || submitting} onClick={submit}>
          {submitting ? "Входим..." : "Войти"}
        </Button>
        <Typography variant="body2" color="text.secondary">
          Нет аккаунта? <Link href="/register">Зарегистрироваться</Link>
        </Typography>
      </Stack>
    </AuthPageShell>
  );
}

function AuthPageShell(props: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <Box sx={{ minHeight: "100vh", display: "grid", placeItems: "center", px: 2, py: 4 }}>
      <Card sx={{ width: "100%", maxWidth: 520, borderRadius: 6, boxShadow: "0 25px 70px rgba(15,23,42,0.16)" }}>
        <CardContent sx={{ p: 4 }}>
          <Stack spacing={2.5}>
            <Box>
              <Typography variant="h4" sx={{ fontWeight: 900, mb: 1 }}>
                {props.title}
              </Typography>
              <Typography color="text.secondary">{props.subtitle}</Typography>
            </Box>
            {props.children}
          </Stack>
        </CardContent>
      </Card>
    </Box>
  );
}

function getNextPath() {
  if (typeof window === "undefined") return "/";
  const next = new URLSearchParams(window.location.search).get("next");
  return next || "/";
}
