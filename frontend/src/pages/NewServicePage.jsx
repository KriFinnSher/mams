import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { NavBar } from "../components/NavBar";

export function NewServicePage() {
  const [status, setStatus] = useState("");
  const nav = useNavigate();

  async function onSubmit(event) {
    event.preventDefault();
    const token = localStorage.getItem("mams_token");
    if (!token) return setStatus("Ошибка авторизации.");
    const form = new FormData(event.currentTarget);
    const payload = {
      name: String(form.get("name") || ""),
      description: String(form.get("description") || ""),
      type: String(form.get("type") || ""),
      test_coverage: Number(form.get("test_coverage") || 0),
      minimum_test_coverage_enabled: Boolean(form.get("minimum_test_coverage_enabled")),
      minimum_test_coverage: Number(form.get("minimum_test_coverage") || 0),
      pii_sensitive: Boolean(form.get("pii_sensitive")),
      responsible_team_ref: String(form.get("responsible_team_ref") || ""),
      importance: String(form.get("importance") || ""),
      repository_url: String(form.get("repository_url") || ""),
      default_branch: String(form.get("default_branch") || ""),
      grafana_dashboard_uid: String(form.get("grafana_dashboard_uid") || ""),
    };
    setStatus("Создание сервиса...");
    try {
      const response = await fetch("/api/services", {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify(payload),
      });
      if (!response.ok) return setStatus("Не удалось создать сервис.");
      setStatus("Сервис создан.");
      nav("/services");
    } catch {
      setStatus("Не удалось создать сервис.");
    }
  }

  return (
    <main className="page">
      <h1>Добавление сервиса</h1>
      <p className="subtitle">Создайте новый сервис в текущей организации.</p>
      <NavBar />
      <form className="service-form" onSubmit={onSubmit}>
        <label>Название<input name="name" type="text" required /></label>
        <label>Описание<textarea name="description" rows="3" /></label>
        <label>Тип<select name="type" defaultValue="business" required><option value="business">business</option><option value="composition">composition</option></select></label>
        <label>Покрытие тестами (%)<input name="test_coverage" type="number" min="0" max="100" defaultValue="0" required /></label>
        <label>Минимальное покрытие (%)<input name="minimum_test_coverage" type="number" min="0" max="100" defaultValue="0" required /></label>
        <label className="checkbox-row"><input name="minimum_test_coverage_enabled" type="checkbox" />Минимальное покрытие включено</label>
        <label className="checkbox-row"><input name="pii_sensitive" type="checkbox" />Сервис работает с PII</label>
        <label>Ссылка на команду<input name="responsible_team_ref" type="text" placeholder="@team" /></label>
        <label>Важность<select name="importance" defaultValue="medium" required><option value="low">low</option><option value="medium">medium</option><option value="high">high</option><option value="critical">critical</option></select></label>
        <label>URL репозитория<input name="repository_url" type="url" required /></label>
        <label>Ветка по умолчанию<input name="default_branch" type="text" defaultValue="main" required /></label>
        <label>UID Grafana dashboard<input name="grafana_dashboard_uid" type="text" /></label>
        <button type="submit">Создать сервис</button>
      </form>
      <p className="status">{status}</p>
    </main>
  );
}
