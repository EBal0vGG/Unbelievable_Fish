import { Badge } from "@/shared/ui/badge";
import { Card } from "@/shared/ui/card";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import type { LotRecord } from "@/shared/types/domain";

function lotTone(status: LotRecord["status"]) {
  switch (status) {
    case "PUBLISHED":
      return "success";
    case "CLOSED":
      return "info";
    case "CANCELLED":
      return "danger";
    default:
      return "warning";
  }
}

export function LotCard({ lot }: { lot: LotRecord }) {
  return (
    <Card className="entity-card">
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Лот</p>
          <h3>{lot.productLabel}</h3>
        </div>
        <Badge tone={lotTone(lot.status)}>{lot.status}</Badge>
      </div>
      <div className="metric-grid">
        <div>
          <span>Старт</span>
          <strong>{formatMoney(lot.startPrice)}</strong>
        </div>
        <div>
          <span>Текущая</span>
          <strong>{formatMoney(lot.currentPrice ?? lot.finalPrice ?? lot.startPrice)}</strong>
        </div>
        <div>
          <span>Объем</span>
          <strong>{lot.quantity}</strong>
        </div>
        <div>
          <span>Старт торгов</span>
          <strong>{formatDateTime(lot.auctionStartsAt)}</strong>
        </div>
      </div>
      <p className="entity-card-meta">
        lot: {shortId(lot.id)} · product: {shortId(lot.productId)}
      </p>
      {lot.notes ? <p className="muted">{lot.notes}</p> : null}
    </Card>
  );
}
