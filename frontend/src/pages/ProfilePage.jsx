import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { NavBar } from "../components/NavBar";

export function ProfilePage() {
  const navigate = useNavigate();
  const [profile, setProfile] = useState(null);
  const [status, setStatus] = useState("Загрузка...");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setStatus("Ошибка авторизации.");
      try {
        const response = await fetch("/api/auth/me", {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
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
      <NavBar />

      <section className="profile-page-head">
        <div>
          <h1>Профиль</h1>
          <p>Информация об учетной записи и доступах к сервисам.</p>
        </div>
      </section>

      {status && <p className="status">{status}</p>}

      {profile && (
        <section className="profile-hero">
          <div className="profile-avatar">
            {(profile.login || "U").slice(0, 1).toUpperCase()}
          </div>

          <div className="profile-hero-main">
            <h2>{profile.login}</h2>
            <p>Организация: {profile.organization_id}</p>
          </div>

          <button
            type="button"
            className="profile-logout-btn"
            onClick={() => {
              localStorage.removeItem("mams_token");
              navigate("/login", { replace: true });
            }}
          >
            Выйти
          </button>
        </section>
      )}

      <section className="profile-roles-card">
        <div className="profile-section-head">
          <div>
            <h2>Роли в сервисах</h2>
            <p>Сервисы, где ваша роль отличается от observer.</p>
          </div>
        </div>

        {profile && Array.isArray(profile.services) && profile.services.filter((item) => String(item.role || "").toLowerCase() !== "observer").length > 0 ? (
          <div className="profile-roles-list">
            {profile.services
              .filter((item) => String(item.role || "").toLowerCase() !== "observer")
              .map((item) => (
                <Link key={item.service_id} to={`/services/${item.service_id}`} className="profile-role-item">
                  <div>
                    <strong>{item.service_name || item.service_id}</strong>
                    <span>{item.service_id}</span>
                  </div>

                  <span className={`profile-role-badge ${String(item.role || "").toLowerCase()}`}>
                    {item.role}
                  </span>
                </Link>
              ))}
          </div>
        ) : (
          <p className="status">Нет сервисов с недефолтной ролью.</p>
        )}
      </section>
    </main>
  );
}
