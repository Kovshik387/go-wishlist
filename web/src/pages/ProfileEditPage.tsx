import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { z } from "zod";
import { useAuth } from "../auth";
import { Avatar } from "../components/Cards";
import { PageHeader } from "../components/PageHeader";
import { api } from "../lib/api";

const schema = z.object({
  displayName: z.string().trim().min(1, "Введите имя").max(60),
  timezone: z.string().min(1),
});
type Values = z.infer<typeof schema>;

export function ProfileEditPage() {
  const { user, setUser } = useAuth();
  const navigate = useNavigate();
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { displayName: user.displayName, timezone: user.timezone },
  });
  const mutation = useMutation({
    mutationFn: api.updateMe,
    onSuccess: (updated) => {
      setUser(updated);
      navigate("/profile");
    },
  });
  return (
    <div className="page page--form">
      <PageHeader back eyebrow="Как вас видят друзья" title="Профиль" />
      <form className="form-card profile-form" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
        <Avatar name={user.displayName} src={user.avatarUrl} size="large" />
        <p className="field-hint">Фото синхронизируется с Telegram при следующем входе.</p>
        <label className="field"><span>Отображаемое имя</span><input {...form.register("displayName")} /></label>
        <label className="field">
          <span>Часовой пояс</span>
          <select {...form.register("timezone")}>
            <option value="Europe/Moscow">Москва</option>
            <option value="Europe/Kaliningrad">Калининград</option>
            <option value="Asia/Yekaterinburg">Екатеринбург</option>
            <option value="Asia/Novosibirsk">Новосибирск</option>
            <option value="Asia/Vladivostok">Владивосток</option>
            <option value="UTC">UTC</option>
          </select>
        </label>
        {mutation.isError && <p className="form-error">{mutation.error.message}</p>}
        <button className="button button--primary button--wide" disabled={mutation.isPending}>
          {mutation.isPending ? "Сохраняем…" : "Сохранить"}
        </button>
      </form>
    </div>
  );
}
