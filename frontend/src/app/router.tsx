import { Navigate, Route, Routes } from "react-router-dom";
import { AppLayout } from "./layout/AppLayout";
import { DoctorPage } from "../pages/DoctorPage";
import { MemoryPage } from "../pages/MemoryPage";
import { RunsPage } from "../pages/RunsPage";

export function AppRouter() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/runs" replace />} />
        <Route path="runs" element={<RunsPage />} />
        <Route path="memory" element={<MemoryPage />} />
        <Route path="doctor" element={<DoctorPage />} />
      </Route>
    </Routes>
  );
}
