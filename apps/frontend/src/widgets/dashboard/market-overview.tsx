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
import { isSellerSession } from "@/shared/lib/access";
import { formatMoney } from "@/shared/lib/format";
import { auctionStateLabels, lotStatusLabels, productStatusLabels } from "@/shared/lib/labels";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Field } from "@/shared/ui/field";
import { Select } from "@/shared/ui/select";

export function MarketOverview() {
  const { session } = useAuth();
  const canCreateSupply = isSellerSession(session);
  const canSeeDeals = Boolean(session?.companyId);
  const productsQuery = useProductsQuery();
  const lotsQuery = useLotsQuery();
  const auctionsQuery = useAuctionsQuery();
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

  const filtered = useMemo(() => {
    const normalized = deferredSearch.trim().toLowerCase();
    const matchesSearch = (value: string) => !normalized || value.toLowerCase().includes(normalized);

    return {
      products: products.filter(
        (item) =>
          matchesSearch(
            `${item.fishName} ${item.processingType} ${item.size} ${item.ownerCompanyId} ${item.id}`,
          ) && (productStatus === "all" || item.status === productStatus),
      ),
      lots: lots.filter(
        (item) =>
          matchesSearch(`${item.productLabel} ${item.sellerCompanyId} ${item.id}`) &&
          (lotStatus === "all" || item.status === lotStatus),
      ),
      auctions: auctions.filter(
        (item) =>
          matchesSearch(`${item.id} ${item.lotId} ${item.state} ${item.sellerCompanyId ?? ""}`) &&
          (auctionStatus === "all" || item.state === auctionStatus),
      ),
      deals: deals.filter(
        (item) =>
          matchesSearch(
            `${item.id} ${item.auctionId} ${item.supplierId} ${item.customerId} ${item.productSnapshot.name}`,
          ) && (dealStatus === "all" || item.status === dealStatus),
      ),
    };
  }, [
    auctionStatus,
    auctions,
    dealStatus,
    deals,
    deferredSearch,
    lotStatus,
    lots,
    productStatus,
    products,
  ]);

  const activeAuctions = auctions.filter((item) => item.state === "PUBLISHED").length;
  const activeDeals = deals.filter((item) => item.status !== "completed" && item.status !== "cancelled");
  const dealTurnover = activeDeals.reduce((sum, item) => sum + item.totalAmount, 0);
  const bestAuction = auctions
    .filter((item) => item.state === "PUBLISHED")
    .sort((left, right) => (right.currentPrice ?? 0) - (left.currentPrice ?? 0))[0];

  return (
    <div className="stack-xl">
      <section className="market-hero">
        <div className="hero-content">
          <p className="eyebrow">Оптовая рыбная биржа</p>
          <h1>Торги, поставки и сделки для крупных покупателей</h1>
          <p className="hero-copy">
            Живой marketplace для рыбной продукции: поставщик выпускает лот, покупатель конкурирует в торгах,
            победитель проходит контракт, оплату и отгрузку.
          </p>
          <div className="hero-actions">
            <Link className={buttonStyles({ variant: "primary", size: "lg" })} href="/auctions">
              Открыть торги
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
          <span>Лидер торгов</span>
          <strong>{bestAuction ? formatMoney(bestAuction.currentPrice) : "нет активных ставок"}</strong>
          <small>{bestAuction ? bestAuction.sellerCompanyId : "Ожидаем публикацию лота"}</small>
        </div>
      </section>

      <section className="stats-grid">
        <Card className="stat-card stat-card-primary">
          <span>Активные торги</span>
          <strong>{activeAuctions}</strong>
        </Card>
        <Card className="stat-card">
          <span>Продукты</span>
          <strong>{products.length}</strong>
        </Card>
        <Card className="stat-card">
          <span>Лоты</span>
          <strong>{lots.length}</strong>
        </Card>
        {canSeeDeals ? (
          <>
            <Card className="stat-card">
              <span>Ваши сделки</span>
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
          <h2>От продукта до закрытой сделки</h2>
        </div>
        <div className="command-lane">
          <Link href="/catalog">Каталог</Link>
          <span>Продукт</span>
          <span>Лот</span>
          <Link href="/auctions">Торги</Link>
          {canSeeDeals ? <Link href="/deals">Ваши сделки</Link> : <span>Закрытие</span>}
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
            <Field label="Продукты">
              <Select value={productStatus} onChange={(event) => setProductStatus(event.target.value)}>
                <option value="all">Все</option>
                <option value="DRAFT">{productStatusLabels.DRAFT}</option>
                <option value="PUBLISHED">{productStatusLabels.PUBLISHED}</option>
              </Select>
            </Field>
            <Field label="Лоты">
              <Select value={lotStatus} onChange={(event) => setLotStatus(event.target.value)}>
                <option value="all">Все</option>
                <option value="DRAFT">{lotStatusLabels.DRAFT}</option>
                <option value="PUBLISHED">{lotStatusLabels.PUBLISHED}</option>
                <option value="CLOSED">{lotStatusLabels.CLOSED}</option>
                <option value="CANCELLED">{lotStatusLabels.CANCELLED}</option>
              </Select>
            </Field>
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
              <Field label="Ваши сделки">
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
            <h2>Живая лента</h2>
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
              <h2>Сделки в работе</h2>
            </div>
            <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/deals">
              Все ваши сделки
            </Link>
          </div>
          <div className="card-grid card-grid-3">
            {filtered.deals.slice(0, 3).map((item) => (
              <DealCard key={item.id} deal={item} />
            ))}
          </div>
        </section>
      ) : null}

      <section className="market-sections">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Поставка</p>
            <h2>Лоты и продукты</h2>
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
    </div>
  );
}
