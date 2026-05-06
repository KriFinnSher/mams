import { Link, Navigate, Route, Routes, useParams } from "react-router-dom";
import "./styles.css";

function Layout({ title }) {
  return (
    <main className="page">
      <h1>{title}</h1>
      <nav className="nav">
        <Link to="/login">Логин</Link>
        <Link to="/services">Сервисы</Link>
        <Link to="/services/new">Новый сервис</Link>
        <Link to="/profile">Профиль</Link>
      </nav>
    </main>
  );
}

function LoginPage() {
  return <Layout title="Страница входа" />;
}

function ServicesPage() {
  return <Layout title="Список сервисов" />;
}

function NewServicePage() {
  return <Layout title="Создание сервиса" />;
}

function ProfilePage() {
  return <Layout title="Профиль пользователя" />;
}

function ServicePage() {
  const { id } = useParams();
  return <Layout title={`Карточка сервиса: ${id}`} />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/services" element={<ServicesPage />} />
      <Route path="/services/new" element={<NewServicePage />} />
      <Route path="/services/:id" element={<ServicePage />} />
      <Route path="/profile" element={<ProfilePage />} />
    </Routes>
  );
}
