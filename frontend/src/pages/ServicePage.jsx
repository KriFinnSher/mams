import { useState } from "react";
import { useParams } from "react-router-dom";
import { NavBar } from "../components/NavBar";

export function ServicePage() {
  const { id } = useParams();
  const [tab, setTab] = useState("overview");

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
          <section className="profile-card"><h2>Service information</h2><p className="status">Заполняется на следующих задачах.</p></section>
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
