import Link from "next/link";

import { Badge } from "@/shared/ui/badge";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { displayCompany } from "@/shared/lib/display";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import type { DealRecord, DealStatus } from "@/shared/types/domain";

const statusLabels: Record<DealStatus, string> = {
  pending: "Ожидает",
  confirmed: "Подтверждена",
  contract_prepared: "Контракт готов",
  contract_signed: "Контракт подписан",
  payment_requested: "Ожидает оплату",
  paid: "Оплачена",
  shipment_requested: "К отгрузке",
  shipped: "Отгружена",
  completed: "Завершена",
  cancelled: "Отменена",
};

function dealTone(status: DealRecord["status"]) {
  switch (status) {
    case "completed":
    case "paid":
      return "success";
    case "cancelled":
      return "danger";
    case "pending":
    case "payment_requested":
      return "warning";
    default:
      return "info";
  }
}

function dealSideLabel(deal: DealRecord, viewerCompanyId?: string): string {
  if (viewerCompanyId === deal.supplierId) {
    return "Продажа";
  }
  if (viewerCompanyId === deal.customerId) {
    return "Покупка";
  }
  return "Сделка";
}

export function DealCard({ deal, viewerCompanyId }: { deal: DealRecord; viewerCompanyId?: string }) {
  return (
    <Card className="entity-card deal-card">
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">{dealSideLabel(deal, viewerCompanyId)}</p>
          <h3>{deal.productSnapshot.name || `Сделка ${shortId(deal.id)}`}</h3>
          <p className="muted">#{shortId(deal.id)} · аукцион {shortId(deal.auctionId)}</p>
        </div>
        <Badge tone={dealTone(deal.status)}>{statusLabels[deal.status]}</Badge>
      </div>

      <div className="deal-flow">
        <span className="deal-party" title={deal.supplierId}>{displayCompany(deal.supplierId)}</span>
        <span className="deal-arrow">→</span>
        <span className="deal-party" title={deal.customerId}>{displayCompany(deal.customerId)}</span>
      </div>

      <div className="metric-grid">
        <div>
          <span>Сумма</span>
          <strong>{formatMoney(deal.totalAmount)}</strong>
        </div>
        <div>
          <span>Цена</span>
          <strong>{formatMoney(deal.unitPrice)}</strong>
        </div>
        <div>
          <span>Объем</span>
          <strong>
            {deal.quantity} {deal.productSnapshot.unit || "ед."}
          </strong>
        </div>
        <div>
          <span>Создана</span>
          <strong>{formatDateTime(deal.createdAt)}</strong>
        </div>
      </div>

      <div className="inline-actions">
        <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/deals/${deal.id}`}>
          Открыть сделку
        </Link>
      </div>
    </Card>
  );
}

export { statusLabels as dealStatusLabels };
