import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { useAuth } from "../auth";
import { Avatar, WishCard } from "../components/Cards";
import { EmptyState, ErrorState, PageLoading } from "../components/States";
import { api } from "../lib/api";
import { Icon } from "../components/Icon";

export function FeedPage() {
  const { user } = useAuth();
  const query = useQuery({ queryKey: ["feed"], queryFn: api.feed });
  return (
    <div className="page page--feed">
      <header className="feed-hero">
        <div>
          <p className="eyebrow">Добрый день</p>
          <h1>{user.displayName.split(" ")[0]}, что подарим?</h1>
        </div>
        <Link to="/profile" aria-label="Профиль"><Avatar name={user.displayName} src={user.avatarUrl} /></Link>
      </header>
      <section className="feed-banner">
        <div>
          <p className="eyebrow">Маленькая подсказка</p>
          <h2>Поделитесь первым списком</h2>
          <p>Друзьям будет проще выбрать подарок, а сюрприз останется секретом.</p>
        </div>
        <Link to="/lists" className="feed-banner__link">К спискам →</Link>
      </section>
      <div className="section-title">
        <div><p className="eyebrow">Обновления друзей</p><h2>Новое в ленте</h2></div>
      </div>
      {query.isPending && <PageLoading />}
      {query.isError && <ErrorState message={query.error.message} retry={() => void query.refetch()} />}
      {query.data?.items.length === 0 && (
        <EmptyState
          visual={<Icon name="feed" size={34} />}
          title="Здесь будут желания друзей"
          text="Откройте их список по ссылке и подпишитесь, чтобы видеть обновления."
        />
      )}
      <div className="wish-grid">
        {query.data?.items.map((wish) => (
          <WishCard wish={wish} key={wish.id} invalidate={["feed"]} />
        ))}
      </div>
    </div>
  );
}
