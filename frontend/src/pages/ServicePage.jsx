import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { Link, useNavigate } from "react-router-dom";

export function NavBar() {
  const navigate = useNavigate();

  function logout() {
    localStorage.removeItem("mams_token");
    navigate("/login", { replace: true });
  }

  return (
    <header className="topbar">
      <div className="topbar-brand">
        <span className="brand-mark">M</span>
        <span className="brand-text">MAMS</span>
      </div>

      <nav className="topbar-nav">
        <Link to="/services" className="topbar-link">Сервисы</Link>
        <Link to="/services/new" className="topbar-link">Новый сервис</Link>
      </nav>

      <div className="topbar-spacer" />

      <Link to="/profile" className="profile-link">
        <span className="profile-icon">👤</span>
        <span>Профиль</span>
      </Link>

      <button type="button" className="logout-btn" onClick={logout}>
        Выйти
      </button>
    </header>
  );
}

function normalizeContractMethod(method) {
  const params = Array.isArray(method?.parameters)
    ? method.parameters
    : Array.isArray(method?.Parameters)
      ? method.Parameters
      : [];

  const outputParams = Array.isArray(method?.output_parameters)
    ? method.output_parameters
    : Array.isArray(method?.OutputParameters)
      ? method.OutputParameters
      : [];

  return {
    name: method?.name ?? method?.Name ?? "",
    input: method?.input ?? method?.Input ?? "",
    output: method?.output ?? method?.Output ?? "",
    parameters: params.map((p) => ({
      name: p?.name ?? p?.Name ?? "",
      type: p?.type ?? p?.Type ?? "",
      children: Array.isArray(p?.children) ? p.children : [],
    })),
    output_parameters: outputParams.map((p) => ({
      name: p?.name ?? p?.Name ?? "",
      type: p?.type ?? p?.Type ?? "",
      children: Array.isArray(p?.children) ? p.children : [],
    })),
  };
}

function renderContractTree(rows, depth = 0) {
  if (!Array.isArray(rows) || rows.length === 0) return null;
  return rows.map((row, idx) => {
    const key = `${row?.name || "field"}-${idx}-${depth}`;
    const hasChildren = Array.isArray(row?.children) && row.children.length > 0;
    return (
      <div key={key}>
        <div className="contract-tree-row" style={{ paddingLeft: `${12 + depth * 20}px` }}>
          <span className="contract-tree-name">{row?.name || "-"}</span>
          <span className="contract-tree-type">{row?.type || "-"}</span>
        </div>
        {hasChildren && renderContractTree(row.children, depth + 1)}
      </div>
    );
  });
}

function logLevelClass(level) {
  switch (String(level || "").toLowerCase()) {
    case "debug":
      return "log-badge debug";

    case "info":
      return "log-badge info";

    case "warn":
      return "log-badge warn";

    case "error":
      return "log-badge error";

    default:
      return "log-badge";
  }
}

function formatDateTime(value) {
  if (!value) return "-";

  const date = new Date(value);

  const datePart = new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(date);

  const timePart = new Intl.DateTimeFormat("ru-RU", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);

  return `${datePart}, ${timePart}`;
}

function getLogKey(item) {
  const ts = item?.timestamp || "";
  const level = item?.level || "";
  const message = item?.message || "";
  return `${ts}|${level}|${message}`;
}

function sortLogsDesc(list) {
  return [...list].sort((a, b) => {
    const aTime = new Date(a?.timestamp || 0).getTime();
    const bTime = new Date(b?.timestamp || 0).getTime();
    return bTime - aTime;
  });
}

function mergeLogs(existing, incoming) {
  const map = new Map();

  for (const item of [...existing, ...incoming]) {
    const key = getLogKey(item);
    if (!map.has(key)) {
      map.set(key, item);
    }
  }

  return sortLogsDesc(Array.from(map.values())).slice(0, 500);
}

function releaseStatusClass(status) {
  switch (String(status || "").toLowerCase()) {
    case "success":
      return "release-status success";

    case "failed":
      return "release-status failed";

    case "pending":
      return "release-status pending";

    case "in_progress":
      return "release-status progress";

    default:
      return "release-status";
  }
}

