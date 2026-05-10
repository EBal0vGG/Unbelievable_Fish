import { Badge } from "@/shared/ui/badge";
import type { AuctionState, LotStatus, ProductStatus } from "@/shared/types/domain";

type Status = ProductStatus | LotStatus | AuctionState;
type Tone = "neutral" | "info" | "success" | "warning" | "danger";

export function statusTone(status: Status): Tone {
  switch (status) {
    case "PUBLISHED":
      return "success";
    case "CLOSED":
    case "WON":
      return "info";
    case "CANCELLED":
      return "danger";
    case "DRAFT":
    default:
      return "warning";
  }
}

export function StatusBadge({ status, label }: { status: Status; label: string }) {
  return <Badge tone={statusTone(status)}>{label}</Badge>;
}
