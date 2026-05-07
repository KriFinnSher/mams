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
  const [moduleTab, setModuleTab] = useState("metrics");
  const [logs, setLogs] = useState([]);
  const [logsStatus, setLogsStatus] = useState("");
  const [logLevel, setLogLevel] = useState("");
  const [logText, setLogText] = useState("");
  const [logFrom, setLogFrom] = useState("");
  const [logTo, setLogTo] = useState("");
  const [logsCursor, setLogsCursor] = useState("");
  const [releaseStrategy, setReleaseStrategy] = useState("rolling");
  const [releaseEnv, setReleaseEnv] = useState("dev");
  const [releaseBranch, setReleaseBranch] = useState("main");
  const [releaseTag, setReleaseTag] = useState("");
  const [releaseDescription, setReleaseDescription] = useState("");
  const [releaseMode, setReleaseMode] = useState("deploy");
  const [rollbackTag, setRollbackTag] = useState("");
  const [releaseStatus, setReleaseStatus] = useState("");

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
    if (moduleTab !== "logs") return;
    let cancelled = false;
    async function loadLogs() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setLogsStatus("Ошибка авторизации.");
      setLogsStatus("Загрузка логов...");
      try {
        const params = new URLSearchParams();
        if (logLevel) params.set("level", logLevel);
        if (logText) params.set("text", logText);
        if (logFrom) params.set("time_from", new Date(logFrom).toISOString());
        if (logTo) params.set("time_to", new Date(logTo).toISOString());
        params.set("limit", "100");
        const resp = await fetch(`/api/services/${id}/logs?${params.toString()}`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (!resp.ok) return setLogsStatus("Не удалось загрузить логи.");
        const data = await resp.json();
        if (cancelled) return;
        const list = Array.isArray(data?.logs) ? data.logs : Array.isArray(data) ? data : [];
        setLogs(list);
        const last = list[list.length - 1];
        setLogsCursor(last?.timestamp || "");
        setLogsStatus(list.length === 0 ? "Логи не найдены." : "");
      } catch {
        if (!cancelled) setLogsStatus("Не удалось загрузить логи.");
      }
    }
    loadLogs();
    return () => { cancelled = true; };
  }, [id, moduleTab, logLevel, logText, logFrom, logTo]);

  useEffect(() => {
    if (moduleTab !== "logs" || String(effectiveRole).toLowerCase() === "observer") return;
    const token = localStorage.getItem("mams_token");
    if (!token) return;
    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    const ws = new WebSocket(`${protocol}://${window.location.host}/api/services/${id}/logs/stream`, [token]);
    ws.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data);
        setLogs((prev) => [entry, ...prev].slice(0, 200));
      } catch {
        // ignore malformed entries
      }
    };
    return () => ws.close();
  }, [id, moduleTab, effectiveRole]);

  return (
    <main className="page">
      <h1>{svc?.name || "Сервис"}</h1>
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
                <label>
                  Стратегия
                  <select value={releaseStrategy} onChange={(e) => setReleaseStrategy(e.target.value)} disabled={releaseMode === "rollback"}>
                    <option value="rolling">rolling</option>
                    <option value="recreate">recreate</option>
                    <option value="canary">canary</option>
                  </select>
                </label>
                <label>
                  Окружение
                  <select value={releaseEnv} onChange={(e) => setReleaseEnv(e.target.value)} disabled={releaseMode === "rollback"}>
                    <option value="dev">dev</option>
                    <option value="staging">staging</option>
                    <option value="prod">prod</option>
                  </select>
                </label>
                {releaseMode === "deploy" && releaseEnv !== "prod" && (
                  <label>
                    Ветка
                    <input type="text" value={releaseBranch} onChange={(e) => setReleaseBranch(e.target.value)} />
                  </label>
                )}
                {releaseMode === "deploy" && releaseEnv === "prod" && (
                  <label>
                    Git tag
                    <input type="text" value={releaseTag} onChange={(e) => setReleaseTag(e.target.value)} placeholder="v1.2.3" />
                  </label>
                )}
                {releaseMode === "rollback" && (
                  <label>
                    Версия для rollback
                    <select value={rollbackTag} onChange={(e) => setRollbackTag(e.target.value)}>
                      <option value="">Выберите git tag</option>
                      <option value="v1.2.4">v1.2.4</option>
                      <option value="v1.2.3">v1.2.3</option>
                      <option value="v1.2.2">v1.2.2</option>
                    </select>
                  </label>
                )}
                <label>
                  Описание
                  <input type="text" value={releaseDescription} onChange={(e) => setReleaseDescription(e.target.value)} disabled={releaseMode === "rollback"} />
                </label>
                <div className="inline-actions">
                  <button type="button" className={releaseMode === "deploy" ? "mode-btn active" : "mode-btn"} onClick={() => setReleaseMode("deploy")}>Deploy</button>
                  <button type="button" className={releaseMode === "rollback" ? "mode-btn active" : "mode-btn"} onClick={() => setReleaseMode("rollback")}>Rollback</button>
                </div>
                <button type="button" onClick={async () => {
                  const token = localStorage.getItem("mams_token");
                  if (!token) return setReleaseStatus("Ошибка авторизации.");
                  try {
                    if (releaseMode === "rollback") {
                      if (!rollbackTag) return setReleaseStatus("Выберите версию для rollback.");
                      const resp = await fetch(`/api/services/${id}/rollback`, {
                        method: "POST",
                        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
                        body: JSON.stringify({ git_tag: rollbackTag }),
                      });
                      if (!resp.ok) return setReleaseStatus("Не удалось выполнить rollback.");
                      return setReleaseStatus("Rollback запущен.");
                    }

                    const payload = {
                      strategy: releaseStrategy,
                      environment: releaseEnv,
                      branch: releaseEnv === "prod" ? "" : releaseBranch,
                      git_tag: releaseEnv === "prod" ? releaseTag : "",
                      description: releaseDescription,
                    };
                    const resp = await fetch(`/api/services/${id}/deploy`, {
                      method: "POST",
                      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
                      body: JSON.stringify(payload),
                    });
                    if (!resp.ok) return setReleaseStatus("Не удалось запустить деплой.");
                    return setReleaseStatus("Деплой запущен.");
                  } catch {
                    setReleaseStatus("Операция недоступна.");
                  }
                }}>
                  {releaseMode === "rollback" ? "Запустить rollback" : "Запустить деплой"}
                </button>
                <p className="status">{releaseStatus}</p>
              </form>
            </section>
            <section className="profile-card">
              <h2>Модули</h2>
              <p className="status">Роль: {effectiveRole}</p>
              <div className="module-tabs">
                {String(effectiveRole).toLowerCase() !== "observer" && (
                  <>
                    <button type="button" className={moduleTab === "metrics" ? "module-tab active" : "module-tab"} onClick={() => setModuleTab("metrics")}>Метрики</button>
                    <button type="button" className={moduleTab === "logs" ? "module-tab active" : "module-tab"} onClick={() => setModuleTab("logs")}>Логи</button>
                  </>
                )}
                <button type="button" className={moduleTab === "contracts" ? "module-tab active" : "module-tab"} onClick={() => setModuleTab("contracts")}>Контракты</button>
              </div>
              <div className="modules-grid">
                {moduleTab === "contracts" && <article className="module-card"><p className="status">Просмотр контрактов сервиса.</p></article>}
                {moduleTab === "metrics" && String(effectiveRole).toLowerCase() !== "observer" && <article className="module-card"><p className="status">Метрики из Grafana.</p></article>}
                {moduleTab === "logs" && String(effectiveRole).toLowerCase() !== "observer" && (
                  <article className="module-card">
                    <div className="logs-filters">
                      <select value={logLevel} onChange={(e) => setLogLevel(e.target.value)}>
                        <option value="">Все уровни</option>
                        <option value="debug">debug</option>
                        <option value="info">info</option>
                        <option value="warn">warn</option>
                        <option value="error">error</option>
                      </select>
                      <input type="datetime-local" value={logFrom} onChange={(e) => setLogFrom(e.target.value)} />
                      <input type="datetime-local" value={logTo} onChange={(e) => setLogTo(e.target.value)} />
                      <input type="text" placeholder="Поиск по тексту" value={logText} onChange={(e) => setLogText(e.target.value)} />
                    </div>
                    {logsStatus && <p className="status">{logsStatus}</p>}
                    {logs.length > 0 && (
                      <div className="logs-list">
                        {logs.map((item, idx) => (
                          <div key={`${item.timestamp || "ts"}-${idx}`} className="log-row">
                            <span className="log-time">{item.timestamp || "-"}</span>
                            <span className="log-level">{item.level || "-"}</span>
                            <span className="log-msg">{item.message || "-"}</span>
                          </div>
                        ))}
                      </div>
                    )}
                    <button
                      type="button"
                      className="more-btn"
                      onClick={async () => {
                        const token = localStorage.getItem("mams_token");
                        if (!token) return;
                        if (!logsCursor) return;
                        try {
                          const params = new URLSearchParams();
                          if (logLevel) params.set("level", logLevel);
                          if (logText) params.set("text", logText);
                          if (logFrom) params.set("time_from", new Date(logFrom).toISOString());
                          params.set("time_to", logsCursor);
                          params.set("limit", "100");
                          const resp = await fetch(`/api/services/${id}/logs?${params.toString()}`, {
                            headers: { Authorization: `Bearer ${token}` },
                          });
                          if (!resp.ok) return;
                          const data = await resp.json();
                          const list = Array.isArray(data?.logs) ? data.logs : Array.isArray(data) ? data : [];
                          if (list.length === 0) return;
                          setLogs((prev) => [...prev, ...list]);
                          const last = list[list.length - 1];
                          setLogsCursor(last?.timestamp || logsCursor);
                        } catch {
                          // ignore load-more error
                        }
                      }}
                    >
                      Загрузить ещё
                    </button>
                  </article>
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
