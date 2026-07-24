type TelegramThemeParams = {
  bg_color?: string;
  text_color?: string;
  hint_color?: string;
  link_color?: string;
  button_color?: string;
  button_text_color?: string;
  secondary_bg_color?: string;
};

type TelegramWebApp = {
  initData: string;
  colorScheme: "light" | "dark";
  themeParams: TelegramThemeParams;
  isExpanded: boolean;
  ready(): void;
  expand(): void;
  close(): void;
  requestWriteAccess(callback?: (allowed: boolean) => void): void;
  openTelegramLink(url: string): void;
  showConfirm(message: string, callback?: (confirmed: boolean) => void): void;
  HapticFeedback?: {
    impactOccurred(style: "light" | "medium" | "heavy"): void;
    notificationOccurred(type: "error" | "success" | "warning"): void;
  };
};

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp };
  }
}

export const telegram = window.Telegram?.WebApp;

export function initTelegram() {
  telegram?.ready();
  telegram?.expand();
  if (telegram?.colorScheme) {
    document.documentElement.dataset.theme = telegram.colorScheme;
  }
}

export function getStartParam() {
  const signed = new URLSearchParams(telegram?.initData ?? "").get("start_param");
  return signed ?? new URLSearchParams(window.location.search).get("tgWebAppStartParam") ?? "";
}

export function shareTelegram(url: string, text: string) {
  const shareURL = `https://t.me/share/url?url=${encodeURIComponent(url)}&text=${encodeURIComponent(text)}`;
  if (telegram) {
    telegram.openTelegramLink(shareURL);
    return;
  }
  if (navigator.share) {
    void navigator.share({ url, text });
    return;
  }
  void navigator.clipboard.writeText(url);
}

export function haptic(type: "success" | "warning" | "error" = "success") {
  telegram?.HapticFeedback?.notificationOccurred(type);
}
