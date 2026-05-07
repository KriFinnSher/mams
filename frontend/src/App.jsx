import { Link, Navigate, Outlet, Route, Routes, useParams } from "react-router-dom";
import { useEffect, useState } from "react";
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
        if (response.status === 401) {
          setStatus("Неверный логин или пароль.");
          return;
        }
        setStatus("Ошибка авторизации.");
        return;
      }
      const data = await response.json();
      if (!data.token) {
        setStatus("Ошибка авторизации.");
        return;
      }
      localStorage.setItem("mams_token", data.token);
      setStatus("Токен сохранен в localStorage.");
    } catch {
      setStatus("Ошибка авторизации.");
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
  const [items, setItems] = useState([]);
  const [status, setStatus] = useState("Загрузка...");

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const token = localStorage.getItem("mams_token");
      if (!token) {
        setStatus("Ошибка авторизации.");
        return;
      }

      try {
        const response = await fetch("/api/services", {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });
        if (!response.ok) {
          setStatus("Не удалось загрузить сервисы.");
          return;
        }

        const data = await response.json();
        const list = Array.isArray(data) ? data : Array.isArray(data.services) ? data.services : [];
        if (cancelled) {
          return;
        }

        setItems(list);
        setStatus(list.length === 0 ? "Сервисы не найдены." : "");
      } catch {
        if (!cancelled) {
          setStatus("Не удалось загрузить сервисы.");
        }
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <main className="page">
      <h1>Список сервисов</h1>
      <nav className="nav">
        <Link to="/login">Логин</Link>
        <Link to="/services">Сервисы</Link>
        <Link to="/services/new">Новый сервис</Link>
        <Link to="/profile">Профиль</Link>
      </nav>
      {status && <p className="status">{status}</p>}
      {items.length > 0 && (
        <ul className="services-list">
          {items.map((item) => (
            <ServiceListCard key={item.id} service={item} />
          ))}
        </ul>
      )}
    </main>
  );
}

function ServiceListCard({ service }) {
  return (
    <li className="services-item">
      <div className="services-item-head">
        <Link to={`/services/${service.id}`} className="services-item-title">
          {service.name || service.id}
        </Link>
        {service.criticality && <span className="services-item-badge">{service.criticality}</span>}
      </div>
      {service.description && <p className="services-item-description">{service.description}</p>}
      <div className="services-item-meta">
        <span>ID: {service.id}</span>
        {service.version && <span>Версия: {service.version}</span>}
      </div>
    </li>
  );
}

function NewServicePage() {
  return (
    <main className="page">
      <h1>Добавление сервиса</h1>
      <p className="subtitle">Создайте новый сервис в текущей организации.</p>
      <nav className="nav">
        <Link to="/login">Логин</Link>
        <Link to="/services">Сервисы</Link>
        <Link to="/services/new">Новый сервис</Link>
        <Link to="/profile">Профиль</Link>
      </nav>
      <form className="service-form">
        <label>
          Название
          <input name="name" type="text" required />
        </label>
        <label>
          Описание
          <textarea name="description" rows="3" />
        </label>
        <label>
          Тип
          <select name="type" defaultValue="business" required>
            <option value="business">business</option>
            <option value="composition">composition</option>
          </select>
        </label>
        <label>
          Покрытие тестами (%)
          <input name="test_coverage" type="number" min="0" max="100" defaultValue="0" required />
        </label>
        <label>
          Минимальное покрытие (%)
          <input name="minimum_test_coverage" type="number" min="0" max="100" defaultValue="0" required />
        </label>
        <label className="checkbox-row">
          <input name="minimum_test_coverage_enabled" type="checkbox" />
          Минимальное покрытие включено
        </label>
        <label className="checkbox-row">
          <input name="pii_sensitive" type="checkbox" />
          Сервис работает с PII
        </label>
        <label>
          Ссылка на команду
          <input name="responsible_team_ref" type="text" placeholder="@team" />
        </label>
        <label>
          Важность
          <select name="importance" defaultValue="medium" required>
            <option value="low">low</option>
            <option value="medium">medium</option>
            <option value="high">high</option>
            <option value="critical">critical</option>
          </select>
        </label>
        <label>
          URL репозитория
          <input name="repository_url" type="url" required />
        </label>
        <label>
          Ветка по умолчанию
          <input name="default_branch" type="text" defaultValue="main" required />
        </label>
        <label>
          UID Grafana dashboard
          <input name="grafana_dashboard_uid" type="text" />
        </label>
        <button type="submit">Создать сервис</button>
      </form>
    </main>
  );
}

function ProfilePage() {
  return <Layout title="Профиль пользователя" />;
}

function ServicePage() {
  const { id } = useParams();
  return <Layout title={`Карточка сервиса: ${id}`} />;
}

function RequireAuth() {
  const token = localStorage.getItem("mams_token");
  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route path="/services" element={<ServicesPage />} />
        <Route path="/services/new" element={<NewServicePage />} />
        <Route path="/services/:id" element={<ServicePage />} />
        <Route path="/profile" element={<ProfilePage />} />
      </Route>
    </Routes>
  );
}
