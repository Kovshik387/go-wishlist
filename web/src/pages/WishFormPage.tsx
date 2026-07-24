import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router-dom";
import { z } from "zod";
import { PageHeader } from "../components/PageHeader";
import { PageLoading } from "../components/States";
import { api, APIError } from "../lib/api";
import { haptic } from "../lib/telegram";

const schema = z.object({
  productUrl: z.string().refine((value) => !value || /^https?:\/\//.test(value), "Нужна ссылка с http:// или https://"),
  title: z.string().trim().min(1, "Напишите название").max(160),
  description: z.string().max(2000),
  imageUrl: z.string().refine(
    (value) => !value || value.startsWith("/") || /^https?:\/\//.test(value),
    "Нужна ссылка с http:// или https://",
  ),
  price: z.string().refine((value) => !value || /^\d+([.,]\d{1,2})?$/.test(value), "Проверьте цену"),
});
type Values = z.infer<typeof schema>;

const defaults: Values = {
  productUrl: "",
  title: "",
  description: "",
  imageUrl: "",
  price: "",
};

export function WishFormPage() {
  const { id = "", wishId } = useParams();
  const editing = Boolean(wishId);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<"url" | "manual">("url");
  const [currency, setCurrency] = useState("RUB");
  const fileRef = useRef<HTMLInputElement>(null);
  const query = useQuery({
    queryKey: ["wish", wishId],
    queryFn: () => api.wish(id, wishId!),
    enabled: editing,
  });
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues: defaults });
  useEffect(() => {
    if (!query.data) return;
    setMode(query.data.productUrl ? "url" : "manual");
    setCurrency(query.data.currency || "RUB");
    form.reset({
      productUrl: query.data.productUrl,
      title: query.data.title,
      description: query.data.description,
      imageUrl: query.data.imageUrl,
      price: query.data.priceMinor === undefined ? "" : String(query.data.priceMinor / 100),
    });
  }, [query.data, form]);
  const previewMutation = useMutation({
    mutationFn: (url: string) => api.preview(url),
    onSuccess: (preview) => {
      if (preview.title) form.setValue("title", preview.title);
      if (preview.description) form.setValue("description", preview.description);
      if (preview.imageUrl) form.setValue("imageUrl", preview.imageUrl);
      if (preview.priceMinor !== undefined) form.setValue("price", String(preview.priceMinor / 100));
      if (preview.currency && ["RUB", "USD", "EUR"].includes(preview.currency)) {
        setCurrency(preview.currency);
      }
    },
  });
  const uploadMutation = useMutation({
    mutationFn: api.upload,
    onSuccess: (media) => form.setValue("imageUrl", media.publicUrl, { shouldValidate: true }),
  });
  const saveMutation = useMutation({
    mutationFn: (values: Values) => {
      const priceMinor = values.price
        ? Math.round(Number(values.price.replace(",", ".")) * 100)
        : undefined;
      const body = {
        productUrl: mode === "url" ? values.productUrl : "",
        title: values.title,
        description: values.description,
        imageUrl: values.imageUrl,
        priceMinor,
        currency,
        priority: query.data?.priority ?? "normal",
        quantity: query.data?.quantity ?? 1,
        attributes: query.data?.attributes ?? {},
        version: query.data?.version,
      };
      return editing ? api.updateWish(id, wishId!, body) : api.createWish(id, body);
    },
    onSuccess: async () => {
      haptic();
      await queryClient.invalidateQueries({ queryKey: ["wishlist", id] });
      await queryClient.invalidateQueries({ queryKey: ["wishlists"] });
      navigate(`/lists/${id}`, { replace: true });
    },
  });
  const deleteMutation = useMutation({
    mutationFn: () => api.deleteWish(id, wishId!),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["wishlist", id] });
      navigate(`/lists/${id}`, { replace: true });
    },
  });
  if (editing && query.isPending) return <div className="page"><PageLoading cards={1} /></div>;
  const imageUrl = form.watch("imageUrl");
  const imagePreview = imageUrl.startsWith("/") || /^https?:\/\//.test(imageUrl) ? imageUrl : "";
  return (
    <div className="page page--form">
      <PageHeader back eyebrow={editing ? "Точная настройка" : "Ещё одна мечта"} title={editing ? "Изменить желание" : "Добавить желание"} />
      <form className="form-card" onSubmit={form.handleSubmit((values) => saveMutation.mutate(values))}>
        {!editing && (
          <div className="mode-switch">
            <button type="button" className={mode === "url" ? "is-active" : ""} onClick={() => setMode("url")}>По ссылке</button>
            <button type="button" className={mode === "manual" ? "is-active" : ""} onClick={() => setMode("manual")}>Вручную</button>
          </div>
        )}
        {mode === "url" && (
          <div className="url-import">
            <label className="field">
              <span>Ссылка на товар</span>
              <div className="field-action">
                <input aria-label="Ссылка на товар" placeholder="https://магазин.ru/товар" {...form.register("productUrl")} />
                <button
                  type="button"
                  className="button button--soft button--small"
                  disabled={!form.watch("productUrl") || previewMutation.isPending}
                  onClick={() => previewMutation.mutate(form.getValues("productUrl"))}
                >
                  {previewMutation.isPending ? "Ищем…" : "Заполнить"}
                </button>
              </div>
              {form.formState.errors.productUrl && <small className="field-error">{form.formState.errors.productUrl.message}</small>}
              {previewMutation.isError && <small className="field-hint">Не получилось прочитать страницу — поля можно заполнить вручную.</small>}
            </label>
          </div>
        )}
        <div className="image-uploader">
          <button type="button" aria-label="Загрузить фото" onClick={() => fileRef.current?.click()}>
            {imagePreview ? <img src={imagePreview} alt="" /> : <span>＋<small>Загрузить фото</small></span>}
          </button>
          <input
            ref={fileRef}
            hidden
            type="file"
            accept="image/jpeg,image/png"
            onChange={(event) => event.target.files?.[0] && uploadMutation.mutate(event.target.files[0])}
          />
          <div className="image-uploader__controls">
            <p>{uploadMutation.isPending ? "Обрабатываем фото…" : "JPEG или PNG, до 6 МБ"}</p>
            <label className="field">
              <span>Или вставьте ссылку на фото</span>
              <input
                aria-label="Ссылка на фото"
                inputMode="url"
                placeholder="https://site.ru/photo.jpg"
                {...form.register("imageUrl")}
              />
              {form.formState.errors.imageUrl && <small className="field-error">{form.formState.errors.imageUrl.message}</small>}
            </label>
            {uploadMutation.isError && <small className="field-error">Не удалось загрузить фото</small>}
          </div>
        </div>
        <label className="field">
          <span>Название</span>
          <input aria-label="Название" placeholder="Что хочется получить?" {...form.register("title")} />
          {form.formState.errors.title && <small className="field-error">{form.formState.errors.title.message}</small>}
        </label>
        <label className="field">
          <span>Комментарий <small>необязательно</small></span>
          <textarea aria-label="Комментарий" placeholder="Почему именно этот вариант, важные детали…" rows={3} {...form.register("description")} />
        </label>
        <label className="field">
          <span>Цена <small>необязательно</small></span>
          <input aria-label="Цена" inputMode="decimal" placeholder="0" {...form.register("price")} />
          {form.formState.errors.price && <small className="field-error">{form.formState.errors.price.message}</small>}
        </label>
        {saveMutation.isError && <p className="form-error">{saveMutation.error instanceof APIError ? saveMutation.error.message : "Не удалось сохранить желание"}</p>}
        <button className="button button--primary button--wide" disabled={saveMutation.isPending}>
          {saveMutation.isPending ? "Сохраняем…" : editing ? "Сохранить изменения" : "Добавить в список"}
        </button>
        {editing && (
          <button
            type="button"
            className="button button--danger button--wide"
            disabled={deleteMutation.isPending}
            onClick={() => window.confirm("Удалить это желание?") && deleteMutation.mutate()}
          >
            Удалить желание
          </button>
        )}
      </form>
    </div>
  );
}
