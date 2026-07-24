import { useQuery } from "@tanstack/react-query";
import { WishCard } from "../components/Cards";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, PageLoading } from "../components/States";
import { api } from "../lib/api";
import { Icon } from "../components/Icon";

export function ReservationsPage() {
  const query = useQuery({ queryKey: ["reservations"], queryFn: api.reservations });
  return (
    <div className="page">
      <PageHeader eyebrow="Секретный раздел" title="Мои брони" />
      <p className="page-intro">Только вы видите этот список. Авторы желаний не узнают о сюрпризе.</p>
      {query.isPending && <PageLoading />}
      {query.isError && <ErrorState message={query.error.message} retry={() => void query.refetch()} />}
      {query.data?.items.length === 0 && (
        <EmptyState visual={<Icon name="lock" size={34} />} title="Пока ничего не занято" text="Откройте список друга и забронируйте подарок, который хочется вручить." />
      )}
      <div className="wish-grid">
        {query.data?.items.map((item) => (
          <WishCard wish={item.wish} key={item.id} invalidate={["reservations"]} />
        ))}
      </div>
    </div>
  );
}
