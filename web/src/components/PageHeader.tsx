import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";

export function PageHeader({
  title,
  eyebrow,
  back = false,
  action,
}: {
  title: string;
  eyebrow?: string;
  back?: boolean;
  action?: ReactNode;
}) {
  const navigate = useNavigate();
  return (
    <header className="page-header">
      <div className="page-header__leading">
        {back && (
          <button className="icon-button icon-button--back" onClick={() => navigate(-1)} aria-label="Назад">
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
