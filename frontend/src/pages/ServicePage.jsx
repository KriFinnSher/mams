import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { NavBar } from "../components/NavBar";

export function ServicePage() {
  const { id } = useParams();
  const [tab, setTab] = useState("overview");
  const [svc, setSvc] = useState(null);
  const [status, setStatus] = useState("Загрузка...");
  const [effectiveRole, setEffectiveRole] = useState("observer");
  const [saveStatus, setSaveStatus] = useState("");
  const [isEditingInfo, setIsEditingInfo] = useState(false);

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
          Обзор
        </button>
        <button type="button" className={tab === "settings" ? "tab tab-active" : "tab"} onClick={() => setTab("settings")}>
          Настройки
        </button>
      </div>
      {tab === "overview" && (
        <section className="overview-grid">
          <div className="overview-col">
            <section className="profile-card">
              <h2>Информация о сервисе</h2>
              {status && <p className="status">{status}</p>}
              {svc && (
                <>
                <dl className="info-grid">
                  <dt>Название</dt><dd>{svc.name || "-"}</dd>
                  <dt>Тип</dt><dd>{svc.type || "-"}</dd>
                  <dt>Версия</dt><dd>{svc.version || "-"}</dd>
                  <dt>Описание</dt><dd>{svc.description || "-"}</dd>
                  <dt>Покрытие тестами</dt><dd>{svc.test_coverage ?? "-"}%</dd>
                  <dt>Минимальное покрытие</dt><dd>{svc.minimum_test_coverage_enabled ? `${svc.minimum_test_coverage}%` : "выключено"}</dd>
                  <dt>Чувствительные данные (PII)</dt><dd>{svc.pii_sensitive ? "да" : "нет"}</dd>
                  <dt>Команда</dt><dd>{svc.responsible_team_ref || "-"}</dd>
                  <dt>Важность</dt><dd>{svc.importance || "-"}</dd>
                  <dt>Репозиторий</dt><dd>{svc.repository_url || "-"}</dd>
                  <dt>Ветка по умолчанию</dt><dd>{svc.default_branch || "-"}</dd>
                  <dt>Grafana UID</dt><dd>{svc.grafana_dashboard_uid || "-"}</dd>
                </dl>
                {!isEditingInfo && (
                  <button type="button" className="edit-btn" onClick={() => {
                    setSaveStatus("");
                    setIsEditingInfo(true);
                  }}>
                    Редактировать
                  </button>
                )}
                {isEditingInfo && (
                <form className="service-form" onSubmit={async (event) => {
                  event.preventDefault();
                  const token = localStorage.getItem("mams_token");
                  if (!token) return setSaveStatus("Ошибка авторизации.");
                  const form = new FormData(event.currentTarget);
                  const payload = {
                    description: String(form.get("description") || ""),
                    type: String(form.get("type") || ""),
                    test_coverage: Number(form.get("test_coverage") || 0),
                    pii_sensitive: Boolean(form.get("pii_sensitive")),
                    responsible_team_ref: String(form.get("responsible_team_ref") || ""),
                    importance: String(form.get("importance") || ""),
                    repository_url: String(form.get("repository_url") || ""),
                    default_branch: String(form.get("default_branch") || ""),
                    grafana_dashboard_uid: String(form.get("grafana_dashboard_uid") || ""),
                  };
                  setSaveStatus("Сохранение...");
                  try {
                    const resp = await fetch(`/api/services/${id}`, {
                      method: "PUT",
                      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
                      body: JSON.stringify(payload),
                    });
                    if (!resp.ok) return setSaveStatus("Не удалось сохранить изменения.");
                    const data = await resp.json();
                    setSvc((prev) => ({ ...prev, ...data }));
                    setSaveStatus("Изменения сохранены.");
                    setIsEditingInfo(false);
                  } catch {
                    setSaveStatus("Не удалось сохранить изменения.");
                  }
                }}>
                  <label>Описание<textarea name="description" rows="3" defaultValue={svc.description || ""} /></label>
                  <label>Тип<select name="type" defaultValue={svc.type || "business"}><option value="business">business</option><option value="composition">composition</option></select></label>
                  <label>Покрытие тестами (%)<input name="test_coverage" type="number" min="0" max="100" defaultValue={svc.test_coverage ?? 0} /></label>
                  <label className="checkbox-row"><input name="pii_sensitive" type="checkbox" defaultChecked={Boolean(svc.pii_sensitive)} />Сервис работает с PII</label>
                  <label>Ссылка на команду<input name="responsible_team_ref" type="text" defaultValue={svc.responsible_team_ref || ""} /></label>
                  <label>Важность<select name="importance" defaultValue={svc.importance || "medium"}><option value="low">low</option><option value="medium">medium</option><option value="high">high</option><option value="critical">critical</option></select></label>
                  <label>URL репозитория<input name="repository_url" type="url" defaultValue={svc.repository_url || ""} /></label>
                  <label>Ветка по умолчанию<input name="default_branch" type="text" defaultValue={svc.default_branch || "main"} /></label>
                  <label>UID Grafana dashboard<input name="grafana_dashboard_uid" type="text" defaultValue={svc.grafana_dashboard_uid || ""} /></label>
                  <div className="inline-actions">
                    <button type="submit">Сохранить изменения</button>
                    <button type="button" className="ghost-btn" onClick={() => {
                      setSaveStatus("");
                      setIsEditingInfo(false);
                    }}>
                      Отмена
                    </button>
                  </div>
                </form>
                )}
                <p className="status">{saveStatus}</p>
                </>
              )}
            </section>
            <section className="profile-card">
              <h2>История версий</h2>
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
          </div>
          <div className="overview-col">
            <section className="profile-card">
              <h2>Релиз</h2>
              <form className="inline-form">
                <label>Стратегия<select defaultValue="rolling"><option value="rolling">rolling</option><option value="recreate">recreate</option><option value="canary">canary</option></select></label>
                <label>Окружение<select defaultValue="dev"><option value="dev">dev</option><option value="staging">staging</option><option value="prod">prod</option></select></label>
                <label>Ветка<input type="text" defaultValue={svc?.default_branch || "main"} /></label>
                <button type="button">Запустить деплой</button>
              </form>
            </section>
            <section className="profile-card">
              <h2>Модули</h2>
              <p className="status">Роль: {effectiveRole}</p>
              <div className="modules-grid">
                <article className="module-card">
                  <h3>Контракты</h3>
                  <p className="status">Просмотр контрактов сервиса.</p>
                </article>
                {String(effectiveRole).toLowerCase() !== "observer" && (
                  <>
                    <article className="module-card">
                      <h3>Метрики</h3>
                      <p className="status">Метрики из Grafana.</p>
                    </article>
                    <article className="module-card">
                      <h3>Логи</h3>
                      <p className="status">История и поток логов.</p>
                    </article>
                  </>
                )}
              </div>
            </section>
          </div>
        </section>
      )}
      {tab === "settings" && (
        <section className="profile-card">
          <h2>Настройки</h2>
          <p className="status">Настройки будут добавлены в следующих задачах.</p>
        </section>
      )}
    </main>
  );
}
