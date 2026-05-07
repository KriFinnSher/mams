import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { NavBar } from "../components/NavBar";

function ServiceListCard({ service }) {
  return (
    <li className="services-item">
      <div className="services-item-head">
        <Link to={`/services/${service.id}`} className="services-item-title">{service.name || service.id}</Link>
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

export function ServicesPage() {
  const [items, setItems] = useState([]);
  const [status, setStatus] = useState("Загрузка...");

  useEffect(() => {
    let cancelled = false;
    async function load() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setStatus("Ошибка авторизации.");
      try {
        const response = await fetch("/api/services", {
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
      <h1>Список сервисов</h1>
      <NavBar />
      {status && <p className="status">{status}</p>}
      {items.length > 0 && <ul className="services-list">{items.map((item) => <ServiceListCard key={item.id} service={item} />)}</ul>}
    </main>
  );
}
