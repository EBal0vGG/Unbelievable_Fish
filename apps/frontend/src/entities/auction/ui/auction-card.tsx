import Link from "next/link";

import { Badge } from "@/shared/ui/badge";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { EntityPhoto } from "@/shared/ui/entity-photo";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { auctionStateLabels } from "@/shared/lib/labels";
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
  photo,
}: {
  auction: AuctionRecord;
  productLabel?: string;
  fishName?: string;
  sellerCompanyId?: string;
  photo?: string;
}) {
  return (
    <Card className="entity-card auction-card">
      <EntityPhoto src={photo} alt={productLabel ?? fishName ?? `Аукцион ${shortId(auction.id)}`} />
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Аукцион</p>
          <h3>{productLabel ?? fishName ?? `Аукцион ${shortId(auction.id)}`}</h3>
          <p className="muted">Сессия #{shortId(auction.id)}</p>
        </div>
        <Badge tone={auctionTone(auction.state)}>{auctionStateLabels[auction.state]}</Badge>
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
        <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/auctions/${auction.id}`}>
          Открыть аукцион
        </Link>
      </div>
    </Card>
  );
}
