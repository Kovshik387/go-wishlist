import type {
  BrandConfig,
  NotificationPreferences,
  Reservation,
  User,
  Wish,
  Wishlist,
} from "../types";

const TOKEN_KEY = "wishtrack_access";
let accessToken = sessionStorage.getItem(TOKEN_KEY) ?? "";
let refreshPromise: Promise<boolean> | null = null;

export class APIError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

type RequestOptions = RequestInit & { skipAuth?: boolean; retry?: boolean };

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (!(options.body instanceof FormData)) headers.set("Content-Type", "application/json");
  if (accessToken && !options.skipAuth) headers.set("Authorization", `Bearer ${accessToken}`);
  const response = await fetch(path, {
    ...options,
    headers,
    credentials: "include",
  });
  if (response.status === 401 && !options.skipAuth && options.retry !== false) {
    const refreshed = await refreshAccess();
    if (refreshed) return request<T>(path, { ...options, retry: false });
  }
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: {} }));
    throw new APIError(
      response.status,
      body.error?.code ?? "UNKNOWN",
      body.error?.message ?? "Не удалось выполнить запрос",
    );
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

function json(method: string, body?: unknown): RequestInit {
  return { method, body: body === undefined ? undefined : JSON.stringify(body) };
}

function saveToken(token: string) {
  accessToken = token;
  sessionStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  accessToken = "";
  sessionStorage.removeItem(TOKEN_KEY);
}

async function refreshAccess() {
  if (!refreshPromise) {
    refreshPromise = fetch("/api/v1/auth/refresh", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    })
      .then(async (response) => {
        if (!response.ok) {
          clearToken();
          return false;
        }
        const body = (await response.json()) as { accessToken: string };
        saveToken(body.accessToken);
        return true;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

type AuthResponse = { accessToken: string; expiresAt: string; user: User };

export const api = {
  config: () => request<BrandConfig>("/api/v1/config", { skipAuth: true }),
  telegramAuth: async (initData: string) => {
    const response = await request<AuthResponse>(
      "/api/v1/auth/telegram",
      { ...json("POST", { initData }), skipAuth: true },
    );
    saveToken(response.accessToken);
    return response;
  },
  devAuth: async (displayName = "Аня") => {
    const response = await request<AuthResponse>(
      "/api/v1/auth/dev",
      { ...json("POST", { displayName }), skipAuth: true },
    );
    saveToken(response.accessToken);
    return response;
  },
  logout: async () => {
    await request<void>("/api/v1/auth/logout", json("POST"));
    clearToken();
  },
  me: () => request<User>("/api/v1/me"),
  updateMe: (body: Pick<User, "displayName" | "timezone"> & { onboardingCompleted?: boolean }) =>
    request<User>("/api/v1/me", json("PATCH", body)),
  deleteMe: () => request<void>("/api/v1/me", json("DELETE")),
  feed: () => request<{ items: Wish[] }>("/api/v1/feed"),
  wishlists: () => request<{ items: Wishlist[]; saved: Wishlist[] }>("/api/v1/wishlists"),
  wishlist: (id: string) => request<Wishlist>(`/api/v1/wishlists/${id}`),
  publicWishlist: (token: string) =>
    request<Wishlist>(`/api/v1/public/wishlists/${encodeURIComponent(token)}`),
  createWishlist: (body: Partial<Wishlist>) =>
    request<Wishlist>("/api/v1/wishlists", json("POST", body)),
  updateWishlist: (id: string, body: Partial<Wishlist>) =>
    request<Wishlist>(`/api/v1/wishlists/${id}`, json("PATCH", body)),
  deleteWishlist: (id: string) =>
    request<void>(`/api/v1/wishlists/${id}`, json("DELETE")),
  rotateLink: (id: string) =>
    request<Wishlist>(`/api/v1/wishlists/${id}/rotate-link`, json("POST")),
  wish: (listID: string, wishID: string) =>
    request<Wish>(`/api/v1/wishlists/${listID}/wishes/${wishID}`),
  createWish: (listID: string, body: Partial<Wish>) =>
    request<Wish>(`/api/v1/wishlists/${listID}/wishes`, json("POST", body)),
  updateWish: (listID: string, wishID: string, body: Partial<Wish>) =>
    request<Wish>(`/api/v1/wishlists/${listID}/wishes/${wishID}`, json("PATCH", body)),
  deleteWish: (listID: string, wishID: string) =>
    request<void>(`/api/v1/wishlists/${listID}/wishes/${wishID}`, json("DELETE")),
  preview: (url: string) =>
    request<Partial<Wish>>("/api/v1/wishes/preview-url", json("POST", { url })),
  follow: (userID: string) =>
    request<void>(`/api/v1/users/${userID}/follow`, json("POST")),
  unfollow: (userID: string) =>
    request<void>(`/api/v1/users/${userID}/follow`, json("DELETE")),
  user: (id: string) =>
    request<{ user: User; following: boolean }>(`/api/v1/users/${id}`),
  mute: (userID: string, muted: boolean) =>
    request<void>(`/api/v1/users/${userID}/notification-settings`, json("PATCH", { muted })),
  reserve: (wishID: string) =>
    request<Reservation>(`/api/v1/wishes/${wishID}/reservation`, json("POST")),
  cancelReservation: (wishID: string) =>
    request<void>(`/api/v1/wishes/${wishID}/reservation`, json("DELETE")),
  reservations: () => request<{ items: Reservation[] }>("/api/v1/reservations"),
  notificationSettings: () =>
    request<NotificationPreferences>("/api/v1/notification-settings"),
  updateNotificationSettings: (body: NotificationPreferences) =>
    request<NotificationPreferences>("/api/v1/notification-settings", json("PATCH", body)),
  upload: async (file: File) => {
    const body = new FormData();
    body.set("file", file);
    return request<{ publicUrl: string }>("/api/v1/media", { method: "POST", body });
  },
  syncMedia: () =>
    request<{ attempted: number; synced: number; failed: number }>(
      "/api/v1/media/sync",
      json("POST"),
    ),
};
