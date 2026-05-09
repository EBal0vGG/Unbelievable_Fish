import type { ReactNode } from "react";

export function SectionCard({
  eyebrow,
  title,
  description,
  actions,
  children,
  className = "",
}: {
  eyebrow?: string;
  title?: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={`section-card ${className}`}>
      {title || eyebrow || description || actions ? (
        <div className="section-heading">
          <div>
            {eyebrow ? <p className="eyebrow">{eyebrow}</p> : null}
            {title ? <h2>{title}</h2> : null}
            {description ? <p className="muted">{description}</p> : null}
          </div>
          {actions ? <div className="inline-actions">{actions}</div> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}
