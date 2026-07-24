import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { api } from "../lib/api";
import { telegram } from "../lib/telegram";
import { useAuth } from "../auth";
import { Icon, type IconName } from "./Icon";

const slides: Array<{ icon: IconName; eyebrow: string; title: string; text: string }> = [
  {
    icon: "lists",
    eyebrow: "Всё важное рядом",
    title: "Желания, которые легко исполнить",
    text: "Собирайте подарки из любых магазинов, добавляйте детали и делитесь одним касанием.",
  },
  {
    icon: "lock",
    eyebrow: "Сюрприз останется сюрпризом",
    title: "Друзья договорятся без вас",
    text: "Они увидят занятые подарки, а вы — нет. Никаких дублей и случайных подсказок.",
  },
  {
    icon: "bell",
    eyebrow: "Ничего не потеряется",
    title: "Позовём друзей вовремя",
    text: "Бот мягко сообщит подписчикам о новых желаниях. Уведомления всегда можно настроить.",
  },
];

export function Onboarding() {
  const { user, setUser } = useAuth();
  const [step, setStep] = useState(0);
  const [name, setName] = useState(user.displayName);
  const mutation = useMutation({
    mutationFn: async () => {
      if (telegram) {
        const webApp = telegram;
        await new Promise<void>((resolve) => {
          webApp.requestWriteAccess(() => resolve());
          window.setTimeout(resolve, 4000);
        });
      }
      return api.updateMe({
        displayName: name.trim() || user.displayName,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "Europe/Moscow",
        onboardingCompleted: true,
      });
    },
    onSuccess: setUser,
  });
  const slide = slides[step];
  return (
    <div className="onboarding" role="dialog" aria-modal="true">
      <section className="onboarding__card">
        <div className="onboarding__progress" aria-label={`Шаг ${step + 1} из ${slides.length}`}>
          {slides.map((_, index) => (
            <span key={index} className={index <= step ? "is-active" : ""} />
          ))}
        </div>
        <div className="onboarding__icon" aria-hidden="true"><Icon name={slide.icon} size={46} /></div>
        <p className="eyebrow">{slide.eyebrow}</p>
        <h1>{slide.title}</h1>
        <p className="onboarding__text">{slide.text}</p>
        {step === slides.length - 1 && (
          <label className="field">
            <span>Как вас называть?</span>
            <input value={name} onChange={(event) => setName(event.target.value)} maxLength={60} />
          </label>
        )}
        <button
          className="button button--primary button--wide"
          disabled={mutation.isPending}
          onClick={() => step < slides.length - 1 ? setStep(step + 1) : mutation.mutate()}
        >
          {mutation.isPending ? "Сохраняем…" : step < slides.length - 1 ? "Дальше" : "Начать"}
        </button>
        {step > 0 && (
          <button className="button button--text" onClick={() => setStep(step - 1)}>Назад</button>
        )}
        {mutation.isError && <p className="form-error">{mutation.error.message}</p>}
      </section>
    </div>
  );
}
