import { Navigate, Route, Routes } from "react-router-dom";
import { RequireAuth } from "./components/RequireAuth";
import { LoginPage } from "./pages/LoginPage";
import { NewServicePage } from "./pages/NewServicePage";
import { ProfilePage } from "./pages/ProfilePage";
import { ServicePage } from "./pages/ServicePage";
import { ServicesPage } from "./pages/ServicesPage";
import "./styles.css";

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route element={<RequireAuth />}>
        <Route path="/services" element={<ServicesPage />} />
        <Route path="/services/new" element={<NewServicePage />} />
        <Route path="/services/:id" element={<ServicePage />} />
        <Route path="/profile" element={<ProfilePage />} />
      </Route>
    </Routes>
  );
}
