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
    <main className="auth-page">
      <section className="auth-card">
        <div className="auth-brand">
          <span className="brand-mark">M</span>
          <span className="brand-text">MAMS</span>
        </div>

        <h1>Вход</h1>
        <p></p>

        <form className="auth-form" onSubmit={onSubmit}>
          <label>
            <span>Логин</span>
            <input type="text" name="login" autoComplete="username" required />
          </label>

          <label>
            <span>Пароль</span>
            <input type="password" name="password" autoComplete="current-password" required />
          </label>

          <button type="submit">Войти</button>
        </form>

        {status && <p className="auth-status">{status}</p>}
      </section>
    </main>
  );
}
