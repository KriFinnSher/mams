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
  const [contracts, setContracts] = useState([]);
  const [contractsStatus, setContractsStatus] = useState("");
  const [expandedMethod, setExpandedMethod] = useState("");

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

  useEffect(() => {
    if (tab !== "contracts") return;
    let cancelled = false;
    async function loadContracts() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setContractsStatus("Ошибка авторизации.");
      setContractsStatus("Загрузка...");
      try {
        const resp = await fetch(`/api/services/${id}/contracts`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (!resp.ok) {
          if (resp.status === 404) {
            setContractsStatus("Файл project.proto не найден.");
            return;
          }
          if (resp.status === 400 || resp.status === 422) {
            setContractsStatus("Файл project.proto содержит невалидный формат.");
            return;
          }
          setContractsStatus("Не удалось загрузить контракты.");
          return;
        }
        const data = await resp.json();
        if (cancelled) return;
        const list = Array.isArray(data?.methods) ? data.methods : Array.isArray(data) ? data : [];
        setContracts(list);
        setContractsStatus(list.length === 0 ? "Контракты не найдены." : "");
      } catch {
        if (!cancelled) setContractsStatus("Не удалось загрузить контракты.");
      }
    }
    loadContracts();
    return () => { cancelled = true; };
  }, [id, tab]);

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
        <button type="button" className={tab === "contracts" ? "tab tab-active" : "tab"} onClick={() => setTab("contracts")}>
          Contracts
        </button>
        {String(effectiveRole).toLowerCase() !== "observer" && (
          <button type="button" className={tab === "metrics" ? "tab tab-active" : "tab"} onClick={() => setTab("metrics")}>
            Metrics
          </button>
        )}
      </div>
      {tab === "overview" && (
        <section className="overview-grid">
          <section className="profile-card">
            <h2>Service information</h2>
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
                <dt>PII sensitive</dt><dd>{svc.pii_sensitive ? "да" : "нет"}</dd>
                <dt>Команда</dt><dd>{svc.responsible_team_ref || "-"}</dd>
                <dt>Важность</dt><dd>{svc.importance || "-"}</dd>
                <dt>Репозиторий</dt><dd>{svc.repository_url || "-"}</dd>
                <dt>Ветка по умолчанию</dt><dd>{svc.default_branch || "-"}</dd>
                <dt>Grafana UID</dt><dd>{svc.grafana_dashboard_uid || "-"}</dd>
              </dl>
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
                <button type="submit">Сохранить изменения</button>
              </form>
              <p className="status">{saveStatus}</p>
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
          <section className="profile-card history-panel">
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
          <section className="profile-card modules-panel">
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
      {tab === "contracts" && (
        <section className="profile-card">
          <h2>Contracts</h2>
          {contractsStatus && <p className="status">{contractsStatus}</p>}
          {contracts.length > 0 && (
            <ul className="roles-list">
              {contracts.map((item, idx) => {
                const methodName = item.name || item.method || `method-${idx}`;
                const key = `${methodName}-${idx}`;
                const params = Array.isArray(item.parameters)
                  ? item.parameters
                  : Array.isArray(item.params)
                    ? item.params
                    : [];

                return (
                  <li key={key} className="contract-item">
                    <button
                      type="button"
                      className="contract-toggle"
                      onClick={() => setExpandedMethod((prev) => (prev === key ? "" : key))}
                    >
                      <span>{methodName}</span>
                      <span>{expandedMethod === key ? "Скрыть" : "Показать"}</span>
                    </button>
                    {expandedMethod === key && (
                      <div className="contract-body">
                        <p>Request: {item.request || item.input || "-"}</p>
                        <p>Response: {item.response || item.output || "-"}</p>
                        {params.length > 0 ? (
                          <ul className="contract-params">
                            {params.map((p, pIdx) => (
                              <li key={`${key}-p-${pIdx}`}>{p.name || p.field || "param"}: {p.type || "-"}</li>
                            ))}
                          </ul>
                        ) : (
                          <p className="status">Параметры отсутствуют.</p>
                        )}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      )}
      {tab === "metrics" && String(effectiveRole).toLowerCase() !== "observer" && (
        <section className="profile-card">
          <h2>Metrics</h2>
          {svc?.grafana_dashboard_uid ? (
            <iframe
              title="grafana-metrics"
              className="metrics-frame"
              src={`/api/services/${id}/metrics`}
            />
          ) : (
            <p className="status">Для сервиса не задан `grafana_dashboard_uid`.</p>
          )}
        </section>
      )}
    </main>
  );
}
