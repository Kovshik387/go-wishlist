import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { ErrorState, PageLoading } from "../components/States";
import { Toggle } from "./WishlistFormPage";
import { api } from "../lib/api";
import type { NotificationPreferences } from "../types";

export function SettingsPage() {
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["notification-settings"], queryFn: api.notificationSettings });
  const [prefs, setPrefs] = useState<NotificationPreferences | null>(null);
  useEffect(() => {
    if (query.data) setPrefs(query.data);
  }, [query.data]);
  const mutation = useMutation({
    mutationFn: api.updateNotificationSettings,
    onSuccess: (updated) => {
      setPrefs(updated);
      queryClient.setQueryData(["notification-settings"], updated);
    },
  });
  const update = <K extends keyof NotificationPreferences>(key: K, value: NotificationPreferences[K]) => {
    if (!prefs) return;
    const next = { ...prefs, [key]: value };
    setPrefs(next);
    mutation.mutate(next);
  };
  if (query.isPending || !prefs) return <div className="page"><PageLoading cards={2} /></div>;
  if (query.isError) return <div className="page"><ErrorState message={query.error.message} retry={() => void query.refetch()} /></div>;
  return (
    <div className="page page--form">
      <PageHeader back eyebrow="Только полезное" title="Уведомления" />
      <section className="settings-panel">
        <Toggle title="Все уведомления" text="Главный выключатель" checked={prefs.enabled} onChange={(value) => update("enabled", value)} />
      </section>
      <section className={`settings-panel ${!prefs.enabled ? "is-disabled" : ""}`}>
        <h2>Что сообщать</h2>
        <Toggle title="Новые желания" text="От людей, на которых вы подписаны" checked={prefs.newWishes} onChange={(value) => update("newWishes", value)} />
        <Toggle title="Новые списки" checked={prefs.newWishlists} onChange={(value) => update("newWishlists", value)} />
        <Toggle title="Скоро событие" text="Напомним о приближающейся дате" checked={prefs.eventReminders} onChange={(value) => update("eventReminders", value)} />
      </section>
      <section className={`settings-panel ${!prefs.enabled ? "is-disabled" : ""}`}>
        <h2>Тихие часы</h2>
        <Toggle title="Не беспокоить ночью" checked={prefs.quietHoursEnabled} onChange={(value) => update("quietHoursEnabled", value)} />
        {prefs.quietHoursEnabled && (
          <div className="field-row quiet-fields">
            <label className="field"><span>С</span><input type="time" value={prefs.quietStart} onChange={(event) => update("quietStart", event.target.value)} /></label>
            <label className="field"><span>До</span><input type="time" value={prefs.quietEnd} onChange={(event) => update("quietEnd", event.target.value)} /></label>
          </div>
        )}
      </section>
      <p className="settings-footnote">Изменения сохраняются сразу. Отдельного автора можно заглушить, не отписываясь от него.</p>
      {mutation.isError && <p className="form-error">{mutation.error.message}</p>}
    </div>
  );
}
