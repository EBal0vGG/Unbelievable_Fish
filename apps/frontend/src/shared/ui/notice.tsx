import type { ReactNode } from "react";

import { cn } from "@/shared/lib/cn";

type NoticeTone = "info" | "warning" | "success";

export function Notice({
  title,
  children,
  tone = "info",
}: {
  title?: string;
  children: ReactNode;
  tone?: NoticeTone;
}) {
  return (
    <div className={cn("notice", `notice-${tone}`)}>
      {title ? <p className="notice-title">{title}</p> : null}
      <div className="notice-copy">{children}</div>
    </div>
  );
}
