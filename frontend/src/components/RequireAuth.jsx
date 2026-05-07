import { Navigate, Outlet } from "react-router-dom";

export function RequireAuth() {
  const token = localStorage.getItem("mams_token");
  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
