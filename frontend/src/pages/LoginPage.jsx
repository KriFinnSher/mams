import { useState } from "react";
import { useNavigate } from "react-router-dom";

export function LoginPage() {
  const [status, setStatus] = useState("");
  const navigate = useNavigate();

  async function onSubmit(event) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const login = String(form.get("login") || "");
    const password = String(form.get("password") || "");

    if (!login || !password) return setStatus("Логин и пароль обязательны.");
    setStatus("Выполняется вход...");

    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ login, password }),
      });
      if (!response.ok) {
        setStatus(response.status === 401 ? "Неверный логин или пароль." : "Ошибка авторизации.");
        return;
      }
      const data = await response.json();
      if (!data.token) return setStatus("Ошибка авторизации.");
      localStorage.setItem("mams_token", data.token);
      navigate("/services", { replace: true });
    } catch {
      setStatus("Ошибка авторизации.");
    }
  }

  return (
    <main className="page">
      <h1>Вход</h1>
      <p className="subtitle">Войдите в MAMS.</p>
      <form className="login-form" onSubmit={onSubmit}>
        <label>Логин<input type="text" name="login" autoComplete="username" required /></label>
        <label>Пароль<input type="password" name="password" autoComplete="current-password" required /></label>
        <button type="submit">Войти</button>
      </form>
      <p className="status">{status}</p>
    </main>
  );
}
