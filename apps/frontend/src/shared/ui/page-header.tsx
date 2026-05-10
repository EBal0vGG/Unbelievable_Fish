import type { ReactNode } from "react";

export function PageHeader({
  eyebrow,
  title,
  description,
  actions,
  metrics,
  compact = false,
}: {
  eyebrow: string;
  title: string;
  description?: string;
  actions?: ReactNode;
  metrics?: ReactNode;
  compact?: boolean;
}) {
  return (
    <section className={`page-hero ${compact ? "compact-hero" : ""}`}>
      <div>
        <p className="eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        {description ? <p className="hero-copy">{description}</p> : null}
        {actions ? <div className="hero-actions">{actions}</div> : null}
      </div>
      {metrics ? <div className="hero-metrics">{metrics}</div> : null}
    </section>
  );
}
