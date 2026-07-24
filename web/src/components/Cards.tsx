import { useId, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import { api, APIError } from "../lib/api";
import { haptic } from "../lib/telegram";
import type { Wish, Wishlist } from "../types";
import { Icon } from "./Icon";

const visibilityLabels = {
  public: "Виден всем",
  link: "Только по ссылке",
  private: "Только мне",
};

export function WishlistCard({ list }: { list: Wishlist }) {
  return (
    <Link to={`/lists/${list.id}`} className="wishlist-card">
      <div className="wishlist-card__cover">
        {list.coverUrl ? <img src={list.coverUrl} alt="" referrerPolicy="no-referrer" /> : <Icon name="lists" size={32} />}
      </div>
      <div className="wishlist-card__body">
        <div className="wishlist-card__topline">
          <span className={`visibility-dot visibility-dot--${list.visibility}`} />
          <span>{visibilityLabels[list.visibility]}</span>
        </div>
        <h2>{list.title}</h2>
        <p>{list.description || "Здесь скоро появятся мечты"}</p>
        <div className="wishlist-card__meta">
          <span>{pluralWishes(list.wishCount)}</span>
          {list.eventDate && <span>{formatDate(list.eventDate)}</span>}
        </div>
      </div>
      <span className="wishlist-card__arrow"><Icon name="arrow" /></span>
    </Link>
  );
}

export function WishCard({
  wish,
  canReserve = true,
  owner = false,
  compact = false,
  invalidate = [],
}: {
  wish: Wish;
  canReserve?: boolean;
  owner?: boolean;
  compact?: boolean;
  invalidate?: string[];
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [commentOpen, setCommentOpen] = useState(false);
  const commentID = useId();
  const mutation = useMutation({
    mutationFn: async () => {
      if (wish.reservedByMe) {
        await api.cancelReservation(wish.id);
      } else {
        await api.reserve(wish.id);
      }
    },
    onSuccess: async () => {
      haptic("success");
      await Promise.all(invalidate.map((key) => queryClient.invalidateQueries({ queryKey: [key] })));
      await queryClient.invalidateQueries({ queryKey: ["wishlist", wish.wishlistId] });
      await queryClient.invalidateQueries({ queryKey: ["feed"] });
      await queryClient.invalidateQueries({ queryKey: ["reservations"] });
    },
  });
  const target = owner
    ? `/lists/${wish.wishlistId}/wishes/${wish.id}/edit`
    : wish.productUrl || "";
  return (
    <article className={`wish-card ${compact ? "wish-card--compact" : ""}`}>
      <button
        className="wish-card__image"
        onClick={() => target && (owner ? navigate(target) : window.open(target, "_blank", "noopener"))}
        aria-label={`Открыть ${wish.title}`}
      >
        {wish.imageUrl
          ? <img src={wish.imageUrl} alt="" loading="lazy" referrerPolicy="no-referrer" />
          : <div className="wish-card__placeholder"><Icon name="gift" size={32} /></div>}
        {wish.priority === "high" && <span className="wish-card__priority">Приоритет</span>}
        {wish.isReserved && !owner && <span className="wish-card__reserved">Уже занят</span>}
      </button>
      <div className="wish-card__body">
        {(wish.author || wish.wishlist) && (
          <p className="wish-card__context">
            {wish.author?.displayName}
            {wish.author && wish.wishlist && " · "}
            {wish.wishlist?.title}
          </p>
        )}
        <h3>{wish.title}</h3>
        <div className="wish-card__details">
          <strong>{formatPrice(wish.priceMinor, wish.currency)}</strong>
          {wish.storeDomain && <span>{wish.storeDomain}</span>}
        </div>
        {wish.description && (
          <div className="wish-card__comment">
            <button
              type="button"
              className="wish-card__comment-toggle"
              aria-expanded={commentOpen}
              aria-controls={commentID}
              onClick={() => setCommentOpen((open) => !open)}
            >
              {commentOpen ? "Скрыть комментарий" : "Показать комментарий"}
            </button>
            {commentOpen && <p className="wish-card__description" id={commentID}>{wish.description}</p>}
          </div>
        )}
        <div className="wish-card__actions">
          {owner ? (
            <button className="button button--soft button--small" onClick={() => navigate(target)}>
              <Icon name="edit" size={17} /> Изменить
            </button>
          ) : canReserve ? (
            <button
              className={`button button--small ${wish.reservedByMe ? "button--soft" : "button--secondary"}`}
              disabled={(wish.isReserved && !wish.reservedByMe) || mutation.isPending}
              onClick={() => mutation.mutate()}
            >
              <Icon name={wish.reservedByMe ? "check" : "gift"} size={17} />
              {mutation.isPending
                ? "Секунду…"
                : wish.reservedByMe
                  ? "Забронировано вами"
                  : wish.isReserved
                    ? "Уже забронировали"
                    : "Забронировать"}
            </button>
          ) : null}
          {wish.productUrl && (
            <a className="icon-button" href={wish.productUrl} target="_blank" rel="noreferrer" aria-label="Открыть магазин">
              <Icon name="external" size={18} />
            </a>
          )}
        </div>
        {mutation.isError && (
          <p className="form-error">
            {mutation.error instanceof APIError ? mutation.error.message : "Не удалось изменить бронь"}
          </p>
        )}
      </div>
    </article>
  );
}

export function Avatar({ name, src, size = "medium" }: { name: string; src?: string; size?: "small" | "medium" | "large" }) {
  return (
    <span className={`avatar avatar--${size}`}>
      {src ? <img src={src} alt="" referrerPolicy="no-referrer" /> : name.slice(0, 1).toUpperCase()}
    </span>
  );
}

export function formatPrice(price?: number, currency = "RUB") {
  if (price === undefined || price === null) return "Цена не указана";
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency,
    maximumFractionDigits: price % 100 === 0 ? 0 : 2,
  }).format(price / 100);
}

export function formatDate(date: string) {
  return new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "long" }).format(new Date(date));
}

function pluralWishes(count: number) {
  const mod10 = count % 10;
  const mod100 = count % 100;
  const noun = mod10 === 1 && mod100 !== 11 ? "желание" : mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14) ? "желания" : "желаний";
  return `${count} ${noun}`;
}
