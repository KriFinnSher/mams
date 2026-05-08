import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { NavBar } from "../components/NavBar";

function ServiceListCard({ service }) {
  const name = service?.overview?.name || service?.name || service?.id;
  const description = service?.overview?.description || "Описание отсутствует";
  const importance = service?.overview?.importance;
  const version = service?.overview?.version;

  return (
    <li className="service-card">
      <div className="service-card-main">
        <div className="service-card-icon">
          {(name || "S").slice(0, 1).toUpperCase()}
        </div>

        <div>
          <Link to={`/services/${service.id}`} className="service-card-title">
            {name}
          </Link>
          <p className="service-card-description">{description}</p>
        </div>
      </div>

      <div className="service-card-meta">
        {version && <span>v{version}</span>}
        {importance && <span className={`importance-badge ${importance}`}>{importance}</span>}
      </div>
    </li>
  );
}

export function ServicesPage() {
  const [items, setItems] = useState([]);
  const [status, setStatus] = useState("Загрузка...");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setStatus("Ошибка авторизации.");
      try {
        const response = await fetch("/api/services/", {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
        if (!response.ok) return setStatus("Не удалось загрузить сервисы.");
        const data = await response.json();
        const list = Array.isArray(data) ? data : Array.isArray(data.services) ? data.services : [];
        if (cancelled) return;
        setItems(list);
        setStatus(list.length === 0 ? "Сервисы не найдены." : "");
      } catch {
        if (!cancelled) setStatus("Не удалось загрузить сервисы.");
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  return (
    <main className="page">
      <NavBar />

      <section className="list-head">
        <div>
          <h1>Сервисы</h1>
          <p>Каталог сервисов организации и быстрый переход к управлению.</p>
        </div>

        <Link to="/services/new" className="list-create-btn">
          Новый сервис
        </Link>
      </section>

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
