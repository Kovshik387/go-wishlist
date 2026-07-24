import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { Avatar, WishlistCard } from "../components/Cards";
import { Icon } from "../components/Icon";
import { PageHeader } from "../components/PageHeader";
import { ErrorState, PageLoading } from "../components/States";
import { api } from "../lib/api";
import { haptic } from "../lib/telegram";

export function UserPage() {
  const { id = "" } = useParams();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["user", id], queryFn: () => api.user(id) });
  const mutation = useMutation({
    mutationFn: async () => {
      const nextFollowing = !query.data?.following;
      if (nextFollowing) await api.follow(id);
      else await api.unfollow(id);
      return nextFollowing;
    },
    onSuccess: async (nextFollowing) => {
      haptic("success");
      queryClient.setQueryData(["user", id], (current: typeof query.data) =>
        current ? { ...current, following: nextFollowing } : current);
      await queryClient.invalidateQueries({ queryKey: ["feed"] });
    },
  });
  if (query.isPending) return <div className="page"><PageLoading cards={1} /></div>;
  if (query.isError) return <div className="page"><ErrorState message={query.error.message} retry={() => void query.refetch()} /></div>;
  const { user, following, wishlists } = query.data;
  return (
    <div className="page">
      <PageHeader back title="Профиль друга" />
      <section className="public-profile">
        <Avatar name={user.displayName} src={user.avatarUrl} size="large" />
        <h1>{user.displayName}</h1>
        <p>{user.username ? `@${user.username}` : "Любит хорошие сюрпризы"}</p>
        <button className={`button ${following ? "button--soft" : "button--primary"}`} onClick={() => mutation.mutate()} disabled={mutation.isPending}>
          <Icon name={following ? "check" : "heart"} size={18} />
          {mutation.isPending ? "Секунду…" : following ? "Вы подписаны" : "Подписаться"}
        </button>
        {mutation.isError && <p className="form-error" role="alert">{mutation.error.message}</p>}
      </section>
      <section className="friend-lists">
        <div className="section-title">
          <h2>Списки {user.displayName}</h2>
        </div>
        {wishlists.length > 0 ? (
          <div className="wishlist-list">
            {wishlists.map((list) => <WishlistCard list={list} key={list.id} />)}
          </div>
        ) : (
          <p className="friend-lists__empty">У друга пока нет открытых списков.</p>
        )}
      </section>
    </div>
  );
}
