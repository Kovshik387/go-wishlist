import { useMutation, useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth";
import { Avatar } from "../components/Cards";
import { Icon } from "../components/Icon";
import { PageHeader } from "../components/PageHeader";
import { api } from "../lib/api";
import { telegram } from "../lib/telegram";

export function ProfilePage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const lists = useQuery({ queryKey: ["wishlists"], queryFn: api.wishlists });
  const logout = useMutation({
    mutationFn: api.logout,
    onSuccess: () => window.location.reload(),
  });
  const remove = useMutation({
    mutationFn: api.deleteMe,
    onSuccess: () => window.location.reload(),
  });
  const openBugReport = () => {
    const url = "https://t.me/yrulewet";
    if (telegram) telegram.openTelegramLink(url);
    else window.open(url, "_blank", "noopener");
  };
  return (
    <div className="page">
      <PageHeader
        eyebrow="Ваше пространство"
        title="Профиль"
        action={<Link className="icon-button" to="/settings"><Icon name="settings" /></Link>}
      />
      <section className="profile-card">
        <Avatar name={user.displayName} src={user.avatarUrl} size="large" />
        <div>
          <h2>{user.displayName}</h2>
          <p>{user.username ? `@${user.username}` : "Без username"}</p>
        </div>
        <button className="button button--soft button--small" onClick={() => navigate("/profile/edit")}>Изменить</button>
      </section>
      <div className="stats-row">
        <div><strong>{lists.data?.items.length ?? "—"}</strong><span>списков</span></div>
        <div><strong>{lists.data?.items.reduce((sum, item) => sum + item.wishCount, 0) ?? "—"}</strong><span>желаний</span></div>
        <div><strong>{user.botWriteAllowed ? "Да" : "Нет"}</strong><span>уведомления</span></div>
      </div>
      <section className="menu-card">
        <Link to="/settings"><span className="menu-card__icon"><Icon name="bell" /></span><span><strong>Уведомления</strong><small>События и тихие часы</small></span><Icon name="arrow" /></Link>
        <Link to="/lists"><span className="menu-card__icon"><Icon name="lock" /></span><span><strong>Приватность списков</strong><small>Настраивается для каждого списка</small></span><Icon name="arrow" /></Link>
        <button onClick={openBugReport}><span className="menu-card__icon"><Icon name="external" /></span><span><strong>Нашли баг?</strong><small>Напишите @yrulewet</small></span><Icon name="arrow" /></button>
      </section>
      <button className="button button--text button--wide" disabled={logout.isPending} onClick={() => logout.mutate()}>
        Выйти из аккаунта
      </button>
      <button
        className="button button--text button--danger-text button--wide"
        disabled={remove.isPending}
        onClick={() => window.confirm("Удалить аккаунт? Профиль будет анонимизирован, действие необратимо.") && remove.mutate()}
      >
        Удалить аккаунт
      </button>
    </div>
  );
}
