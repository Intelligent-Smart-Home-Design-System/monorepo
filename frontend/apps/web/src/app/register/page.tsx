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

export default function RegisterPage() {
  const auth = useAuth();
  const router = useRouter();
  const next = getNextPath();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const submit = async () => {
    const trimmedEmail = email.trim();
    const trimmedName = name.trim();

    if (!trimmedEmail.includes("@")) {
      setError("Введите корректный email.");
      return;
    }

    if (password.length < 8) {
      setError("Пароль должен быть не короче 8 символов.");
      return;
    }

    if (password !== confirmPassword) {
      setError("Пароли не совпадают.");
      return;
    }

    setSubmitting(true);
    setError("");
    try {
      await auth.register({ email: trimmedEmail, password, name: trimmedName });
      router.push(next);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Не удалось зарегистрироваться.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AuthPageShell
      title="Регистрация"
      subtitle="Создайте аккаунт, чтобы продолжить работу с планировщиком."
    >
      <Stack spacing={2}>
        {error && <Alert severity="error">{error}</Alert>}
        <TextField label="Имя" value={name} onChange={(e) => setName(e.target.value)} fullWidth />
        <TextField label="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} fullWidth />
        <TextField
          label="Пароль"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          fullWidth
        />
        <TextField
          label="Повторите пароль"
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          fullWidth
        />
        <Button
          variant="contained"
          size="large"
          disabled={!email || !password || !confirmPassword || submitting}
          onClick={submit}
        >
          {submitting ? "Регистрируем..." : "Зарегистрироваться"}
        </Button>
        <Typography variant="body2" color="text.secondary">
          Уже есть аккаунт? <Link href="/login">Войти</Link>
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
