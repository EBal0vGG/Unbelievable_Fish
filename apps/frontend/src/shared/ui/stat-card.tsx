import type { ReactNode } from "react";

import { Card } from "@/shared/ui/card";

export function StatCard({
  label,
  value,
  detail,
  tone = "default",
}: {
  label: string;
  value: ReactNode;
  detail?: ReactNode;
  tone?: "default" | "primary" | "accent";
}) {
  return (
    <Card className={`stat-card stat-card-${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      {detail ? <small>{detail}</small> : null}
    </Card>
  );
}
