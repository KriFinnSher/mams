import { Link, Navigate, Route, Routes, useParams } from "react-router-dom";
import { useState } from "react";
import "./styles.css";

function Layout({ title }) {
  return (
    <main className="page">
      <h1>{title}</h1>
      <nav className="nav">
        <Link to="/login">Логин</Link>
        <Link to="/services">Сервисы</Link>
        <Link to="/services/new">Новый сервис</Link>
        <Link to="/profile">Профиль</Link>
      </nav>
    </main>
  );
}

function LoginPage() {
  const [status, setStatus] = useState("");

  async function onSubmit(event) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const login = String(form.get("login") || "");
    const password = String(form.get("password") || "");

    if (!login || !password) {
      setStatus("Логин и пароль обязательны.");
      return;
    }

    setStatus("Выполняется вход...");

    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ login, password }),
      });
      if (!response.ok) {
        setStatus("Ошибка входа.");
        return;
      }
      const data = await response.json();
      if (!data.token) {
        setStatus("Токен отсутствует в ответе.");
        return;
      }
      localStorage.setItem("mams_token", data.token);
      setStatus("Токен сохранен в localStorage.");
    } catch {
      setStatus("Не удалось подключиться к backend.");
    }
  }

  return (
    <main className="page">
      <h1>Вход</h1>
      <p className="subtitle">Войдите в MAMS.</p>
      <form className="login-form" onSubmit={onSubmit}>
        <label>
          Логин
          <input type="text" name="login" autoComplete="username" required />
        </label>
        <label>
          Пароль
          <input type="password" name="password" autoComplete="current-password" required />
        </label>
        <button type="submit">Войти</button>
      </form>
      <p className="status">{status}</p>
    </main>
  );
}

function ServicesPage() {
  return <Layout title="Список сервисов" />;
}

function NewServicePage() {
  return <Layout title="Создание сервиса" />;
}

function ProfilePage() {
  return <Layout title="Профиль пользователя" />;
}

function ServicePage() {
  const { id } = useParams();
  return <Layout title={`Карточка сервиса: ${id}`} />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/services" element={<ServicesPage />} />
      <Route path="/services/new" element={<NewServicePage />} />
      <Route path="/services/:id" element={<ServicePage />} />
      <Route path="/profile" element={<ProfilePage />} />
    </Routes>
  );
}
