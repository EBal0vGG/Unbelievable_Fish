"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useMutation } from "@tanstack/react-query";

import { useAuctionDetailsQuery } from "@/entities/auction/model/hooks";
import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { PlaceBidForm } from "@/features/auction/ui/place-bid-form";
import { useAuth } from "@/entities/session/model/auth-context";
import { cancelAuction, closeAuction } from "@/shared/api/trading-service";
import { displayCompany, displayId, displayText } from "@/shared/lib/display";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { auctionStateLabels } from "@/shared/lib/labels";
import { isAdminSession, isSellerSession } from "@/shared/lib/access";
import { getAuctionEffectiveCurrentPrice } from "@/shared/lib/trading-domain";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { EntityPhoto } from "@/shared/ui/entity-photo";
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

  const closeAuctionMu = useMutation({
    mutationFn: async () => {
      if (!session?.accessToken) {
        throw new Error("Войдите в систему");
      }
      await closeAuction(auctionId, session);
    },
    onSuccess: async () => {
      await auctionQuery.refetch();
    },
  });

  const cancelAuctionMu = useMutation({
    mutationFn: async () => {
      if (!session?.accessToken) {
        throw new Error("Войдите в систему");
      }
      await cancelAuction(auctionId, session);
    },
    onSuccess: async () => {
      await auctionQuery.refetch();
    },
  });

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

  if (auctionQuery.isPending) {
    return (
      <div className="page-stack">
        <Notice title="Загрузка аукциона">Запрашиваем торги, проекцию сделки и контекст лота…</Notice>
      </div>
    );
  }

  const details = auctionQuery.data?.data;

  if (auctionQuery.isError) {
    return (
      <div className="page-stack">
        <Notice tone="warning" title="Не удалось загрузить аукцион">
          {auctionQuery.error instanceof Error ? auctionQuery.error.message : "Повторите попытку или откройте список торгов."}
        </Notice>
        <EmptyState
          title="Аукцион недоступен"
          description="Проверьте сеть и сервисы trading/deals."
          actionHref="/auctions"
          actionLabel="Вернуться к аукционам"
        />
      </div>
    );
  }

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
  const hasBids = bids.length > 0;
  const showSellerCancel =
    Boolean(session) &&
    isSellerSession(session) &&
    sellerCompanyId === session?.companyId &&
    auction.state === "PUBLISHED" &&
    !hasBids;
  const showAdminClose = Boolean(session) && isAdminSession(session) && auction.state === "PUBLISHED";

  return (
    <div className="page-stack auction-detail-page">
      <div className="section-heading">
        <div className="page-heading">
          <p className="eyebrow">Аукцион</p>
          <h1>{productTitle}</h1>
          <p className="muted">
            {auctionStateLabels[auction.state]} · {displayId(auction.id, "Аукцион #")} · лот {shortId(auction.lotId)}
          </p>
        </div>
        <Button onClick={() => auctionQuery.refetch()} variant="secondary" type="button">
          Обновить
        </Button>
      </div>

      <div className="info-grid">
        <Card className="form-card auction-visual-card">
          <EntityPhoto src={lot?.photo} alt={productTitle} className="detail-photo" />
          <div className="stack-md">
            <div>
              <p className="eyebrow">Лот</p>
              <h2>{productTitle}</h2>
              {fishDescription ? <p className="muted">{fishDescription}</p> : null}
            </div>
            <div className="metric-grid">
              <div>
                <span>Продавец</span>
                <strong title={sellerCompanyId}>{displayCompany(sellerCompanyId)}</strong>
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
            <h2>Сводка аукциона</h2>
            <div className="metric-grid auction-summary-metrics">
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
                <strong title={auction.leaderCompanyId}>{auction.leaderCompanyId ? displayCompany(auction.leaderCompanyId) : "—"}</strong>
              </div>
              <div>
                <span>Победитель</span>
                <strong title={auction.winnerCompanyId}>{auction.winnerCompanyId ? displayCompany(auction.winnerCompanyId) : "—"}</strong>
              </div>
              <div>
                <span>Chain status</span>
                <strong>{auction.chainFinalizeStatus || "—"}</strong>
              </div>
              <div>
                <span>Chain tx</span>
                <strong title={auction.chainFinalizeTxHash}>{auction.chainFinalizeTxHash ? shortId(auction.chainFinalizeTxHash) : "—"}</strong>
              </div>
            </div>
          </div>
        </Card>

        {showAdminClose || showSellerCancel ? (
          <Card className="form-card">
            <div className="stack-md">
              <h2>Управление торгами</h2>
              {showAdminClose ? (
                <div className="stack-sm">
                  <Notice tone="warning" title="Операция администратора">
                    Принудительное закрытие обходит ограничения для продавца. Используйте только в demo / support.
                  </Notice>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={closeAuctionMu.isPending}
                    onClick={() => closeAuctionMu.mutate()}
                  >
                    {closeAuctionMu.isPending ? "Закрываем…" : "Закрыть аукцион (admin)"}
                  </Button>
                  {closeAuctionMu.error ? (
                    <p className="muted text-sm">{closeAuctionMu.error.message}</p>
                  ) : null}
                </div>
              ) : null}
              {showSellerCancel ? (
                <div className="stack-sm">
                  <p className="muted text-sm">
                    Отмена доступна для опубликованного лота без ставок. При наличии ставок обратитесь к администратору.
                  </p>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={cancelAuctionMu.isPending}
                    onClick={() => cancelAuctionMu.mutate()}
                  >
                    {cancelAuctionMu.isPending ? "Отмена…" : "Отменить аукцион"}
                  </Button>
                  {cancelAuctionMu.error ? (
                    <p className="muted text-sm">{cancelAuctionMu.error.message}</p>
                  ) : null}
                </div>
              ) : null}
            </div>
          </Card>
        ) : null}

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
              minBidStep={auction.minBidStep}
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
            <div className="metric-grid auction-product-metrics">
              <div>
                <span>Рыба</span>
                <strong>{displayText(product?.fishName ?? projection?.productSnapshot.name)}</strong>
              </div>
              <div>
                <span>Обработка</span>
                <strong>{displayText(product?.processingType ?? projection?.productSnapshot.processingType)}</strong>
              </div>
              <div>
                <span>Размер</span>
                <strong>{displayText(product?.size ?? projection?.productSnapshot.size)}</strong>
              </div>
              <div>
                <span>Вес</span>
                <strong>
                  {product ? `${product.weight} ${product.unit}` : projection ? `${projection.productSnapshot.weight} ${projection.productSnapshot.unit}` : "—"}
                </strong>
              </div>
              <div>
                <span>Продавец</span>
                <strong title={sellerCompanyId}>{displayCompany(sellerCompanyId)}</strong>
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
              <div className="table-scroll">
                <table className="table auction-bids-table">
                  <thead>
                    <tr>
                      <th>Компания</th>
                      <th>Сумма</th>
                      <th>Время</th>
                      <th>Chain</th>
                    </tr>
                  </thead>
                  <tbody>
                    {bids.map((bid, index) => (
                      <tr key={`${bid.auctionId}-${bid.bidderCompanyId}-${bid.amount}-${bid.placedAt}-${index}`}>
                        <td title={bid.bidderCompanyId}>{displayCompany(bid.bidderCompanyId)}</td>
                        <td>{formatMoney(bid.amount)}</td>
                        <td>{formatDateTime(bid.placedAt)}</td>
                        <td title={bid.chainTxHash}>{bid.chainStatus ?? "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
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
                  <strong title={projection.supplierId}>{displayCompany(projection.supplierId)}</strong>
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
              <div className="deal-inline stack-md">
                <div className="metric-grid">
                  <div>
                    <span>Сделка</span>
                    <strong title={deal.id}>{displayId(deal.id, "#")}</strong>
                  </div>
                  <div>
                    <span>Статус</span>
                    <strong>{deal.status}</strong>
                  </div>
                  <div>
                    <span>Сумма</span>
                    <strong>{formatMoney(deal.totalAmount)}</strong>
                  </div>
                  <div>
                    <span>Покупатель</span>
                    <strong title={deal.customerId}>{displayCompany(deal.customerId)}</strong>
                  </div>
                </div>
                <Link className="text-link" href={`/deals/${deal.id}`}>
                  Открыть lifecycle сделки
                </Link>
              </div>
            ) : null}
          </div>
        </Card>
      ) : null}
    </div>
  );
}
