import type { ReactNode } from "react";

import { cn } from "@/shared/lib/cn";

type BadgeTone = "neutral" | "info" | "success" | "warning" | "danger";

const toneClasses: Record<BadgeTone, string> = {
  neutral: "badge badge-neutral",
  info: "badge badge-info",
  success: "badge badge-success",
  warning: "badge badge-warning",
  danger: "badge badge-danger",
};

export function Badge({
  children,
  tone = "neutral",
}: {
  children: ReactNode;
  tone?: BadgeTone;
}) {
  return <span className={cn(toneClasses[tone])}>{children}</span>;
}
