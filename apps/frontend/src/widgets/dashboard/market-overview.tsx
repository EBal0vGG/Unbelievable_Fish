"use client";

import Link from "next/link";
import { useDeferredValue, useMemo, useState } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { AuctionCard } from "@/entities/auction/ui/auction-card";
import { DealCard } from "@/entities/deal/ui/deal-card";
import { useDealsQuery } from "@/entities/deal/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { LotCard } from "@/entities/lot/ui/lot-card";
import { ProductCard } from "@/entities/product/ui/product-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { getMainInterfaceMode, isSellerSession } from "@/shared/lib/access";
import { formatMoney } from "@/shared/lib/format";
import { auctionStateLabels, lotStatusLabels, productStatusLabels } from "@/shared/lib/labels";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Field } from "@/shared/ui/field";
import { Select } from "@/shared/ui/select";

export function MarketOverview() {
  const { session } = useAuth();
  const interfaceMode = getMainInterfaceMode(session);
  const isBuyerInterface = interfaceMode === "buyer";
  const isSellerInterface = interfaceMode === "seller";
  const canCreateSupply = isSellerSession(session);
  const canSeeDeals = Boolean(session?.companyId);
  const productsQuery = useProductsQuery();
  const lotsQuery = useLotsQuery();
  const auctionsQuery = useAuctionsQuery(session);
  const dealsQuery = useDealsQuery(session);
  const [search, setSearch] = useState("");
  const [productStatus, setProductStatus] = useState("all");
  const [lotStatus, setLotStatus] = useState("all");
  const [auctionStatus, setAuctionStatus] = useState("all");
  const [dealStatus, setDealStatus] = useState("all");
  const deferredSearch = useDeferredValue(search);

  const products = productsQuery.data?.data ?? [];
  const lots = lotsQuery.data?.data ?? [];
  const auctions = auctionsQuery.data?.data ?? [];
  const deals = dealsQuery.data?.data ?? [];
  const visibleProducts = isSellerInterface
    ? products.filter((item) => item.ownerCompanyId === session?.companyId)
    : isBuyerInterface
      ? []
      : products;
  const visibleLots = isSellerInterface
    ? lots.filter((item) => item.sellerCompanyId === session?.companyId)
    : isBuyerInterface
      ? []
      : lots;
  const visibleAuctions = isSellerInterface
    ? auctions.filter((item) => {
        const lot = lots.find((lotItem) => lotItem.id === item.lotId);
        return item.sellerCompanyId === session?.companyId || lot?.sellerCompanyId === session?.companyId;
      })
    : auctions;
  const visibleDeals = isSellerInterface
    ? deals.filter((item) => item.supplierId === session?.companyId)
    : isBuyerInterface
      ? deals.filter((item) => item.customerId === session?.companyId)
      : deals;

  const heroText =
    interfaceMode === "seller"
      ? {
          eyebrow: "Кабинет продавца",
          title: "Лоты, торги и продажи вашей компании",
          copy:
            "Управляйте продуктами и лотами, отслеживайте свои аукционы и доводите продажи до оплаты и отгрузки.",
          ticketLabel: "Лучший активный лот",
          ticketEmpty: "Опубликуйте лот, чтобы начать торги",
          dealsTitle: "Сделки продаж",
          dealsLink: "Все продажи",
          auctionTitle: "Мои торги",
          supplyTitle: "Мои лоты и продукты",
        }
      : interfaceMode === "buyer"
        ? {
            eyebrow: "Кабинет покупателя",
            title: "Торги, ставки и закупочные сделки",
            copy:
              "Ищите подходящие лоты, участвуйте в торгах и ведите закупку через контракт, оплату и поставку.",
            ticketLabel: "Лидер торгов",
            ticketEmpty: "Нет активных ставок",
            dealsTitle: "Сделки покупок",
            dealsLink: "Все закупки",
            auctionTitle: "Доступные торги",
            supplyTitle: "Лоты и продукты",
          }
        : {
            eyebrow: "Оптовая рыбная биржа",
            title: "Торги, поставки и сделки для крупных покупателей",
            copy:
              "Живой marketplace для рыбной продукции: поставщик выпускает лот, покупатель конкурирует в торгах, победитель проходит контракт, оплату и отгрузку.",
            ticketLabel: "Лидер торгов",
            ticketEmpty: "нет активных ставок",
            dealsTitle: "Сделки в работе",
            dealsLink: "Все ваши сделки",
            auctionTitle: "Живая лента",
            supplyTitle: "Лоты и продукты",
          };

  const filtered = useMemo(() => {
    const normalized = deferredSearch.trim().toLowerCase();
    const matchesSearch = (value: string) => !normalized || value.toLowerCase().includes(normalized);

    return {
      products: visibleProducts.filter(
        (item) =>
          matchesSearch(
            `${item.fishName} ${item.processingType} ${item.size} ${item.ownerCompanyId} ${item.id}`,
          ) && (productStatus === "all" || item.status === productStatus),
      ),
      lots: visibleLots.filter(
        (item) =>
          matchesSearch(`${item.productLabel} ${item.sellerCompanyId} ${item.id}`) &&
          (lotStatus === "all" || item.status === lotStatus),
      ),
      auctions: visibleAuctions.filter(
        (item) =>
          matchesSearch(`${item.id} ${item.lotId} ${item.state} ${item.sellerCompanyId ?? ""}`) &&
          (auctionStatus === "all" || item.state === auctionStatus),
      ),
      deals: visibleDeals.filter(
        (item) =>
          matchesSearch(
            `${item.id} ${item.auctionId} ${item.supplierId} ${item.customerId} ${item.productSnapshot.name}`,
          ) && (dealStatus === "all" || item.status === dealStatus),
      ),
    };
  }, [
    auctionStatus,
    dealStatus,
    deferredSearch,
    lotStatus,
    productStatus,
    visibleAuctions,
    visibleDeals,
    visibleLots,
    visibleProducts,
  ]);

  const activeAuctions = visibleAuctions.filter((item) => item.state === "PUBLISHED").length;
  const activeDeals = visibleDeals.filter((item) => item.status !== "completed" && item.status !== "cancelled");
  const dealTurnover = activeDeals.reduce((sum, item) => sum + item.totalAmount, 0);
  const bestAuction = visibleAuctions
    .filter((item) => item.state === "PUBLISHED")
    .sort((left, right) => (right.currentPrice ?? 0) - (left.currentPrice ?? 0))[0];

  return (
    <div className="stack-xl">
      <section className="market-hero">
        <div className="hero-content">
          <p className="eyebrow">{heroText.eyebrow}</p>
          <h1>{heroText.title}</h1>
          <p className="hero-copy">{heroText.copy}</p>
          <div className="hero-actions">
            <Link className={buttonStyles({ variant: "primary", size: "lg" })} href="/auctions">
              {isSellerInterface ? "Мои торги" : "Открыть торги"}
            </Link>
            {canCreateSupply ? (
              <Link className={buttonStyles({ variant: "secondary", size: "lg" })} href="/create/lot">
                Разместить лот
              </Link>
            ) : null}
            {canSeeDeals ? (
              <Link className={buttonStyles({ variant: "ghost", size: "lg" })} href="/deals">
                Ваши сделки
              </Link>
            ) : null}
          </div>
        </div>
        <div className="hero-market-ticket">
          <span>{heroText.ticketLabel}</span>
          <strong>{bestAuction ? formatMoney(bestAuction.currentPrice) : heroText.ticketEmpty}</strong>
          <small>{bestAuction ? bestAuction.sellerCompanyId : "Ожидаем публикацию лота"}</small>
        </div>
      </section>

      <section className="stats-grid">
        <Card className="stat-card stat-card-primary">
          <span>{isSellerInterface ? "Мои активные торги" : "Активные торги"}</span>
          <strong>{activeAuctions}</strong>
        </Card>
        {!isBuyerInterface ? (
          <>
            <Card className="stat-card">
              <span>{isSellerInterface ? "Мои продукты" : "Продукты"}</span>
              <strong>{visibleProducts.length}</strong>
            </Card>
            <Card className="stat-card">
              <span>{isSellerInterface ? "Мои лоты" : "Лоты"}</span>
              <strong>{visibleLots.length}</strong>
            </Card>
          </>
        ) : null}
        {canSeeDeals ? (
          <>
            <Card className="stat-card">
              <span>
                {isSellerInterface ? "Продажи в работе" : isBuyerInterface ? "Закупки в работе" : "Ваши сделки"}
              </span>
              <strong>{activeDeals.length}</strong>
            </Card>
            <Card className="stat-card">
              <span>Оборот в работе</span>
              <strong>{formatMoney(dealTurnover)}</strong>
            </Card>
          </>
        ) : null}
      </section>

      <section className="command-strip">
        <div className="command-copy">
          <p className="eyebrow">Операционный поток</p>
          <h2>
            {isBuyerInterface
              ? "От ставки до поставки"
              : isSellerInterface
                ? "От лота до закрытой продажи"
                : "От продукта до закрытой сделки"}
          </h2>
        </div>
        <div className="command-lane">
          <Link href="/catalog">Каталог</Link>
          {!isBuyerInterface ? (
            <>
              <span>Продукт</span>
              <span>Лот</span>
            </>
          ) : (
            <span>Ставка</span>
          )}
          <Link href="/auctions">Торги</Link>
          {canSeeDeals ? (
            <Link href="/deals">
              {isSellerInterface ? "Продажи" : isBuyerInterface ? "Закупки" : "Ваши сделки"}
            </Link>
          ) : (
            <span>Закрытие</span>
          )}
        </div>
      </section>

      <FilterBar
        search={search}
        onSearchChange={setSearch}
        status="all"
        onStatusChange={() => undefined}
        statusOptions={[{ label: "Все статусы", value: "all" }]}
        source="all"
        onSourceChange={() => undefined}
        showSource={false}
        showStatus={false}
        extraFilters={
          <>
            {!isBuyerInterface ? (
              <>
                <Field label={isSellerInterface ? "Мои продукты" : "Продукты"}>
                  <Select value={productStatus} onChange={(event) => setProductStatus(event.target.value)}>
                    <option value="all">Все</option>
                    <option value="DRAFT">{productStatusLabels.DRAFT}</option>
                    <option value="PUBLISHED">{productStatusLabels.PUBLISHED}</option>
                  </Select>
                </Field>
                <Field label={isSellerInterface ? "Мои лоты" : "Лоты"}>
                  <Select value={lotStatus} onChange={(event) => setLotStatus(event.target.value)}>
                    <option value="all">Все</option>
                    <option value="DRAFT">{lotStatusLabels.DRAFT}</option>
                    <option value="PUBLISHED">{lotStatusLabels.PUBLISHED}</option>
                    <option value="CLOSED">{lotStatusLabels.CLOSED}</option>
                    <option value="CANCELLED">{lotStatusLabels.CANCELLED}</option>
                  </Select>
                </Field>
              </>
            ) : null}
            <Field label="Торги">
              <Select value={auctionStatus} onChange={(event) => setAuctionStatus(event.target.value)}>
                <option value="all">Все</option>
                <option value="PUBLISHED">{auctionStateLabels.PUBLISHED}</option>
                <option value="CLOSED">{auctionStateLabels.CLOSED}</option>
                <option value="WON">{auctionStateLabels.WON}</option>
                <option value="CANCELLED">{auctionStateLabels.CANCELLED}</option>
              </Select>
            </Field>
            {canSeeDeals ? (
              <Field label={isSellerInterface ? "Продажи" : isBuyerInterface ? "Закупки" : "Ваши сделки"}>
                <Select value={dealStatus} onChange={(event) => setDealStatus(event.target.value)}>
                  <option value="all">Все</option>
                  <option value="pending">pending</option>
                  <option value="confirmed">confirmed</option>
                  <option value="contract_signed">contract_signed</option>
                  <option value="payment_requested">payment_requested</option>
                  <option value="paid">paid</option>
                  <option value="completed">completed</option>
                </Select>
              </Field>
            ) : null}
          </>
        }
      />

      <section className="market-sections">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Торги</p>
            <h2>{heroText.auctionTitle}</h2>
          </div>
          <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/auctions">
            Все торги
          </Link>
        </div>
        <div className="card-grid card-grid-2">
          {filtered.auctions.slice(0, 2).map((item) => (
            <AuctionCard
              key={item.id}
              auction={item}
              photo={lots.find((lot) => lot.id === item.lotId)?.photo}
              sellerCompanyId={item.sellerCompanyId}
            />
          ))}
        </div>
      </section>

      {canSeeDeals ? (
        <section className="market-sections">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Ваши сделки</p>
              <h2>{heroText.dealsTitle}</h2>
            </div>
            <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/deals">
              {heroText.dealsLink}
            </Link>
          </div>
          <div className="card-grid card-grid-3">
            {filtered.deals.slice(0, 3).map((item) => (
              <DealCard key={item.id} deal={item} viewerCompanyId={session?.companyId} />
            ))}
          </div>
        </section>
      ) : null}

      {!isBuyerInterface ? (
        <section className="market-sections">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Поставка</p>
              <h2>{heroText.supplyTitle}</h2>
            </div>
            <div className="inline-actions">
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/products">
                Продукты
              </Link>
              <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/lots">
                Лоты
              </Link>
            </div>
          </div>
          <div className="split-market-grid">
            <div className="card-grid">
              {filtered.products.slice(0, 2).map((item) => (
                <ProductCard key={item.id} product={item} />
              ))}
            </div>
            <div className="card-grid">
              {filtered.lots.slice(0, 2).map((item) => (
                <LotCard key={item.id} lot={item} />
              ))}
            </div>
          </div>
        </section>
      ) : null}
    </div>
  );
}
