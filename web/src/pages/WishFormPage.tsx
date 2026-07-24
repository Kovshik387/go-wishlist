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
  imageUrl: z.string(),
  price: z.string().refine((value) => !value || /^\d+([.,]\d{1,2})?$/.test(value), "Проверьте цену"),
  currency: z.enum(["RUB", "USD", "EUR"]),
  priority: z.enum(["normal", "high"]),
  quantity: z.number().int().min(1).max(99),
  size: z.string().max(80),
  color: z.string().max(80),
  variant: z.string().max(120),
});
type Values = z.infer<typeof schema>;

const defaults: Values = {
  productUrl: "",
  title: "",
  description: "",
  imageUrl: "",
  price: "",
  currency: "RUB",
  priority: "normal",
  quantity: 1,
  size: "",
  color: "",
  variant: "",
};

export function WishFormPage() {
  const { id = "", wishId } = useParams();
  const editing = Boolean(wishId);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<"url" | "manual">("url");
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
    form.reset({
      productUrl: query.data.productUrl,
      title: query.data.title,
      description: query.data.description,
      imageUrl: query.data.imageUrl,
      price: query.data.priceMinor === undefined ? "" : String(query.data.priceMinor / 100),
      currency: query.data.currency as Values["currency"],
      priority: query.data.priority,
      quantity: query.data.quantity,
      size: query.data.attributes.size ?? "",
      color: query.data.attributes.color ?? "",
      variant: query.data.attributes.variant ?? "",
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
        form.setValue("currency", preview.currency as Values["currency"]);
      }
    },
  });
  const uploadMutation = useMutation({
    mutationFn: api.upload,
    onSuccess: (media) => form.setValue("imageUrl", media.publicUrl),
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
        currency: values.currency,
        priority: values.priority,
        quantity: values.quantity,
        attributes: { size: values.size, color: values.color, variant: values.variant },
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
          <button type="button" onClick={() => fileRef.current?.click()}>
            {form.watch("imageUrl") ? <img src={form.watch("imageUrl")} alt="Фото желания" /> : <span>＋<small>Добавить фото</small></span>}
          </button>
          <input
            ref={fileRef}
            hidden
            type="file"
            accept="image/jpeg,image/png"
            onChange={(event) => event.target.files?.[0] && uploadMutation.mutate(event.target.files[0])}
          />
          <p>{uploadMutation.isPending ? "Обрабатываем фото…" : "JPEG или PNG, до 6 МБ"}</p>
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
        <div className="field-row field-row--price">
          <label className="field">
            <span>Цена</span>
            <input aria-label="Цена" inputMode="decimal" placeholder="0" {...form.register("price")} />
            {form.formState.errors.price && <small className="field-error">{form.formState.errors.price.message}</small>}
          </label>
          <label className="field field--currency">
            <span>Валюта</span>
            <select {...form.register("currency")}><option>RUB</option><option>USD</option><option>EUR</option></select>
          </label>
          <label className="field field--quantity">
            <span>Кол-во</span>
            <input type="number" min="1" max="99" {...form.register("quantity", { valueAsNumber: true })} />
          </label>
        </div>
        <fieldset className="segmented-field segmented-field--two">
          <legend>Насколько хочется</legend>
          <label><input type="radio" value="normal" {...form.register("priority")} /><span>Буду рад</span></label>
          <label><input type="radio" value="high" {...form.register("priority")} /><span>Высокий приоритет</span></label>
        </fieldset>
        <details className="optional-fields">
          <summary>Размер, цвет и вариант</summary>
          <div className="field-row">
            <label className="field"><span>Размер</span><input placeholder="M, 42…" {...form.register("size")} /></label>
            <label className="field"><span>Цвет</span><input placeholder="Синий…" {...form.register("color")} /></label>
          </div>
          <label className="field"><span>Вариант</span><input placeholder="Модель или комплектация" {...form.register("variant")} /></label>
        </details>
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
