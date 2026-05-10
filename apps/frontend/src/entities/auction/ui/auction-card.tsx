import Link from "next/link";

import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { EntityPhoto } from "@/shared/ui/entity-photo";
import { displayCompany, displayText } from "@/shared/lib/display";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { auctionStateLabels } from "@/shared/lib/labels";
import { withEffectiveAuctionState } from "@/shared/lib/trading-domain";
import { StatusBadge } from "@/shared/ui/status-badge";
import type { AuctionRecord } from "@/shared/types/domain";

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
  const effectiveAuction = withEffectiveAuctionState(auction);
  const isClosedByTime = auction.state === "PUBLISHED" && effectiveAuction.state !== "PUBLISHED";

  return (
    <Card className="entity-card auction-card">
      <EntityPhoto src={photo} alt={productLabel ?? fishName ?? `Аукцион ${shortId(auction.id)}`} />
      <div className="entity-card-header">
        <div>
          <p className="eyebrow">Аукцион</p>
          <h3>{productLabel ?? fishName ?? `Аукцион ${shortId(auction.id)}`}</h3>
          <p className="muted">Сессия #{shortId(auction.id)}</p>
        </div>
        <StatusBadge status={effectiveAuction.state} label={auctionStateLabels[effectiveAuction.state]} />
      </div>
      {isClosedByTime ? (
        <p className="muted">Время приема ставок истекло. Финальный статус синхронизируется с backend.</p>
      ) : null}
      <div className="metric-grid">
        <div>
          <span>{effectiveAuction.state === "PUBLISHED" ? "Текущая ставка" : "Финальная цена"}</span>
          <strong>{formatMoney(effectiveAuction.currentPrice ?? effectiveAuction.finalPrice)}</strong>
        </div>
        <div>
          <span>Лот</span>
          <strong>{shortId(effectiveAuction.lotId)}</strong>
        </div>
        <div>
          <span>Рыба</span>
          <strong>{displayText(fishName)}</strong>
        </div>
        <div>
          <span>Продавец</span>
          <strong title={sellerCompanyId}>{displayCompany(sellerCompanyId)}</strong>
        </div>
        <div>
          <span>Старт</span>
          <strong>{formatDateTime(effectiveAuction.startsAt)}</strong>
        </div>
        <div>
          <span>Завершение</span>
          <strong>{formatDateTime(effectiveAuction.endsAt)}</strong>
        </div>
      </div>
      <div className="entity-card-footer">
        <span>{effectiveAuction.state === "PUBLISHED" ? "Биржевой стакан" : "Итог торгов"}</span>
        <strong>{effectiveAuction.leaderCompanyId ?? effectiveAuction.winnerCompanyId ? displayCompany(effectiveAuction.leaderCompanyId ?? effectiveAuction.winnerCompanyId) : "Лидер не выбран"}</strong>
      </div>
      <div className="inline-actions">
        <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/auctions/${auction.id}`}>
          Открыть аукцион
        </Link>
      </div>
    </Card>
  );
}
