import Link from "next/link";

import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";

export function EmptyState({
  title,
  description,
  actionHref,
  actionLabel,
}: {
  title: string;
  description: string;
  actionHref?: string;
  actionLabel?: string;
}) {
  return (
    <Card className="empty-state">
      <h3>{title}</h3>
      <p>{description}</p>
      {actionHref && actionLabel ? (
        <Link className={buttonStyles({ variant: "secondary" })} href={actionHref}>
          {actionLabel}
        </Link>
      ) : null}
    </Card>
  );
}
