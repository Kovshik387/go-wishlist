import { useEffect } from "react";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { AuthProvider, useAuth } from "./auth";
import { Onboarding } from "./components/Onboarding";
import { Shell } from "./components/Shell";
import { FeedPage } from "./pages/FeedPage";
import { ListsPage } from "./pages/ListsPage";
import { ProfileEditPage } from "./pages/ProfileEditPage";
import { ProfilePage } from "./pages/ProfilePage";
import { ReservationsPage } from "./pages/ReservationsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { UserPage } from "./pages/UserPage";
import { WishFormPage } from "./pages/WishFormPage";
import { WishlistFormPage } from "./pages/WishlistFormPage";
import { WishlistPage } from "./pages/WishlistPage";
import { getStartParam } from "./lib/telegram";

export function App() {
  return (
    <AuthProvider>
      <AuthenticatedApp />
    </AuthProvider>
  );
}

function AuthenticatedApp() {
  const { user } = useAuth();
  const navigate = useNavigate();
  useEffect(() => {
    const start = getStartParam();
    if (start.startsWith("wishlist_")) {
      navigate(`/public/${start.slice("wishlist_".length)}`, { replace: true });
    }
  }, [navigate]);
  return (
    <>
      <Routes>
        <Route element={<Shell />}>
          <Route index element={<FeedPage />} />
          <Route path="lists" element={<ListsPage />} />
          <Route path="lists/new" element={<WishlistFormPage />} />
          <Route path="lists/:id" element={<WishlistPage />} />
          <Route path="lists/:id/edit" element={<WishlistFormPage />} />
          <Route path="lists/:id/wishes/new" element={<WishFormPage />} />
          <Route path="lists/:id/wishes/:wishId/edit" element={<WishFormPage />} />
          <Route path="public/:token" element={<WishlistPage publicView />} />
          <Route path="reservations" element={<ReservationsPage />} />
          <Route path="profile" element={<ProfilePage />} />
          <Route path="profile/edit" element={<ProfileEditPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="users/:id" element={<UserPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
      {!user.onboardingCompleted && <Onboarding />}
    </>
  );
}
