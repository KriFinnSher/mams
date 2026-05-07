import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { NavBar } from "../components/NavBar";

export function ServicePage() {
  const { id } = useParams();
  const [tab, setTab] = useState("overview");
  const [svc, setSvc] = useState(null);
  const [status, setStatus] = useState("Загрузка...");
  const [effectiveRole, setEffectiveRole] = useState("observer");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setStatus("Ошибка авторизации.");
      try {
        const response = await fetch(`/api/services/${id}`, { headers: { Authorization: `Bearer ${token}` } });
        if (!response.ok) return setStatus("Не удалось загрузить сервис.");
        const data = await response.json();
        if (cancelled) return;
        setSvc(data);
        setStatus("");

        const meResp = await fetch("/api/auth/me", { headers: { Authorization: `Bearer ${token}` } });
        if (meResp.ok) {
          const me = await meResp.json();
          const match = Array.isArray(me.services) ? me.services.find((item) => item.service_id === id) : null;
          setEffectiveRole(match?.role || "observer");
        }
      } catch {
        if (!cancelled) setStatus("Не удалось загрузить сервис.");
      }
    }
    load();
    return () => { cancelled = true; };
  }, [id]);

  return (
    <main className="page">
      <h1>Карточка сервиса: {id}</h1>
      <NavBar />
      <div className="tabs">
        <button type="button" className={tab === "overview" ? "tab tab-active" : "tab"} onClick={() => setTab("overview")}>
          Overview
        </button>
        <button type="button" className={tab === "settings" ? "tab tab-active" : "tab"} onClick={() => setTab("settings")}>
          Settings
        </button>
      </div>
      {tab === "overview" && (
        <section className="panel-grid">
          <section className="profile-card">
            <h2>Service information</h2>
            {status && <p className="status">{status}</p>}
            {svc && (
              <>
                <p>Название: {svc.name || "-"}</p>
                <p>Тип: {svc.type || "-"}</p>
                <p>Версия: {svc.version || "-"}</p>
                <p>Описание: {svc.description || "-"}</p>
              </>
            )}
          </section>
          <section className="profile-card">
            <h2>Release</h2>
            <form className="inline-form">
              <label>Стратегия<select defaultValue="rolling"><option value="rolling">rolling</option><option value="recreate">recreate</option><option value="canary">canary</option></select></label>
              <label>Окружение<select defaultValue="dev"><option value="dev">dev</option><option value="staging">staging</option><option value="prod">prod</option></select></label>
              <label>Ветка<input type="text" defaultValue={svc?.default_branch || "main"} /></label>
              <button type="button">Запустить деплой</button>
            </form>
          </section>
          <section className="profile-card">
            <h2>Versions history</h2>
            <table className="history-table">
              <thead>
                <tr>
                  <th>Дата</th>
                  <th>Окружение</th>
                  <th>Стратегия</th>
                  <th>Версия</th>
                  <th>Статус</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td colSpan="5" className="status">История релизов пока пуста.</td>
                </tr>
              </tbody>
            </table>
          </section>
          <section className="profile-card">
            <h2>Modules</h2>
            <p className="status">Роль: {effectiveRole}</p>
            <div className="modules-grid">
              <article className="module-card">
                <h3>Contracts</h3>
                <p className="status">Просмотр контрактов сервиса.</p>
              </article>
              {String(effectiveRole).toLowerCase() !== "observer" && (
                <>
                  <article className="module-card">
                    <h3>Metrics</h3>
                    <p className="status">Метрики из Grafana.</p>
                  </article>
                  <article className="module-card">
                    <h3>Logs</h3>
                    <p className="status">История и поток логов.</p>
                  </article>
                </>
              )}
            </div>
          </section>
        </section>
      )}
      {tab === "settings" && (
        <section className="profile-card">
          <h2>Settings</h2>
          <p className="status">Настройки будут добавлены в следующих задачах.</p>
        </section>
      )}
    </main>
  );
}
