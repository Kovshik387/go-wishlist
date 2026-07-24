import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { Icon } from "./Icon";

const navigation = [
  { to: "/", label: "Лента", icon: "feed" as const },
  { to: "/lists", label: "Мои списки", icon: "lists" as const },
  { to: "/reservations", label: "Брони", icon: "gift" as const },
  { to: "/profile", label: "Профиль", icon: "profile" as const },
];

export function Shell() {
  const navigate = useNavigate();
  return (
    <div className="app-shell">
      <main className="app-main">
        <Outlet />
      </main>
      <nav className="bottom-nav" aria-label="Основная навигация">
        {navigation.slice(0, 2).map((item) => <NavItem {...item} key={item.to} />)}
        <button
          className="bottom-nav__add"
          aria-label="Создать список"
          onClick={() => navigate("/lists/new")}
        >
          <Icon name="plus" size={24} />
        </button>
        {navigation.slice(2).map((item) => <NavItem {...item} key={item.to} />)}
      </nav>
    </div>
  );
}

function NavItem({ to, label, icon }: (typeof navigation)[number]) {
  return (
    <NavLink
      to={to}
      end={to === "/"}
      className={({ isActive }) => `bottom-nav__item ${isActive ? "is-active" : ""}`}
    >
      <Icon name={icon} />
      <span>{label}</span>
    </NavLink>
  );
}