function matchesLogFilters(entry, { logLevel, logText, logFrom, logTo }) {
  if (logLevel && String(entry?.level || "").toLowerCase() !== logLevel) {
    return false;
  }

  if (logText) {
    const message = String(entry?.message || "").toLowerCase();
    const needle = String(logText).toLowerCase();
    if (!message.includes(needle)) {
      return false;
    }
  }

  const timestamp = entry?.timestamp ? new Date(entry.timestamp).getTime() : null;

  if (logFrom && timestamp !== null) {
    const from = new Date(logFrom).getTime();
    if (timestamp < from) {
      return false;
    }
  }

  if (logTo && timestamp !== null) {
    const to = new Date(logTo).getTime();
    if (timestamp > to) {
      return false;
    }
  }

  return true;
}

export function ServicePage() {
  const { id } = useParams();
  const [tab, setTab] = useState("overview");
  const [svc, setSvc] = useState(null);
  const [status, setStatus] = useState("Загрузка...");
  const [effectiveRole, setEffectiveRole] = useState("observer");
  const [saveStatus, setSaveStatus] = useState("");
  const [isEditingInfo, setIsEditingInfo] = useState(false);
  const [moduleTab, setModuleTab] = useState("metrics");
  const [roleLoaded, setRoleLoaded] = useState(false);
  const [logs, setLogs] = useState([]);
  const [logsStatus, setLogsStatus] = useState("");
  const [logLevel, setLogLevel] = useState("");
  const [logText, setLogText] = useState("");
  const [logFrom, setLogFrom] = useState("");
  const [logTo, setLogTo] = useState("");
  const [contracts, setContracts] = useState([]);
  const [contractsStatus, setContractsStatus] = useState("");
  const [openContractPanels, setOpenContractPanels] = useState({});
  const [metricsEmbedURL, setMetricsEmbedURL] = useState("");
  const [metricsStatus, setMetricsStatus] = useState("");
  const [releaseStrategy, setReleaseStrategy] = useState("rolling");
  const [releaseEnv, setReleaseEnv] = useState("dev");
  const [releaseBranch, setReleaseBranch] = useState("main");
  const [releaseTag, setReleaseTag] = useState("");
  const [releaseDescription, setReleaseDescription] = useState("");
  const [releaseMode, setReleaseMode] = useState("deploy");
  const [rollbackTag, setRollbackTag] = useState("");
  const [rollbackCandidates, setRollbackCandidates] = useState([]);
  const [releaseStatus, setReleaseStatus] = useState("");
  const [releases, setReleases] = useState([]);
  const [releasesStatus, setReleasesStatus] = useState("");
  const [activeReleaseID, setActiveReleaseID] = useState("");
  const coverageEnabled = Boolean(svc?.settings?.minimum_test_coverage_enabled ?? svc?.minimum_test_coverage_enabled);
  const coverageMin = Number(svc?.settings?.minimum_test_coverage ?? svc?.minimum_test_coverage ?? 0);
  const coverageCurrent = Number(svc?.overview?.test_coverage ?? svc?.test_coverage ?? 0);
  const releaseBlocked = Boolean(
    releaseMode === "deploy" &&
    coverageEnabled &&
    coverageCurrent < coverageMin,
  );
  const releaseBlockedHint = `Релиз заблокирован: текущее покрытие (${coverageCurrent}%) ниже минимального порога (${coverageMin}%).`;
  const isObserver = String(effectiveRole).toLowerCase() === "observer";
  const isOwner = String(effectiveRole).toLowerCase() === "service_owner";
  const isDeveloper = String(effectiveRole).toLowerCase() === "developer";
  const [settingsEnabled, setSettingsEnabled] = useState(false);
  const [settingsMinCoverage, setSettingsMinCoverage] = useState(0);
  const [settingsStatus, setSettingsStatus] = useState("");
  const [isEditingSettings, setIsEditingSettings] = useState(false);
  const [reloadTick, setReloadTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setStatus("Ошибка авторизации.");
      try {
        const response = await fetch(`/api/services/${id}`, {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
        if (!response.ok) return setStatus("Не удалось загрузить сервис.");
        const data = await response.json();
        if (cancelled) return;
        setSvc(data);
        setSettingsEnabled(Boolean(data?.settings?.minimum_test_coverage_enabled ?? data.minimum_test_coverage_enabled));
        setSettingsMinCoverage(Number(data?.settings?.minimum_test_coverage ?? data.minimum_test_coverage ?? 0));
        setStatus("");

        const meResp = await fetch("/api/auth/me", {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
        if (meResp.ok) {
          const me = await meResp.json();
          const match = Array.isArray(me.services) ? me.services.find((item) => item.service_id === id) : null;
          setEffectiveRole(match?.role || "observer");
          setRoleLoaded(true);
        }
      } catch {
        if (!cancelled) setStatus("Не удалось загрузить сервис.");
      }
    }
    load();
    return () => { cancelled = true; };
  }, [id, reloadTick]);

  useEffect(() => {
    if (releaseMode !== "rollback") return;
    let cancelled = false;
    async function loadRollbackCandidates() {
      const token = localStorage.getItem("mams_token");
      if (!token) return;
      try {
        const resp = await fetch(`/api/services/${id}/rollback/candidates`, {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
        if (!resp.ok) return;
        const data = await resp.json();
        if (cancelled) return;
        setRollbackCandidates(Array.isArray(data?.git_tags) ? data.git_tags : []);
      } catch {
        if (!cancelled) setRollbackCandidates([]);
      }
    }
    loadRollbackCandidates();
    return () => { cancelled = true; };
  }, [id, releaseMode]);

  useEffect(() => {
    function onVisible() {
      if (document.visibilityState === "visible") {
        setReloadTick((v) => v + 1);
      }
    }
    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("focus", onVisible);
    return () => {
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("focus", onVisible);
    };
  }, []);

  useEffect(() => {
    if (moduleTab !== "logs") return;
    if (!roleLoaded) return;
    if (String(effectiveRole).toLowerCase() === "observer") return;

    let cancelled = false;

    async function loadLogs() {
      const token = localStorage.getItem("mams_token");
      if (!token) {
        setLogsStatus("Ошибка авторизации.");
        return;
      }

      setLogsStatus("Загрузка логов...");

      try {
        const params = new URLSearchParams();
        params.set("limit", "100");

        if (logLevel) params.set("level", logLevel);
        if (logText) params.set("text", logText);
        if (logFrom) params.set("time_from", new Date(logFrom).toISOString());
        if (logTo) params.set("time_to", new Date(logTo).toISOString());

        const resp = await fetch(`/api/services/${id}/logs?${params.toString()}`, {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });

        if (!resp.ok) {
          if (!cancelled) setLogsStatus("Не удалось загрузить логи.");
          return;
        }

        const data = await resp.json();
        if (cancelled) return;

        const list = Array.isArray(data?.logs) ? data.logs : Array.isArray(data) ? data : [];
        setLogs(sortLogsDesc(list).slice(0, 100));
        setLogsStatus(list.length === 0 ? "Логи не найдены." : "");
      } catch {
        if (!cancelled) setLogsStatus("Не удалось загрузить логи.");
      }
    }

    loadLogs();

    return () => {
      cancelled = true;
    };
  }, [id, moduleTab, roleLoaded, effectiveRole, logLevel, logText, logFrom, logTo]);

  useEffect(() => {
    if (moduleTab !== "logs") return;
    if (!roleLoaded) return;
    if (String(effectiveRole).toLowerCase() === "observer") return;

    const token = localStorage.getItem("mams_token");
    if (!token) return;

    const protocol = window.location.protocol === "https:" ? "wss" : "ws";
    const wsUrl = `${protocol}://${window.location.host}/ws/services/${id}/logs?token=${token}`;
    const ws = new WebSocket(wsUrl);

    ws.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data);

        if (!matchesLogFilters(entry, { logLevel, logText, logFrom, logTo })) {
          return;
        }

        setLogs((prev) => mergeLogs(prev, [entry]));
      } catch (e) {
        console.error("Failed to parse WS message:", e, event.data);
      }
    };

    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
    };

    return () => ws.close();
  }, [id, moduleTab, effectiveRole, roleLoaded, logLevel, logText, logFrom, logTo]);

  useEffect(() => {
    if (moduleTab !== "contracts") return;
    let cancelled = false;
    async function loadContracts() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setContractsStatus("Ошибка авторизации.");
      setContractsStatus("Загрузка контрактов...");
      try {
        const resp = await fetch(`/api/services/${id}/contracts`, {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
        if (resp.status === 404) return setContractsStatus("Файл project.proto отсутствует.");
        if (resp.status === 400) return setContractsStatus("Файл project.proto невалиден.");
        if (!resp.ok) return setContractsStatus("Не удалось загрузить контракты.");
        const data = await resp.json();
        if (cancelled) return;
        const raw = Array.isArray(data?.methods) ? data.methods : [];
        const list = raw.map(normalizeContractMethod);
        setContracts(list);
        setOpenContractPanels({});
        setContractsStatus(list.length === 0 ? "Контракты не найдены." : "");
      } catch {
        if (!cancelled) setContractsStatus("Не удалось загрузить контракты.");
      }
    }
    loadContracts();
    return () => { cancelled = true; };
  }, [id, moduleTab]);

  useEffect(() => {
    if (moduleTab !== "metrics" || isObserver) return;
    let cancelled = false;
    async function loadMetrics() {
      const token = localStorage.getItem("mams_token");
      if (!token) return setMetricsStatus("Ошибка авторизации.");
      setMetricsStatus("Загрузка метрик...");
      try {
        const resp = await fetch(`/api/services/${id}/metrics`, {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
        if (!resp.ok) return setMetricsStatus("Не удалось загрузить метрики.");
        const data = await resp.json();
        if (cancelled) return;
        setMetricsEmbedURL(data?.embed_url || "");
        setMetricsStatus(data?.embed_url ? "" : "Для сервиса не задан Grafana dashboard.");
      } catch {
        if (!cancelled) setMetricsStatus("Не удалось загрузить метрики.");
      }
    }
    loadMetrics();
    return () => { cancelled = true; };
  }, [id, moduleTab, isObserver]);

  useEffect(() => {
    let cancelled = false;
    async function loadReleases() {
      const token = localStorage.getItem("mams_token");
      if (!token) return;
      try {
        const resp = await fetch(`/api/services/${id}/releases`, {
          headers: { Authorization: `Bearer ${token}` },
          cache: "no-store",
        });
        if (!resp.ok) return setReleasesStatus("История релизов недоступна.");
        const data = await resp.json();
        if (cancelled) return;
        const list = Array.isArray(data?.releases) ? data.releases : Array.isArray(data) ? data : [];
        list.sort((a, b) => new Date(b.deployed_at || b.date || 0) - new Date(a.deployed_at || a.date || 0));
        setReleases(list);
        if (activeReleaseID) {
          const current = list.find((item) => item.id === activeReleaseID);
          if (current?.status === "success") {
            setReleaseStatus("Операция успешно завершена.");
            setActiveReleaseID("");
          } else if (current?.status === "failed") {
            setReleaseStatus("Операция завершилась ошибкой.");
            setActiveReleaseID("");
          } else if (current?.status === "in_progress" || current?.status === "pending") {
            setReleaseStatus(`Операция выполняется: ${current.status}.`);
          }
        }
      } catch {
        if (!cancelled) setReleasesStatus("История релизов недоступна.");
      }
    }
    loadReleases();
    if (!activeReleaseID) {
      return () => { cancelled = true; };
    }
    const timer = setInterval(loadReleases, 2500);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [id, activeReleaseID]);

  return (
    <main className="page">
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
              <div className="service-header">
                <div>
                  <h2>{svc?.overview?.name}</h2>
                  <p className="service-subtitle">
                    {svc?.overview?.description || "Описание отсутствует"}
                  </p>
                </div>

                <div className="service-badges">
                  <span className={`importance-badge ${svc?.overview?.importance}`}>
                    {svc?.overview?.importance}
                  </span>

                  <span className="type-badge">
                    {svc?.overview?.type}
                  </span>
                </div>
              </div>
              {status && <p className="status">{status}</p>}
              {svc && (
                <>
                  <div className="service-meta-grid">
                    <div className="meta-card">
                      <span className="meta-label">Версия</span>
                      <strong>{svc?.overview?.version || "-"}</strong>
                    </div>

                    <div className="meta-card">
                      <span className="meta-label">Покрытие тестами</span>
                      <strong>{svc?.overview?.test_coverage ?? "-"}%</strong>
                    </div>

                    <div className="meta-card">
                      <span className="meta-label">Минимальное покрытие</span>
                      <strong>{coverageEnabled ? `${coverageMin}%` : "Выключено"}</strong>
                    </div>

                    <div className="meta-card">
                      <span className="meta-label">PII</span>
                      <strong>{svc?.overview?.pii_sensitive ? "Да" : "Нет"}</strong>
                    </div>

                    <div className="meta-card">
                      <span className="meta-label">Команда</span>
                      <strong>{svc?.overview?.responsible_team_ref || "-"}</strong>
                    </div>

                    <div className="meta-card">
                      <span className="meta-label">Ветка по умолчанию</span>
                      <strong>{svc?.modules?.default_branch || "-"}</strong>
                    </div>
                  </div>

                  <div className="service-links">
                    <div>
                      <span className="meta-label">Репозиторий</span>
                      {svc?.modules?.repository_url ? (
                        <a
                          href={svc.modules.repository_url}
                          target="_blank"
                          rel="noreferrer"
                          className="service-link"
                        >
                          {svc.modules.repository_url}
                        </a>
                      ) : (
                        <strong>-</strong>
                      )}
                    </div>

                    <div>
                      <span className="meta-label">Grafana UID</span>
                      <strong>{svc?.modules?.grafana_dashboard_uid || "-"}</strong>
                    </div>
                  </div>
                  {!isEditingInfo && (
                    <button type="button" className="edit-btn" onClick={() => {
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
                      try {

                        const resp = await fetch(`/api/services/${id}`, {
                          method: "PUT",
                          headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
                          body: JSON.stringify(payload),
                        });

                        if (!resp.ok) {
                          setSaveStatus("Не удалось сохранить изменения.");
                          return;
                        }

                        await resp.json().catch(() => null);

                        const fresh = await fetch(`/api/services/${id}`, {
                          headers: { Authorization: `Bearer ${token}` },
                          cache: "no-store",
                        });

                        if (fresh.ok) {
                          const freshData = await fresh.json();
                          setSvc(freshData);
                        }

                        setIsEditingInfo(false);
                      } catch {
                        setSaveStatus("Не удалось сохранить изменения.");
                      }
                    }}>
                      <label>
                        Описание
                        <textarea name="description" rows="3" defaultValue={svc?.overview?.description || ""} />
                      </label>

                      <label>
                        Тип
                        <select name="type" defaultValue={svc?.overview?.type || "business"}>
                          <option value="business">business</option>
                          <option value="composition">composition</option>
                        </select>
                      </label>

                      <label>
                        Покрытие тестами (%)
                        <input name="test_coverage" type="number" min="0" max="100" defaultValue={svc?.overview?.test_coverage ?? 0} />
                      </label>

                      <label className="checkbox-row">
                        <input name="pii_sensitive" type="checkbox" defaultChecked={Boolean(svc?.overview?.pii_sensitive)} />
                        Сервис работает с PII
                      </label>

                      <label>
                        Ссылка на команду
                        <input name="responsible_team_ref" type="text" defaultValue={svc?.overview?.responsible_team_ref || ""} />
                      </label>

                      <label>
                        Важность
                        <select name="importance" defaultValue={svc?.overview?.importance || "medium"}>
                          <option value="low">low</option>
                          <option value="medium">medium</option>
                          <option value="high">high</option>
                          <option value="critical">critical</option>
                        </select>
                      </label>

                      <label>
                        URL репозитория
                        <input name="repository_url" type="url" defaultValue={svc?.modules?.repository_url || ""} />
                      </label>

                      <label>
                        Ветка по умолчанию
                        <input name="default_branch" type="text" defaultValue={svc?.modules?.default_branch || "main"} />
                      </label>

                      <label>
                        UID Grafana dashboard
                        <input name="grafana_dashboard_uid" type="text" defaultValue={svc?.modules?.grafana_dashboard_uid || ""} />
                      </label>
                      <div className="inline-actions">
                        <button type="submit">Сохранить изменения</button>
                        <button type="button" className="ghost-btn" onClick={() => {
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
              {releasesStatus && <p className="status">{releasesStatus}</p>}
              <div className="history-scroll">
                <table className="history-table">
                  <thead>
                    <tr>
                      <th>Git tag</th>
                      <th>Branch</th>
                      <th>Strategy</th>
                      <th>Environment</th>
                      <th>Author</th>
                      <th>Status</th>
                      <th>Date</th>
                      <th>Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    {releases.length === 0 && (
                      <tr><td colSpan="8" className="status">История релизов пока пуста.</td></tr>
                    )}
                    {releases.map((r, idx) => (
                      <tr key={`${r.id || "r"}-${idx}`}>
                        <td>{r.git_tag || "-"}</td>
                        <td>{r.branch || "-"}</td>
                        <td>{r.strategy || "-"}</td>
                        <td>{r.environment || "-"}</td>
                        <td>{r.author || r.author_user_id || "-"}</td>
                        <td>
                          <span className={releaseStatusClass(r.status)}>
                            {r.status || "-"}
                          </span>
                        </td>
                        <td>{formatDateTime(r.deployed_at || r.date)}</td>                        <td>{r.description || "-"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          </div>
          <div className="overview-col">
            <section className="profile-card release-card">
              <div className="release-head">
                <div>
                  <h2>Релиз</h2>
                  <p className="release-subtitle">
                    Управление деплоем, стратегией выката и rollback.
                  </p>
                </div>
                <span className={releaseBlocked ? "release-health blocked" : "release-health ready"}>
                  {releaseBlocked ? "blocked" : "ready"}
                </span>
              </div>

              {isObserver && (
                <p className="status">У вас нет прав на управление релизами.</p>
              )}

              {!isObserver && (
                <form className="release-form">
                  <div className="release-stats">
                    <div className="release-stat">
                      <span>Окружение</span>
                      <strong>{releaseEnv}</strong>
                    </div>
                    <div className="release-stat">
                      <span>Стратегия</span>
                      <strong>{releaseStrategy}</strong>
                    </div>
                    <div className="release-stat">
                      <span>Ветка</span>
                      <strong>
                        {releaseEnv === "prod"
                          ? releaseTag || "tag required"
                          : releaseBranch || "main"}
                      </strong>
                    </div>
                  </div>

                  <div className="release-mode-switch">
                    <button
                      type="button"
                      className={releaseMode === "deploy" ? "release-mode-btn active" : "release-mode-btn"}
                      onClick={() => setReleaseMode("deploy")}
                    >
                      Deploy
                    </button>
                    <button
                      type="button"
                      className={releaseMode === "rollback" ? "release-mode-btn active" : "release-mode-btn"}
                      onClick={() => setReleaseMode("rollback")}
                    >
                      Rollback
                    </button>
                  </div>

                  <div className="release-fields">
                    <label>
                      <span>Стратегия</span>
                      <select value={releaseStrategy} onChange={(e) => setReleaseStrategy(e.target.value)} disabled={releaseMode === "rollback"}>
                        <option value="rolling">rolling</option>
                        <option value="recreate">recreate</option>
                        <option value="canary">canary</option>
                      </select>
                    </label>

                    <label>
                      <span>Окружение</span>
                      <select value={releaseEnv} onChange={(e) => setReleaseEnv(e.target.value)} disabled={releaseMode === "rollback"}>
                        <option value="dev">dev</option>
                        <option value="staging">staging</option>
                        <option value="prod">prod</option>
                      </select>
                    </label>

                    {releaseMode === "deploy" && releaseEnv !== "prod" && (
                      <label>
                        <span>Ветка</span>
                        <input type="text" value={releaseBranch} onChange={(e) => setReleaseBranch(e.target.value)} />
                      </label>
                    )}

                    {releaseMode === "deploy" && releaseEnv === "prod" && (
                      <label>
                        <span>Git tag</span>
                        <input type="text" value={releaseTag} onChange={(e) => setReleaseTag(e.target.value)} placeholder="v1.2.3" />
                      </label>
                    )}

                    {releaseMode === "rollback" && (
                      <label className="release-field-full">
                        <span>Версия для rollback</span>
                        <select value={rollbackTag} onChange={(e) => setRollbackTag(e.target.value)}>
                          <option value="">Выберите git tag</option>
                          {rollbackCandidates.map((tag) => (
                            <option key={tag} value={tag}>{tag}</option>
                          ))}
                        </select>
                      </label>
                    )}

                    <label className="release-field-full">
                      <span>Описание</span>
                      <input type="text" value={releaseDescription} onChange={(e) => setReleaseDescription(e.target.value)} disabled={releaseMode === "rollback"} />
                    </label>
                  </div>

                  {releaseBlocked && (
                    <div className="release-warning">
                      {releaseBlockedHint}
                    </div>
                  )}

                  <div className="release-footer">
                    <button
                      type="button"
                      className={releaseBlocked ? "release-primary disabled" : "release-primary"}
                      disabled={releaseBlocked}
                      onClick={async () => {
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
                            const data = await resp.json();
                            if (data?.release_id) setActiveReleaseID(data.release_id);
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
                          const data = await resp.json();
                          if (data?.release_id) setActiveReleaseID(data.release_id);
                          return setReleaseStatus("Деплой запущен.");
                        } catch {
                          setReleaseStatus("Операция недоступна.");
                        }
                      }}
                    >
                      {releaseMode === "rollback" ? "Запустить rollback" : "Запустить деплой"}
                    </button>
                  </div>

                  {releaseStatus && <p className="release-status-text">{releaseStatus}</p>}
                </form>
              )}
            </section>
            <section className="profile-card">
              <h2>Модули</h2>
              <p className="status">Роль: <span className="role-badge">{effectiveRole}</span></p>
              <div className="module-tabs">
                {!isObserver && (
                  <>
                    <button type="button" className={moduleTab === "metrics" ? "module-tab active" : "module-tab"} onClick={() => setModuleTab("metrics")}>Метрики</button>
                    <button type="button" className={moduleTab === "logs" ? "module-tab active" : "module-tab"} onClick={() => setModuleTab("logs")}>Логи</button>
                  </>
                )}
                <button type="button" className={moduleTab === "contracts" ? "module-tab active" : "module-tab"} onClick={() => setModuleTab("contracts")}>Контракты</button>
              </div>
              <div className="modules-grid">
                {moduleTab === "contracts" && (
                  <article className="module-card">
                    {contractsStatus && <p className="status">{contractsStatus}</p>}
                    <div className="contracts-pane">
                      {contracts.map((m, idx) => {
                        const inOpen = openContractPanels[m.name || idx] === "in";
                        const outOpen = openContractPanels[m.name || idx] === "out";
                        return (
                          <div key={`${m.name || "m"}-${idx}`} className="contract-method">
                            <div className="contract-method-head">
                              <strong>{m.name}</strong>
                              <div className="contract-head-actions">
                                <button
                                  type="button"
                                  className={inOpen ? "contract-io-btn active" : "contract-io-btn"}
                                  onClick={() => setOpenContractPanels((prev) => {
                                    const key = m.name || idx;
                                    const next = { ...prev };
                                    next[key] = prev[key] === "in" ? "" : "in";
                                    return next;
                                  })}
                                >
                                  IN
                                </button>
                                <button
                                  type="button"
                                  className={outOpen ? "contract-io-btn active" : "contract-io-btn"}
                                  onClick={() => setOpenContractPanels((prev) => {
                                    const key = m.name || idx;
                                    const next = { ...prev };
                                    next[key] = prev[key] === "out" ? "" : "out";
                                    return next;
                                  })}
                                >
                                  OUT
                                </button>
                              </div>
                            </div>
                            {inOpen && (
                              <div className="contract-expand">
                                <div className="contract-expand-title">Вход: {m.input || "-"}</div>
                                {Array.isArray(m.parameters) && m.parameters.length > 0 ? renderContractTree(m.parameters) : <div className="contract-empty">Параметры не указаны</div>}
                              </div>
                            )}
                            {outOpen && (
                              <div className="contract-expand">
                                <div className="contract-expand-title">Выход: {m.output || "-"}</div>
                                {Array.isArray(m.output_parameters) && m.output_parameters.length > 0 ? renderContractTree(m.output_parameters) : <div className="contract-empty">Поля ответа не указаны</div>}
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                    {contracts.length === 0 && !contractsStatus && (
                      <div className="contract-empty">Контракты не найдены.</div>
                    )}
                  </article>
                )}
                {moduleTab === "metrics" && !isObserver && (
                  <article className="module-card">
                    {metricsStatus && <p className="status">{metricsStatus}</p>}
                    {metricsEmbedURL && (
                      <iframe
                        className="metrics-frame"
                        src={metricsEmbedURL}
                        title="metrics"
                        loading="lazy"
                      />
                    )}
                  </article>
                )}
                {moduleTab === "logs" && !isObserver && (
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
                          <div key={`${getLogKey(item)}-${idx}`} className="log-row">
                            <span className="log-time">{formatDateTime(item.timestamp)}</span>
                            <span className={logLevelClass(item.level)}>
                              {item.level || "-"}
                            </span>
                            <span className="log-msg">{item.message || "-"}</span>
                          </div>
                        ))}
                      </div>
                    )}
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
          {isObserver ? (
            <p className="status">Вкладка недоступна для роли observer.</p>
          ) : (
            <>
              <div className="settings-view">
                <p><strong>Минимальный порог включен:</strong> {settingsEnabled ? "да" : "нет"}</p>
                <p><strong>Минимальный порог покрытия:</strong> {settingsMinCoverage}%</p>
              </div>
              {!isEditingSettings && (
                <button
                  type="button"
                  className="edit-btn"
                  disabled={isDeveloper}
                  onClick={() => {
                    setSettingsStatus("");
                    setIsEditingSettings(true);
                  }}
                >
                  Редактировать
                </button>
              )}
              {isDeveloper && (
                <p className="status">Режим только чтение: owner-only настройки может менять только Service Owner.</p>
              )}
              {isEditingSettings && (
                <form className="inline-form settings-form" onSubmit={async (event) => {
                  event.preventDefault();
                  if (!isOwner) return setSettingsStatus("Изменение owner-only настроек доступно только Service Owner.");
                  const token = localStorage.getItem("mams_token");
                  if (!token) return setSettingsStatus("Ошибка авторизации.");
                  try {
                    const payload = {
                      settings: {
                        minimum_test_coverage_enabled: settingsEnabled,
                        minimum_test_coverage: Number(settingsMinCoverage),
                      },
                    };
                    const resp = await fetch(`/api/services/${id}/settings`, {
                      method: "PUT",
                      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
                      body: JSON.stringify(payload),
                    });
                    if (!resp.ok) return setSettingsStatus("Не удалось сохранить настройки.");
                    setSvc((prev) => (prev ? ({
                      ...prev,
                      minimum_test_coverage_enabled: settingsEnabled,
                      minimum_test_coverage: Number(settingsMinCoverage),
                    }) : prev));
                    const fresh = await fetch(`/api/services/${id}`, {
                      headers: { Authorization: `Bearer ${token}` },
                      cache: "no-store",
                    });
                    if (fresh.ok) {
                      const freshData = await fresh.json();
                      setSvc(freshData);
                      setSettingsEnabled(Boolean(freshData.minimum_test_coverage_enabled));
                      setSettingsMinCoverage(Number(freshData.minimum_test_coverage || 0));
                    }
                    setSettingsStatus("Настройки сохранены.");
                    setIsEditingSettings(false);
                  } catch {
                    setSettingsStatus("Не удалось сохранить настройки.");
                  }
                }}>
                  <label className="checkbox-row">
                    <input
                      type="checkbox"
                      checked={settingsEnabled}
                      onChange={(e) => setSettingsEnabled(e.target.checked)}
                    />
                    Задать минимальный порог покрытия
                  </label>
                  <label>
                    Минимальный порог покрытия (%)
                    <input
                      type="number"
                      min="0"
                      max="100"
                      value={settingsMinCoverage}
                      disabled={!settingsEnabled}
                      onChange={(e) => setSettingsMinCoverage(Number(e.target.value || 0))}
                    />
                  </label>
                  <div className="inline-actions">
                    <button type="submit">Сохранить настройки</button>
                    <button
                      type="button"
                      className="ghost-btn"
                      onClick={() => {
                        setSettingsStatus("");
                        setIsEditingSettings(false);
                      }}
                    >
                      Отмена
                    </button>
                  </div>
                </form>
              )}
              <p className="status">{settingsStatus}</p>
            </>
          )}
        </section>
      )}
    </main>
  );
}
