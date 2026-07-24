import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { WishlistCard } from "../components/Cards";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, PageLoading } from "../components/States";
import { api } from "../lib/api";
import { Icon } from "../components/Icon";

export function ListsPage() {
  const query = useQuery({ queryKey: ["wishlists"], queryFn: api.wishlists });
  return (
    <div className="page">
      <PageHeader
        eyebrow="Личное пространство"
        title="Мои списки"
        action={<Link to="/lists/new" className="icon-button icon-button--accent"><Icon name="plus" /></Link>}
      />
      {query.isPending && <PageLoading />}
      {query.isError && <ErrorState message={query.error.message} retry={() => void query.refetch()} />}
      {query.data && query.data.items.length === 0 && query.data.saved.length === 0 && (
        <EmptyState
          visual={<Icon name="lists" size={34} />}
          title="Пора загадать первое желание"
          text="Создайте список для дня рождения, путешествия или просто хорошего дня."
          action={<Link className="button button--primary" to="/lists/new">Создать список</Link>}
        />
      )}
      <div className="wishlist-list">
        {query.data?.items.map((list) => <WishlistCard list={list} key={list.id} />)}
      </div>
      {query.data && query.data.saved.length > 0 && (
        <section className="saved-lists">
          <div className="section-title">
            <h2>Списки друзей</h2>
          </div>
          <div className="wishlist-list">
            {query.data.saved.map((list) => <WishlistCard list={list} key={list.id} />)}
          </div>
        </section>
      )}
    </div>
  );
}
