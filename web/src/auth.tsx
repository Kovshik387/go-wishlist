import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
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
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    async function authenticate() {
      try {
        let current: User;
        try {
          current = await api.me();
        } catch {
          if (telegram?.initData) {
            current = (await api.telegramAuth(telegram.initData)).user;
          } else {
            current = (await api.devAuth()).user;
          }
        }
        if (active) setUser(current);
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
  }, []);

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
