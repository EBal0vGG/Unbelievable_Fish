import { Card } from "@/shared/ui/card";
import { EntityPhoto } from "@/shared/ui/entity-photo";
import { displayCompany } from "@/shared/lib/display";
import { formatDateTime, formatMoney } from "@/shared/lib/format";
import { lotStatusLabels } from "@/shared/lib/labels";
import { StatusBadge } from "@/shared/ui/status-badge";
import type { LotRecord } from "@/shared/types/domain";

export function LotCard({ lot }: { lot: LotRecord }) {
  return (
    <Card className="entity-card lot-card">
      <EntityPhoto src={lot.photo} alt={lot.productLabel} />
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Лот</p>
          <h3>{lot.productLabel}</h3>
        </div>
        <StatusBadge status={lot.status} label={lotStatusLabels[lot.status]} />
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
          <span>Компания</span>
          <strong title={lot.sellerCompanyId}>{displayCompany(lot.sellerCompanyId)}</strong>
        </div>
        <div>
          <span>Старт торгов</span>
          <strong>{formatDateTime(lot.auctionStartsAt)}</strong>
        </div>
      </div>
      <div className="entity-card-footer">
        <span>{lot.auctionId ? "Связан с аукционом" : "Аукцион не создан"}</span>
        <strong>{lot.auctionId ?? "ожидает публикации"}</strong>
      </div>
    </Card>
  );
}
