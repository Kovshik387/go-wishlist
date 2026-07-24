import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { Avatar } from "../components/Cards";
import { Icon } from "../components/Icon";
import { PageHeader } from "../components/PageHeader";
import { ErrorState, PageLoading } from "../components/States";
import { api } from "../lib/api";

export function UserPage() {
  const { id = "" } = useParams();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["user", id], queryFn: () => api.user(id) });
  const mutation = useMutation({
    mutationFn: () => query.data?.following ? api.unfollow(id) : api.follow(id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["user", id] });
      await queryClient.invalidateQueries({ queryKey: ["feed"] });
    },
  });
  if (query.isPending) return <div className="page"><PageLoading cards={1} /></div>;
  if (query.isError) return <div className="page"><ErrorState message={query.error.message} retry={() => void query.refetch()} /></div>;
  const { user, following } = query.data;
  return (
    <div className="page">
      <PageHeader back title="Профиль друга" />
      <section className="public-profile">
        <Avatar name={user.displayName} src={user.avatarUrl} size="large" />
        <h1>{user.displayName}</h1>
        <p>{user.username ? `@${user.username}` : "Любит хорошие сюрпризы"}</p>
        <button className={`button ${following ? "button--soft" : "button--primary"}`} onClick={() => mutation.mutate()} disabled={mutation.isPending}>
          <Icon name={following ? "check" : "heart"} size={18} /> {following ? "Вы подписаны" : "Подписаться"}
        </button>
      </section>
    </div>
  );
}
