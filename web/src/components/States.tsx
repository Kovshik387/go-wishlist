import type { ReactNode } from "react";

export function PageLoading({ cards = 3 }: { cards?: number }) {
  return (
    <div className="skeleton-list" aria-label="Загрузка">
      {Array.from({ length: cards }).map((_, index) => (
        <div className="skeleton-card" key={index}>
          <div className="skeleton skeleton--image" />
          <div className="skeleton skeleton--title" />
          <div className="skeleton skeleton--line" />
        </div>
      ))}
    </div>
  );
}

export function EmptyState({
  visual,
  title,
  text,
  action,
}: {
  visual: ReactNode;
  title: string;
  text: string;
  action?: ReactNode;
}) {
  return (
    <section className="empty-state">
      <div className="empty-state__art" aria-hidden="true">{visual}</div>
      <h2>{title}</h2>
      <p>{text}</p>
      {action}
    </section>
  );
}

export function ErrorState({
  message = "Не удалось загрузить данные",
  retry,
}: {
  message?: string;
  retry: () => void;
}) {
  return (
    <section className="empty-state">
      <div className="empty-state__art empty-state__art--error" aria-hidden="true">↻</div>
      <h2>Что-то не сложилось</h2>
      <p>{message}</p>
      <button className="button button--secondary" onClick={retry}>Попробовать снова</button>
    </section>
  );
}
