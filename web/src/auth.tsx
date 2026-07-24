import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { api, APIError } from "./lib/api";
import { telegram } from "./lib/telegram";
import type { User } from "./types";

type AuthState = {
  user: User;
  setUser: (user: User) => void;
};

const AuthContext = createContext<AuthState | null>(null);

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    async function authenticate() {
      try {
        let current: User;
        if (telegram?.initData) {
          try {
            current = (await api.telegramAuth(telegram.initData)).user;
          } catch {
            current = await api.me();
          }
        } else {
          try {
            current = await api.me();
          } catch {
            current = (await api.devAuth()).user;
          }
        }
        if (active) {
          setUser(current);
          void api.syncMedia()
            .then((result) => {
              if (active && result.synced > 0) void queryClient.invalidateQueries();
            })
            .catch(() => undefined);
        }
      } catch (caught) {
        if (!active) return;
        if (caught instanceof APIError && caught.status === 404 && !telegram) {
          setError("Откройте приложение из Telegram — так мы безопасно узнаем, что это вы.");
        } else {
          setError(caught instanceof Error ? caught.message : "Не удалось войти");
        }
      }
    }
    void authenticate();
    return () => {
      active = false;
    };
  }, [queryClient]);

  const context = useMemo(() => (user ? { user, setUser } : null), [user]);
  if (error) {
    return (
      <main className="launch-screen">
        <div className="launch-orb launch-orb--one" />
        <div className="launch-card">
          <span className="launch-logo">W</span>
          <p className="eyebrow">WishTrack</p>
          <h1>Лучше внутри Telegram</h1>
          <p>{error}</p>
          <button className="button button--primary" onClick={() => window.location.reload()}>
            Попробовать снова
          </button>
        </div>
      </main>
    );
  }
  if (!context) {
    return (
      <main className="launch-screen" aria-label="Загрузка приложения">
        <div className="launch-logo launch-logo--pulse">W</div>
        <h1>WishTrack</h1>
        <p>Собираем желания…</p>
        <div className="loader" />
      </main>
    );
  }
  return <AuthContext.Provider value={context}>{children}</AuthContext.Provider>;
}
