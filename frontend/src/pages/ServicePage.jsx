import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { NavBar } from "../components/NavBar";

export function ServicePage() {
  const { id } = useParams();
  const [tab, setTab] = useState("overview");
  const [svc, setSvc] = useState(null);
  const [status, setStatus] = useState("Загрузка...");

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
          <section className="profile-card"><h2>Release</h2><p className="status">Заполняется на следующих задачах.</p></section>
          <section className="profile-card"><h2>Versions history</h2><p className="status">Заполняется на следующих задачах.</p></section>
          <section className="profile-card"><h2>Modules</h2><p className="status">Заполняется на следующих задачах.</p></section>
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
