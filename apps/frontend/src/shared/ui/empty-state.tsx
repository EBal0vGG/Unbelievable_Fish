import Link from "next/link";

import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";

export function EmptyState({
  title,
  description,
  actionHref,
  actionLabel,
  framed = true,
  size = "default",
}: {
  title: string;
  description: string;
  actionHref?: string;
  actionLabel?: string;
  framed?: boolean;
  size?: "default" | "compact";
}) {
  const content = (
    <>
      <div aria-hidden="true" className="empty-state-icon">
        <span />
      </div>
      <h3>{title}</h3>
      <p>{description}</p>
      {actionHref && actionLabel ? (
        <Link className={buttonStyles({ variant: "secondary" })} href={actionHref}>
          {actionLabel}
        </Link>
      ) : null}
    </>
  );

  return framed ? (
    <Card className={`empty-state empty-state-${size}`}>{content}</Card>
  ) : (
    <div className={`empty-state empty-state-inline empty-state-${size}`}>{content}</div>
  );
}
