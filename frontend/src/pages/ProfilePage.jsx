import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { NavBar } from "../components/NavBar";

export function ProfilePage() {
  const [profile, setProfile] = useState(null);
  const [status, setStatus] = useState("Загрузка...");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setStatus("Ошибка авторизации.");
      try {
        const response = await fetch("/api/auth/me", { headers: { Authorization: `Bearer ${token}` } });
        if (!response.ok) return setStatus("Не удалось загрузить профиль.");
        const data = await response.json();
        if (cancelled) return;
        setProfile(data);
        setStatus("");
      } catch {
        if (!cancelled) setStatus("Не удалось загрузить профиль.");
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  return (
    <main className="page">
      <h1>Профиль пользователя</h1>
      <NavBar />
      {status && <p className="status">{status}</p>}
      {profile && <section className="profile-card"><h2>Основная информация</h2><p>Логин: {profile.login}</p><p>Организация: {profile.organization_id}</p></section>}
      <section className="profile-card">
        <h2>Роли в сервисах</h2>
        <p>Здесь отображаются только сервисы, где ваша роль отличается от наблюдателя (`observer`).</p>
        {profile && Array.isArray(profile.services) && profile.services.length > 0 ? (
          <ul className="roles-list">
            {profile.services.filter((item) => String(item.role || "").toLowerCase() !== "observer").map((item) => (
              <li key={item.service_id} className="roles-item">
                <Link to={`/services/${item.service_id}`}>{item.service_name || item.service_id}</Link>
                <span>{item.role}</span>
              </li>
            ))}
          </ul>
        ) : <p className="status">Нет сервисов с недефолтной ролью.</p>}
      </section>
    </main>
  );
}
