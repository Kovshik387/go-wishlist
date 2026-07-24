import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";
import { z } from "zod";
import { Icon } from "../components/Icon";
import { PageHeader } from "../components/PageHeader";
import { PageLoading } from "../components/States";
import { api, APIError } from "../lib/api";
import { haptic } from "../lib/telegram";

const schema = z.object({
  title: z.string().trim().min(1, "Напишите название").max(80, "Не больше 80 символов"),
  description: z.string().max(500, "Не больше 500 символов"),
  emoji: z.string().min(1, "Выберите emoji").max(8),
  coverUrl: z.string(),
  occasion: z.enum(["birthday", "wedding", "new_year", "housewarming", "other"]),
  eventDate: z.string(),
  visibility: z.enum(["public", "link", "private"]),
  allowReservations: z.boolean(),
  ownerSeesReservations: z.boolean(),
});
type Values = z.infer<typeof schema>;

const defaults: Values = {
  title: "",
  description: "",
  emoji: "🎁",
  coverUrl: "",
  occasion: "birthday",
  eventDate: "",
  visibility: "public",
  allowReservations: true,
  ownerSeesReservations: false,
};

export function WishlistFormPage() {
  const { id } = useParams();
  const editing = Boolean(id);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);
  const query = useQuery({
    queryKey: ["wishlist", id],
    queryFn: () => api.wishlist(id!),
    enabled: editing,
  });
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: defaults });
  useEffect(() => {
    if (query.data) {
      form.reset({
        title: query.data.title,
        description: query.data.description,
        emoji: query.data.emoji,
        coverUrl: query.data.coverUrl,
        occasion: query.data.occasion,
        eventDate: query.data.eventDate?.slice(0, 10) ?? "",
        visibility: query.data.visibility,
        allowReservations: query.data.allowReservations,
        ownerSeesReservations: query.data.ownerSeesReservations,
      });
    }
  }, [query.data, form]);
  const mutation = useMutation({
    mutationFn: (values: Values) => editing
      ? api.updateWishlist(id!, { ...values, version: query.data!.version })
      : api.createWishlist(values),
    onSuccess: async (list) => {
      haptic();
      await queryClient.invalidateQueries({ queryKey: ["wishlists"] });
      navigate(`/lists/${list.id}`, { replace: true });
    },
  });
  const uploadMutation = useMutation({
    mutationFn: api.upload,
    onSuccess: (media) => form.setValue("coverUrl", media.publicUrl, { shouldDirty: true, shouldValidate: true }),
  });
  const deleteMutation = useMutation({
    mutationFn: () => api.deleteWishlist(id!),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["wishlists"] });
      navigate("/lists", { replace: true });
    },
  });
  if (editing && query.isPending) return <div className="page"><PageLoading cards={1} /></div>;
  const coverUrl = form.watch("coverUrl");
  const coverPreview = coverUrl.startsWith("/") || /^https?:\/\//.test(coverUrl) ? coverUrl : "";
  return (
    <div className="page page--form">
      <PageHeader
        back
        eyebrow={editing ? "Настройки списка" : "Новая история"}
        title={editing ? "Редактировать" : "Создать список"}
      />
      <form className="form-card" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
        <input type="hidden" {...form.register("emoji")} />
        <label className="field">
          <span>Название</span>
          <input aria-label="Название" placeholder="Например, День рождения" autoFocus={!editing} {...form.register("title")} />
          <FieldError message={form.formState.errors.title?.message} />
        </label>
        <label className="field">
          <span>Коротко о списке <small>необязательно</small></span>
          <textarea aria-label="Коротко о списке" placeholder="Подсказка для друзей" rows={3} {...form.register("description")} />
          <FieldError message={form.formState.errors.description?.message} />
        </label>
        <div className="image-uploader image-uploader--cover">
          <button
            type="button"
            aria-label={coverPreview ? "Заменить фотографию списка" : "Добавить фотографию списка"}
            disabled={uploadMutation.isPending}
            onClick={() => fileRef.current?.click()}
          >
            {coverPreview
              ? <img src={coverPreview} alt="Обложка списка" referrerPolicy="no-referrer" />
              : <span><Icon name="plus" size={24} /><small>Фото списка</small></span>}
          </button>
          <input
            ref={fileRef}
            hidden
            type="file"
            aria-label="Фотография списка"
            accept="image/*"
            onChange={(event) => event.target.files?.[0] && uploadMutation.mutate(event.target.files[0])}
          />
          <div className="image-uploader__controls">
            <p>{uploadMutation.isPending ? "Загружаем фотографию…" : "JPEG, PNG, WebP, GIF или AVIF, до 6 МБ"}</p>
            {coverPreview && (
              <button
                type="button"
                className="button button--soft button--small"
                onClick={() => form.setValue("coverUrl", "", { shouldDirty: true })}
              >
                Убрать фотографию
              </button>
            )}
            {uploadMutation.isError && (
              <p className="form-error" role="alert">
                {uploadMutation.error instanceof APIError ? uploadMutation.error.message : "Не удалось загрузить фотографию"}
              </p>
            )}
          </div>
        </div>
        <div className="field-row">
          <label className="field">
            <span>Повод</span>
            <select {...form.register("occasion")}>
              <option value="birthday">День рождения</option>
              <option value="wedding">Свадьба</option>
              <option value="new_year">Новый год</option>
              <option value="housewarming">Новоселье</option>
              <option value="other">Другое</option>
            </select>
          </label>
          <label className="field">
            <span>Дата <small>необязательно</small></span>
            <input type="date" {...form.register("eventDate")} />
          </label>
        </div>
        <fieldset className="segmented-field">
          <legend>Кто увидит список</legend>
          <label><input type="radio" value="public" {...form.register("visibility")} /><span>Все</span></label>
          <label><input type="radio" value="link" {...form.register("visibility")} /><span>По ссылке</span></label>
          <label><input type="radio" value="private" {...form.register("visibility")} /><span>Только я</span></label>
        </fieldset>
        <div className="settings-group">
          <Toggle
            title="Разрешить бронирование"
            text="Друзья смогут договориться без дублей"
            checked={form.watch("allowReservations")}
            onChange={(value) => form.setValue("allowReservations", value)}
          />
          <Toggle
            title="Показывать мне брони"
            text="По умолчанию скрыто, чтобы сохранить сюрприз"
            checked={form.watch("ownerSeesReservations")}
            onChange={(value) => form.setValue("ownerSeesReservations", value)}
          />
        </div>
        {mutation.isError && (
          <p className="form-error">{mutation.error instanceof APIError ? mutation.error.message : "Не удалось сохранить список"}</p>
        )}
        <button className="button button--primary button--wide" disabled={mutation.isPending || uploadMutation.isPending}>
          {mutation.isPending ? "Сохраняем…" : editing ? "Сохранить изменения" : "Создать список"}
        </button>
        {editing && (
          <button
            type="button"
            className="button button--danger button--wide"
            disabled={deleteMutation.isPending}
            onClick={() => window.confirm("Удалить список и все желания?") && deleteMutation.mutate()}
          >
            Удалить список
          </button>
        )}
      </form>
    </div>
  );
}

export function Toggle({
  title,
  text,
  checked,
  onChange,
}: {
  title: string;
  text?: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <label className="toggle-row">
      <span><strong>{title}</strong>{text && <small>{text}</small>}</span>
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      <i aria-hidden="true" />
    </label>
  );
}

function FieldError({ message }: { message?: string }) {
  return message ? <small className="field-error">{message}</small> : null;
}
