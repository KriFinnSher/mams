import { Link } from "react-router-dom";
import { useEffect, useState } from "react";

export function NavBar() {
  const [login, setLogin] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function loadMe() {
      const token = localStorage.getItem("mams_token");
      if (!token) return;
      try {
        const response = await fetch("/api/auth/me", {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (!response.ok) return;
        const data = await response.json();
        if (!cancelled) {
          setLogin(String(data?.login || ""));
        }
      } catch {
        // ignore
      }
    }
    loadMe();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <nav className="topbar">
      <div className="topbar-left">
        <Link to="/services" className="topbar-link">Сервисы</Link>
        <Link to="/services/new" className="topbar-link topbar-link-accent">Новый сервис</Link>
      </div>
      <div className="nav-spacer" />
      <Link to="/profile" className="profile-link" aria-label="Профиль">
        <span className="profile-icon">👤</span>
        <span>{login || "Профиль"}</span>
      </Link>
    </nav>
  );
}
