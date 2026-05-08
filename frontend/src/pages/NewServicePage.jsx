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
      const response = await fetch("/api/services/", {
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
      <NavBar />

      <section className="new-service-layout">
        <div className="new-service-head">
          <h1>Новый сервис</h1>
          <p>Создайте карточку сервиса и подключите базовые параметры платформы.</p>
        </div>

        <form className="new-service-form" onSubmit={onSubmit}>
          <section className="new-form-section">
            <div className="new-form-section-head">
              <h2>Основное</h2>
              <p>Базовая карточка сервиса.</p>
            </div>

            <label className="new-field new-field-full">
              <span>Название</span>
              <input name="name" type="text" required />
            </label>

            <label className="new-field new-field-full">
              <span>Описание</span>
              <textarea name="description" rows="3" />
            </label>

            <div className="new-form-grid">
              <label className="new-field">
                <span>Тип</span>
                <select name="type" defaultValue="business" required>
                  <option value="business">business</option>
                  <option value="composition">composition</option>
                </select>
              </label>

              <label className="new-field">
                <span>Важность</span>
                <select name="importance" defaultValue="medium" required>
                  <option value="low">low</option>
                  <option value="medium">medium</option>
                  <option value="high">high</option>
                  <option value="critical">critical</option>
                </select>
              </label>

              <label className="new-field">
                <span>Покрытие тестами (%)</span>
                <input name="test_coverage" type="number" min="0" max="100" defaultValue="0" required />
              </label>

              <label className="new-field">
                <span>Ссылка на команду</span>
                <input name="responsible_team_ref" type="text" placeholder="@team" />
              </label>
            </div>

            <label className="new-check">
              <input name="pii_sensitive" type="checkbox" />
              <span>
                <strong>Сервис работает с PII</strong>
                <small>Отметьте, если сервис обрабатывает персональные данные.</small>
              </span>
            </label>
          </section>

          <section className="new-form-section">
            <div className="new-form-section-head">
              <h2>Quality gate</h2>
              <p>Порог покрытия, который может блокировать релиз.</p>
            </div>

            <div className="new-form-grid">
              <label className="new-field">
                <span>Минимальное покрытие (%)</span>
                <input name="minimum_test_coverage" type="number" min="0" max="100" defaultValue="0" required />
              </label>

              <label className="new-check compact">
                <input name="minimum_test_coverage_enabled" type="checkbox" />
                <span>
                  <strong>Порог включен</strong>
                  <small>Контроль будет применяться при релизе.</small>
                </span>
              </label>
            </div>
          </section>

          <section className="new-form-section">
            <div className="new-form-section-head">
              <h2>Интеграции</h2>
              <p>Репозиторий, ветка и Grafana dashboard.</p>
            </div>

            <label className="new-field new-field-full">
              <span>URL репозитория</span>
              <input name="repository_url" type="url" required />
            </label>

            <div className="new-form-grid">
              <label className="new-field">
                <span>Ветка по умолчанию</span>
                <input name="default_branch" type="text" defaultValue="main" required />
              </label>

              <label className="new-field">
                <span>UID Grafana dashboard</span>
                <input name="grafana_dashboard_uid" type="text" />
              </label>
            </div>
          </section>

          <div className="new-form-actions">
            <button type="submit">Создать сервис</button>
          </div>
        </form>

        {status && <p className="status">{status}</p>}
      </section>
    </main>
  );
}
