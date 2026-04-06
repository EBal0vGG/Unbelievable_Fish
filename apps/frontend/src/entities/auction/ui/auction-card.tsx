import Link from "next/link";

import { Badge } from "@/shared/ui/badge";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import type { AuctionRecord } from "@/shared/types/domain";

function auctionTone(state: AuctionRecord["state"]) {
  switch (state) {
    case "PUBLISHED":
      return "success";
    case "WON":
    case "CLOSED":
      return "info";
    case "CANCELLED":
      return "danger";
    default:
      return "warning";
  }
}

export function AuctionCard({
  auction,
  productLabel,
  fishName,
  sellerCompanyId,
}: {
  auction: AuctionRecord;
  productLabel?: string;
  fishName?: string;
  sellerCompanyId?: string;
}) {
  const canOpenDetails = !auction.id.startsWith("pending-");
  return (
    <Card className="entity-card">
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Аукцион</p>
          <h3>{productLabel ?? fishName ?? `Аукцион ${shortId(auction.id)}`}</h3>
          <p className="muted">Сессия #{shortId(auction.id)}</p>
        </div>
        <Badge tone={auctionTone(auction.state)}>{auction.state}</Badge>
      </div>
      <div className="metric-grid">
        <div>
          <span>Текущая ставка</span>
          <strong>{formatMoney(auction.currentPrice ?? auction.finalPrice)}</strong>
        </div>
        <div>
          <span>Лот</span>
          <strong>{shortId(auction.lotId)}</strong>
        </div>
        <div>
          <span>Рыба</span>
          <strong>{fishName ?? "н/д"}</strong>
        </div>
        <div>
          <span>Продавец</span>
          <strong>{sellerCompanyId ?? "н/д"}</strong>
        </div>
        <div>
          <span>Старт</span>
          <strong>{formatDateTime(auction.startsAt)}</strong>
        </div>
        <div>
          <span>Завершение</span>
          <strong>{formatDateTime(auction.endsAt)}</strong>
        </div>
      </div>
      <div className="inline-actions">
        {canOpenDetails ? (
          <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/auctions/${auction.id}`}>
            Открыть аукцион
          </Link>
        ) : (
          <span className="muted">Ожидается синхронизация read-model Trading</span>
        )}
      </div>
    </Card>
  );
}
