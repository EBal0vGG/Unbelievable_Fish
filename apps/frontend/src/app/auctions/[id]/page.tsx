"use client";

import { useParams } from "next/navigation";

import { useAuctionDetailsQuery } from "@/entities/auction/model/hooks";
import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { PlaceBidForm } from "@/features/auction/ui/place-bid-form";
import { useAuth } from "@/entities/session/model/auth-context";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { getAuctionEffectiveCurrentPrice } from "@/shared/lib/trading-domain";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { EmptyState } from "@/shared/ui/empty-state";
import { Notice } from "@/shared/ui/notice";

export default function AuctionDetailsPage() {
  const params = useParams();
  const { session } = useAuth();
  const rawId = params.id;
  const auctionId = Array.isArray(rawId) ? rawId[0] : rawId ?? "";
  const auctionQuery = useAuctionDetailsQuery(auctionId, session);
  const lotsQuery = useLotsQuery();
  const productsQuery = useProductsQuery();
  const fishQuery = useFishCatalogQuery(session);

  if (!auctionId) {
    return (
      <EmptyState
        title="Аукцион не выбран"
        description="Откройте страницу из списка аукционов."
        actionHref="/auctions"
        actionLabel="К списку аукционов"
      />
    );
  }

  const details = auctionQuery.data?.data;

  if (!details?.auction) {
    return (
      <EmptyState
        title="Аукцион не найден"
        description="Проверьте ссылку или вернитесь к списку аукционов."
        actionHref="/auctions"
        actionLabel="Вернуться к аукционам"
      />
    );
  }

  const { auction, bids, projection, deal } = details;
  const lot =
    (lotsQuery.data?.data ?? []).find((item) => item.auctionId === auction.id) ??
    (lotsQuery.data?.data ?? []).find((item) => item.id === auction.lotId);
  const product = lot
    ? (productsQuery.data?.data ?? []).find((item) => item.id === lot.productId)
    : undefined;
  const fish = product
    ? (fishQuery.data?.data ?? []).find((item) => item.id === product.fishId)
    : undefined;
  const productTitle = projection?.productSnapshot.name ?? lot?.productLabel ?? product?.fishName ?? "Продукт";
  const fishDescription = projection?.productSnapshot.description || fish?.description;
  const currentPrice = getAuctionEffectiveCurrentPrice(auction, bids);
  const sellerCompanyId = auction.sellerCompanyId ?? lot?.sellerCompanyId ?? projection?.supplierId;

  return (
    <div className="page-stack">
      <div className="section-heading">
        <div className="page-heading">
          <p className="eyebrow">Аукцион</p>
          <h1>{shortId(auction.id)}</h1>
          <p className="muted">
            Статус: {auction.state} · лот {shortId(auction.lotId)}
          </p>
        </div>
        <Button onClick={() => auctionQuery.refetch()} variant="secondary" type="button">
          Обновить
        </Button>
      </div>

      <div className="info-grid">
        <Card className="form-card">
          <div className="stack-md">
            <h2>Сводка аукциона</h2>
            <div className="metric-grid">
              <div>
                <span>Текущая ставка</span>
                <strong>{formatMoney(currentPrice || auction.finalPrice)}</strong>
              </div>
              <div>
                <span>Финальная цена</span>
                <strong>{formatMoney(auction.finalPrice)}</strong>
              </div>
              <div>
                <span>Начало</span>
                <strong>{formatDateTime(auction.startsAt)}</strong>
              </div>
              <div>
                <span>Окончание</span>
                <strong>{formatDateTime(auction.endsAt)}</strong>
              </div>
              <div>
                <span>Лидер</span>
                <strong>{auction.leaderCompanyId ?? "н/д"}</strong>
              </div>
              <div>
                <span>Победитель</span>
                <strong>{auction.winnerCompanyId ?? "н/д"}</strong>
              </div>
            </div>
          </div>
        </Card>

        <Card className="form-card">
          <div className="stack-md">
            <h2>Сделать ставку</h2>
            <p className="muted">Разместите ставку для участия в торгах по выбранному лоту.</p>
            <PlaceBidForm
              auctionId={auction.id}
              auctionState={auction.state}
              startsAt={auction.startsAt}
              endsAt={auction.endsAt}
              currentPrice={currentPrice}
              sellerCompanyId={sellerCompanyId}
              leaderCompanyId={auction.leaderCompanyId}
              bids={bids}
            />
          </div>
        </Card>
      </div>

      <div className="info-grid">
        <Card className="form-card">
          <div className="stack-md">
            <h2>Продукт</h2>
            <div>
              <p className="eyebrow">Наименование</p>
              <h3>{productTitle}</h3>
              {fishDescription ? <p className="muted">{fishDescription}</p> : null}
            </div>
            <div className="metric-grid">
              <div>
                <span>Рыба</span>
                <strong>{product?.fishName ?? projection?.productSnapshot.name ?? "н/д"}</strong>
              </div>
              <div>
                <span>Обработка</span>
                <strong>{product?.processingType ?? projection?.productSnapshot.processingType ?? "н/д"}</strong>
              </div>
              <div>
                <span>Размер</span>
                <strong>{product?.size ?? projection?.productSnapshot.size ?? "н/д"}</strong>
              </div>
              <div>
                <span>Вес</span>
                <strong>
                  {product ? `${product.weight} ${product.unit}` : projection ? `${projection.productSnapshot.weight} ${projection.productSnapshot.unit}` : "н/д"}
                </strong>
              </div>
              <div>
                <span>Продавец</span>
                <strong>{sellerCompanyId ?? "н/д"}</strong>
              </div>
              <div>
                <span>Лот</span>
                <strong>{shortId(auction.lotId)}</strong>
              </div>
            </div>
          </div>
        </Card>

        <Card className="form-card">
          <div className="stack-md">
            <h2>История ставок</h2>
            {bids.length ? (
              <table className="table">
                <thead>
                  <tr>
                    <th>Компания</th>
                    <th>Сумма</th>
                    <th>Время</th>
                  </tr>
                </thead>
                <tbody>
                  {bids.map((bid, index) => (
                    <tr key={`${bid.auctionId}-${bid.bidderCompanyId}-${bid.amount}-${bid.placedAt}-${index}`}>
                      <td>{bid.bidderCompanyId}</td>
                      <td>{formatMoney(bid.amount)}</td>
                      <td>{formatDateTime(bid.placedAt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p className="muted">Ставок по этому аукциону пока нет.</p>
            )}
          </div>
        </Card>
      </div>

      {projection || deal ? (
        <Card className="form-card">
          <div className="stack-md">
            <h2>Сопутствующая информация</h2>
            {projection ? (
              <div className="metric-grid">
                <div>
                  <span>Статус публикации</span>
                  <strong>{projection.status}</strong>
                </div>
                <div>
                  <span>Поставщик</span>
                  <strong>{projection.supplierId}</strong>
                </div>
                <div>
                  <span>Дата публикации</span>
                  <strong>{formatDateTime(projection.publishedAt)}</strong>
                </div>
                <div>
                  <span>Стартовая цена</span>
                  <strong>{formatMoney(projection.startPrice)}</strong>
                </div>
              </div>
            ) : null}
            {deal ? (
              <Notice tone="success" title="Сделка">
                {deal.id} · статус {deal.status} · сумма {formatMoney(deal.totalAmount)}
              </Notice>
            ) : null}
          </div>
        </Card>
      ) : null}
    </div>
  );
}
