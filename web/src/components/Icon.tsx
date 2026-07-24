export type IconName =
  | "feed"
  | "lists"
  | "gift"
  | "profile"
  | "plus"
  | "arrow"
  | "share"
  | "heart"
  | "settings"
  | "check"
  | "external"
  | "edit"
  | "bell"
  | "lock"
  | "close";

export function Icon({ name, size = 22 }: { name: IconName; size?: number }) {
  const paths: Record<IconName, React.ReactNode> = {
    feed: <><path d="M4 6.5h16M4 12h16M4 17.5h10" /><circle cx="18" cy="17.5" r="2" /></>,
    lists: <><rect x="3.5" y="4" width="17" height="16" rx="3" /><path d="M8 9h8M8 14h5" /></>,
    gift: <><path d="M4 10h16v10H4zM3 7h18v4H3zM12 7v13M12 7H8.8A2.3 2.3 0 1 1 11 4.2L12 7Zm0 0h3.2A2.3 2.3 0 1 0 13 4.2L12 7Z" /></>,
    profile: <><circle cx="12" cy="8" r="4" /><path d="M4.5 20c.8-4.1 3.3-6 7.5-6s6.7 1.9 7.5 6" /></>,
    plus: <path d="M12 5v14M5 12h14" />,
    arrow: <path d="m9 18 6-6-6-6" />,
    share: <><circle cx="18" cy="5" r="2.5" /><circle cx="6" cy="12" r="2.5" /><circle cx="18" cy="19" r="2.5" /><path d="m8.2 10.8 7.6-4.5M8.2 13.2l7.6 4.5" /></>,
    heart: <path d="M20.8 5.7c-2-2.2-5.2-2.1-7.2 0L12 7.4l-1.6-1.7c-2-2.1-5.2-2.2-7.2 0-2.2 2.4-2 6 .3 8.3l8.5 7 8.5-7c2.3-2.3 2.5-5.9.3-8.3Z" />,
    settings: <><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-2.8 2.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6v.2h-4V21a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1L4.2 17l.1-.1a1.7 1.7 0 0 0 .3-1.9A1.7 1.7 0 0 0 3 14H2.8v-4H3a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9L4.2 7 7 4.2l.1.1A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-1.6v-.2h4V3a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1L19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.2v4H21a1.7 1.7 0 0 0-1.6 1Z" /></>,
    check: <path d="m5 12 4.5 4.5L19 7" />,
    external: <><path d="M14 4h6v6M20 4l-9 9" /><path d="M18 13v6H5V6h6" /></>,
    edit: <><path d="m14 5 5 5L9 20H4v-5L14 5Z" /><path d="m12 7 5 5" /></>,
    bell: <><path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 8h18c0-1-3-1-3-8Z" /><path d="M10 21h4" /></>,
    lock: <><rect x="4" y="10" width="16" height="11" rx="3" /><path d="M8 10V7a4 4 0 0 1 8 0v3" /></>,
    close: <path d="m6 6 12 12M18 6 6 18" />,
  };
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {paths[name]}
    </svg>
  );
}
