"use client";

import Link from "next/link";
import { useDeferredValue, useMemo, useState } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { useDealsQuery } from "@/entities/deal/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { getMainInterfaceMode, isSellerSession } from "@/shared/lib/access";
import { displayCompany } from "@/shared/lib/display";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { auctionStateLabels, lotStatusLabels, productStatusLabels } from "@/shared/lib/labels";
import { buttonStyles } from "@/shared/ui/button";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { Pipeline } from "@/shared/ui/pipeline";
import { SectionCard } from "@/shared/ui/section-card";
import { Select } from "@/shared/ui/select";
import { StatCard } from "@/shared/ui/stat-card";
import { StatusBadge } from "@/shared/ui/status-badge";

const ecosystem = [
  {
    href: "/catalog",
    title: "Каталог продукции",
    copy: "Единая база рыбных активов для продуктов и лотов.",
    icon: "CAT",
  },
  {
    href: "/products",
    title: "Продукты компании",
    copy: "Позиции поставщика с обработкой, размером и весом.",
    icon: "PRD",
  },
  {
    href: "/lots",
    title: "Лоты",
    copy: "Партии с ценой старта, объемом и расписанием торгов.",
    icon: "LOT",
  },
  {
    href: "/auctions",
    title: "Торги",
    copy: "Аукционная лента со статусами, ценой и продавцами.",
    icon: "BID",
  },
  {
    href: "/deals",
    title: "Сделки",
    copy: "Контракт, оплата и отгрузка после выбора победителя.",
    icon: "DOC",
  },
  {
    href: "/me",
    title: "Профиль надежности",
    copy: "Роль, компания, рейтинг и операционная активность.",
    icon: "4.8",
  },
];

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
  const [marketTab, setMarketTab] = useState("seafood");
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

  const filtered = useMemo(() => {
    const normalized = deferredSearch.trim().toLowerCase();
    const matchesSearch = (value: string) => !normalized || value.toLowerCase().includes(normalized);

    return {
      products: visibleProducts.filter(
        (item) =>
          matchesSearch(`${item.fishName} ${item.processingType} ${item.size} ${item.ownerCompanyId} ${item.id}`) &&
          (productStatus === "all" || item.status === productStatus),
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
  const offerItems = filtered.auctions.slice(0, 8);
  const pipelineSteps = [
    { label: "Продукт", detail: "Создайте позицию компании", href: "/products" },
    { label: "Лот", detail: "Соберите партию и цену", href: "/create/lot" },
    { label: "Торги", detail: "Запустите прием ставок", href: "/auctions" },
    { label: "Победитель", detail: "Зафиксируйте результат" },
    { label: "Сделка", detail: "Контракт, оплата, отгрузка", href: canSeeDeals ? "/deals" : undefined },
  ];

  return (
    <div className="marketplace-home marketplace-home-stack">
      <section className="hero-marketplace">
        <div className="hero-marketplace-copy">
          <p className="eyebrow">Отраслевая B2B-платформа</p>
          <h1>Маркетплейс рыбных активов для B2B-сделок</h1>
          <p>
            Каталог, продукты компании, лоты, аукционы и сделки в едином рабочем пространстве для поставщиков и
            покупателей рыбной продукции.
          </p>
          <div className="hero-actions">
            <Link className={buttonStyles({ variant: "primary", size: "lg" })} href="/auctions">
              Открыть торги
            </Link>
            {canCreateSupply ? (
              <Link className={buttonStyles({ variant: "secondary", size: "lg" })} href="/create/lot">
                Создать лот
              </Link>
            ) : null}
            {canSeeDeals ? (
              <Link className={buttonStyles({ variant: "ghost", size: "lg" })} href="/deals">
                Смотреть сделки
              </Link>
            ) : null}
          </div>
        </div>

        <div className="market-visual" aria-hidden="true">
          <div className="iso-card iso-card-main">
            <span>Торги</span>
            <strong>{bestAuction ? formatMoney(bestAuction.currentPrice) : "ожидание лота"}</strong>
            <small>{bestAuction?.sellerCompanyId ? displayCompany(bestAuction.sellerCompanyId) : "Поставка готовится"}</small>
          </div>
          <div className="iso-card iso-card-chart">
            <span>Индекс</span>
            <i />
            <i />
            <i />
          </div>
          <div className="iso-card iso-card-deal">
            <span>Сделки</span>
            <strong>{activeDeals.length}</strong>
          </div>
          <div className="iso-orbit iso-orbit-fish">Рыба</div>
          <div className="iso-orbit iso-orbit-bid">₽</div>
          <div className="iso-orbit iso-orbit-auction">LOT</div>
        </div>

        <div className="hero-search-panel">
          <div className="search-tabs">
            {[
              ["seafood", "Рыба и морепродукты"],
              ["logistics", "Логистика"],
              ["storage", "Хранение"],
            ].map(([value, label]) => (
              <button
                className={marketTab === value ? "search-tab search-tab-active" : "search-tab"}
                key={value}
                onClick={() => setMarketTab(value)}
                type="button"
              >
                {label}
              </button>
            ))}
          </div>
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
            searchPlaceholder={
              marketTab === "seafood"
                ? "Искать продукт, лот или компанию"
                : marketTab === "logistics"
                  ? "Искать направление или поставщика"
                  : "Искать условия хранения"
            }
            extraFilters={
              <>
                {!isBuyerInterface ? (
                  <Field label="Категория">
                    <Select value={productStatus} onChange={(event) => setProductStatus(event.target.value)}>
                      <option value="all">Все продукты</option>
                      <option value="DRAFT">{productStatusLabels.DRAFT}</option>
                      <option value="PUBLISHED">{productStatusLabels.PUBLISHED}</option>
                    </Select>
                  </Field>
                ) : null}
                <Field label="Лоты">
                  <Select value={lotStatus} onChange={(event) => setLotStatus(event.target.value)}>
                    <option value="all">Все лоты</option>
                    <option value="DRAFT">{lotStatusLabels.DRAFT}</option>
                    <option value="PUBLISHED">{lotStatusLabels.PUBLISHED}</option>
                    <option value="CLOSED">{lotStatusLabels.CLOSED}</option>
                    <option value="CANCELLED">{lotStatusLabels.CANCELLED}</option>
                  </Select>
                </Field>
                <Field label="Торги">
                  <Select value={auctionStatus} onChange={(event) => setAuctionStatus(event.target.value)}>
                    <option value="all">Все торги</option>
                    <option value="PUBLISHED">{auctionStateLabels.PUBLISHED}</option>
                    <option value="CLOSED">{auctionStateLabels.CLOSED}</option>
                    <option value="WON">{auctionStateLabels.WON}</option>
                    <option value="CANCELLED">{auctionStateLabels.CANCELLED}</option>
                  </Select>
                </Field>
                <Link className={buttonStyles({ variant: "primary", size: "md" })} href="/auctions">
                  Показать предложения
                </Link>
              </>
            }
          />
        </div>
      </section>

      <section className="stats-grid market-metrics">
        <StatCard label="Активные торги" value={activeAuctions} tone="primary" />
        {!isBuyerInterface ? <StatCard label="Продукты" value={visibleProducts.length} /> : null}
        {!isBuyerInterface ? <StatCard label="Лоты" value={visibleLots.length} /> : null}
        {canSeeDeals ? <StatCard label="Сделки в работе" value={activeDeals.length} /> : null}
        {canSeeDeals ? <StatCard label="Оборот" value={formatMoney(dealTurnover)} tone="accent" /> : null}
      </section>

      <section className="offer-ticker-section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Актуальные предложения</p>
            <h2>Лента торгов и лотов</h2>
          </div>
          <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/auctions">
            Все торги
          </Link>
        </div>
        {offerItems.length ? (
          <div className="offer-ticker">
            {offerItems.map((item) => {
              const lot = lots.find((lotItem) => lotItem.id === item.lotId);

              return (
                <Link className="offer-chip" href={`/auctions/${item.id}`} key={item.id}>
                  <span>{lot?.productLabel ?? `Лот ${shortId(item.lotId)}`}</span>
                  <strong>{formatMoney(item.currentPrice ?? item.finalPrice)}</strong>
                  <small>{displayCompany(item.sellerCompanyId ?? lot?.sellerCompanyId)} · {auctionStateLabels[item.state]}</small>
                </Link>
              );
            })}
          </div>
        ) : (
          <EmptyState
            framed={false}
            title="Предложения появятся после публикации лотов"
            description="Лента обновится после публикации первых лотов и запуска торгов."
          />
        )}
      </section>

      <section className="ecosystem-section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Экосистема</p>
            <h2>Рабочие модули рыбной биржи</h2>
          </div>
        </div>
        <div className="ecosystem-grid">
          {ecosystem.map((item) => (
            <Link className="ecosystem-card" href={item.href} key={item.href}>
              <span>{item.icon}</span>
              <h3>{item.title}</h3>
              <p>{item.copy}</p>
            </Link>
          ))}
        </div>
      </section>

      <div className="trust-workflow-grid">
        <SectionCard
          eyebrow="Контролируемые сделки"
          title="От продукта до закрытия внутри платформы"
          description="Платформа сохраняет текущий бизнес-процесс: продукт, лот, торги, победитель и сделка."
        >
          <Pipeline steps={pipelineSteps} />
        </SectionCard>

        <SectionCard eyebrow="Market overview" title="Операционный срез">
          <div className="metric-grid">
            <div>
              <span>Лидер торгов</span>
              <strong>{displayCompany(bestAuction?.leaderCompanyId ?? bestAuction?.sellerCompanyId)}</strong>
            </div>
            <div>
              <span>Лучшее предложение</span>
              <strong>{bestAuction ? formatMoney(bestAuction.currentPrice) : "—"}</strong>
            </div>
            <div>
              <span>Завершение</span>
              <strong>{formatDateTime(bestAuction?.endsAt)}</strong>
            </div>
            <div>
              <span>Сделки</span>
              <strong>{filtered.deals.length}</strong>
            </div>
          </div>
          {canSeeDeals ? (
            <Field label="Статус сделок">
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
        </SectionCard>
      </div>

      <SectionCard eyebrow="Live auctions" title="Торговая лента">
        {filtered.auctions.length ? (
          <div className="data-list auction-feed">
            {filtered.auctions.slice(0, 5).map((item) => {
              const lot = lots.find((lotItem) => lotItem.id === item.lotId);
              const rowTone =
                item.state === "PUBLISHED"
                  ? "data-row-active"
                  : item.state === "CANCELLED"
                    ? "data-row-cancelled"
                    : "data-row-complete";

              return (
                <div className={`data-row auction-row ${rowTone}`} key={item.id}>
                  <div className="auction-thumb">
                    <span>{lot?.productLabel?.slice(0, 2).toUpperCase() ?? "UF"}</span>
                  </div>
                  <div className="data-row-main">
                    <h3>{lot?.productLabel ?? `Аукцион ${shortId(item.id)}`}</h3>
                    <p>Сессия #{shortId(item.id)} · лот {shortId(item.lotId)}</p>
                  </div>
                  <div className="data-cell">
                    <span>Статус</span>
                    <strong>
                      <StatusBadge status={item.state} label={auctionStateLabels[item.state]} />
                    </strong>
                  </div>
                  <div className="data-cell">
                    <span>Цена</span>
                    <strong>{formatMoney(item.currentPrice ?? item.finalPrice)}</strong>
                  </div>
                  <div className="data-cell">
                    <span>Завершение</span>
                    <strong>{formatDateTime(item.endsAt)}</strong>
                  </div>
                  <div className="data-actions">
                    <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href={`/auctions/${item.id}`}>
                      Открыть
                    </Link>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <EmptyState
            framed={false}
            size="compact"
            title="Торгов пока нет"
            description="После публикации лотов здесь появятся активные и завершенные аукционы."
          />
        )}
      </SectionCard>
    </div>
  );
}
