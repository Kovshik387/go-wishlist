import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";

export function PageHeader({
  title,
  eyebrow,
  back = false,
  backTo = "/lists",
  action,
}: {
  title: string;
  eyebrow?: string;
  back?: boolean;
  backTo?: string;
  action?: ReactNode;
}) {
  const navigate = useNavigate();
  const handleBack = () => {
    const historyIndex = window.history.state?.idx;
    if (typeof historyIndex === "number" && historyIndex > 0) {
      navigate(-1);
      return;
    }
    navigate(backTo, { replace: true });
  };
  return (
    <header className="page-header">
      <div className="page-header__leading">
        {back && (
          <button className="icon-button icon-button--back" onClick={handleBack} aria-label="Назад">
            ←
          </button>
        )}
        <div>
          {eyebrow && <p className="eyebrow">{eyebrow}</p>}
          <h1>{title}</h1>
        </div>
      </div>
      {action}
    </header>
  );
}
