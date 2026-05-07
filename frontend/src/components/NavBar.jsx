import { Link } from "react-router-dom";

export function NavBar() {
  return (
    <nav className="nav">
      <Link to="/login">Логин</Link>
      <Link to="/services">Сервисы</Link>
      <Link to="/services/new">Новый сервис</Link>
      <Link to="/profile">Профиль</Link>
    </nav>
  );
}
