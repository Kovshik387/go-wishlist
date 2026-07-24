import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useAuth } from "../auth";
import { Avatar, formatDate, WishCard } from "../components/Cards";
import { Icon } from "../components/Icon";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, PageLoading } from "../components/States";
import { api } from "../lib/api";
import { shareTelegram } from "../lib/telegram";
import type { BrandConfig, Wishlist } from "../types";

export function WishlistPage({ publicView = false }: { publicView?: boolean }) {
  const { id = "", token = "" } = useParams();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const query = useQuery({
    queryKey: publicView ? ["public-wishlist", token] : ["wishlist", id],
    queryFn: () => publicView ? api.publicWishlist(token) : api.wishlist(id),
  });
  const configQuery = useQuery({ queryKey: ["config"], queryFn: api.config });
  const followMutation = useMutation({
    mutationFn: (ownerID: string) => api.follow(ownerID),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["feed"] }),
  });
  if (query.isPending) return <div className="page"><PageLoading /></div>;
  if (query.isError) return <div className="page"><ErrorState message={query.error.message} retry={() => void query.refetch()} /></div>;
  const list = query.data;
  const owner = list.ownerId === user.id;
  const share = () => shareList(list, configQuery.data);
  return (
    <div className="page page--detail">
      <PageHeader
        back
        title={list.title}
        eyebrow={owner ? "Ваш список" : `Список · ${list.owner?.displayName ?? "друга"}`}
        action={
          <button className="icon-button" onClick={share} aria-label="Поделиться">
            <Icon name="share" />
          </button>
        }
      />
      <section className="list-hero">
        {list.coverUrl && <img className="list-hero__cover" src={list.coverUrl} alt="" />}
        <div className="list-hero__content">
          <span className="list-hero__icon"><Icon name="lists" size={30} /></span>
          <div>
            <p className="eyebrow">{visibilityText(list.visibility)}</p>
            <h1>{list.title}</h1>
            {list.description && <p>{list.description}</p>}
          </div>
        </div>
        <div className="list-hero__meta">
          {list.eventDate && <span>◷ {formatDate(list.eventDate)}</span>}
          <span>{list.wishCount} желаний</span>
        </div>
      </section>
      {!owner && list.owner && (
        <section className="owner-strip">
          <Avatar name={list.owner.displayName} src={list.owner.avatarUrl} />
          <div><strong>{list.owner.displayName}</strong><span>@{list.owner.username || "без username"}</span></div>
          <button
            className="button button--soft button--small"
            disabled={followMutation.isPending}
            onClick={() => followMutation.mutate(list.ownerId)}
          >
            <Icon name="heart" size={17} /> Подписаться
          </button>
        </section>
      )}
      <div className="detail-actions">
        {owner && (
          <>
            <Link className="button button--primary" to={`/lists/${list.id}/wishes/new`}>
              <Icon name="plus" size={18} /> Добавить желание
            </Link>
            <Link className="icon-button" to={`/lists/${list.id}/edit`} aria-label="Редактировать список">
              <Icon name="edit" />
            </Link>
          </>
        )}
        {!owner && <button className="button button--primary" onClick={share}><Icon name="share" size={18} /> Переслать другу</button>}
      </div>
      {list.wishes?.length === 0 && (
        <EmptyState
          visual={<Icon name={owner ? "plus" : "lists"} size={34} />}
          title={owner ? "Чего хочется?" : "Пока здесь тихо"}
          text={owner ? "Добавьте подарок вручную или вставьте ссылку на товар." : "Автор ещё не добавил желания в этот список."}
          action={owner ? <Link className="button button--secondary" to={`/lists/${list.id}/wishes/new`}>Добавить первое</Link> : undefined}
        />
      )}
      <div className="wish-grid">
        {list.wishes?.map((wish) => (
          <WishCard
            wish={wish}
            key={wish.id}
            owner={owner}
            canReserve={!owner && list.allowReservations}
            invalidate={publicView ? ["public-wishlist"] : ["wishlist"]}
          />
        ))}
      </div>
      {owner && list.publicToken && (
        <section className="share-card">
          <div className="share-card__icon"><Icon name="share" /></div>
          <div><h3>Список готов к встрече с друзьями</h3><p>Отправьте ссылку — открыть её можно прямо в Telegram.</p></div>
          <button className="button button--secondary button--small" onClick={share}>Поделиться</button>
        </section>
      )}
      <button className="sr-only" onClick={() => navigate("/lists")}>К спискам</button>
    </div>
  );
}

function shareList(list: Wishlist, config?: BrandConfig) {
  if (!list.publicToken || !config) return;
  const url = `https://t.me/${config.botUsername}?start=wishlist_${list.publicToken}`;
  shareTelegram(url, `Загляни в мой список «${list.title}»`);
}

function visibilityText(value: Wishlist["visibility"]) {
  return { public: "Публичный список", link: "Доступен по ссылке", private: "Личный список" }[value];
}
