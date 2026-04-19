"use client";

import Link from "next/link";
import { useDeferredValue, useMemo, useState } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { AuctionCard } from "@/entities/auction/ui/auction-card";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { LotCard } from "@/entities/lot/ui/lot-card";
import { ProductCard } from "@/entities/product/ui/product-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { isSellerSession } from "@/shared/lib/access";
import { buttonStyles } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Card } from "@/shared/ui/card";
import { Select } from "@/shared/ui/select";

export function MarketOverview() {
  const { session } = useAuth();
  const canCreateSupply = isSellerSession(session);
  const productsQuery = useProductsQuery();
  const lotsQuery = useLotsQuery();
  const auctionsQuery = useAuctionsQuery();
  const [search, setSearch] = useState("");
  const [productStatus, setProductStatus] = useState("all");
  const [lotStatus, setLotStatus] = useState("all");
  const [auctionStatus, setAuctionStatus] = useState("all");
  const deferredSearch = useDeferredValue(search);

  const products = productsQuery.data?.data ?? [];
  const lots = lotsQuery.data?.data ?? [];
  const auctions = auctionsQuery.data?.data ?? [];

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
    };
  }, [auctionStatus, auctions, deferredSearch, lotStatus, lots, productStatus, products]);

  return (
    <div className="stack-xl">
      <section className="hero-panel">
        <div className="stack-md">
          <p className="eyebrow">Биржа</p>
          <h1>Рыбная биржа</h1>
          <p className="hero-copy">
            Платформа для управления ассортиментом, лотами и торгами в одном рабочем пространстве.
          </p>
        </div>
        <div className="hero-actions">
          <Link className={buttonStyles({ variant: "secondary" })} href="/products">
            Мои продукты
          </Link>
          {canCreateSupply ? (
            <Link className={buttonStyles({ variant: "primary" })} href="/create/lot">
              Разместить лот
            </Link>
          ) : null}
          <Link className={buttonStyles({ variant: "secondary" })} href="/auctions">
            Открыть аукционы
          </Link>
        </div>
      </section>

      <section className="stats-grid">
        <Card className="stat-card">
          <span>Продукты</span>
          <strong>{products.length}</strong>
        </Card>
        <Card className="stat-card">
          <span>Лоты</span>
          <strong>{lots.length}</strong>
        </Card>
        <Card className="stat-card">
          <span>Аукционы</span>
          <strong>{auctions.length}</strong>
        </Card>
        <Card className="stat-card">
          <span>Мой профиль</span>
          <strong>{session ? "Рейтинг 4.8" : "Гостевой вход"}</strong>
        </Card>
      </section>

      <section className="card-grid card-grid-2">
        <Card className="form-card">
          <div className="stack-md">
            <p className="eyebrow">Логистика</p>
            <h2>Маршруты и окна отгрузки</h2>
            <p className="muted">
              Заглушка для планирования перевозок, статусов доставки и расписания отгрузок.
            </p>
          </div>
        </Card>
        <Card className="form-card">
          <div className="stack-md">
            <p className="eyebrow">Хранение</p>
            <h2>Остатки и свободные мощности</h2>
            <p className="muted">
              Заглушка для складских остатков, температуры хранения и доступных ячеек.
            </p>
          </div>
        </Card>
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
                <option value="DRAFT">DRAFT</option>
                <option value="PUBLISHED">PUBLISHED</option>
              </Select>
            </Field>
            <Field label="Лоты">
              <Select value={lotStatus} onChange={(event) => setLotStatus(event.target.value)}>
                <option value="all">Все</option>
                <option value="DRAFT">DRAFT</option>
                <option value="PUBLISHED">PUBLISHED</option>
                <option value="CLOSED">CLOSED</option>
                <option value="CANCELLED">CANCELLED</option>
              </Select>
            </Field>
            <Field label="Аукционы">
              <Select value={auctionStatus} onChange={(event) => setAuctionStatus(event.target.value)}>
                <option value="all">Все</option>
                <option value="PUBLISHED">PUBLISHED</option>
                <option value="CLOSED">CLOSED</option>
                <option value="WON">WON</option>
                <option value="CANCELLED">CANCELLED</option>
              </Select>
            </Field>
          </>
        }
      />

      <section className="stack-md">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Продукты</p>
            <h2>Текущие позиции</h2>
          </div>
          <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/products">
            Все продукты
          </Link>
        </div>
        <div className="card-grid card-grid-3">
          {filtered.products.slice(0, 3).map((item) => (
            <ProductCard key={item.id} product={item} />
          ))}
        </div>
      </section>

      <section className="stack-md">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Лоты</p>
            <h2>Активные позиции</h2>
          </div>
          <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/lots">
            Все лоты
          </Link>
        </div>
        <div className="card-grid card-grid-3">
          {filtered.lots.slice(0, 3).map((item) => (
            <LotCard key={item.id} lot={item} />
          ))}
        </div>
      </section>

      <section className="stack-md">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Аукционы</p>
            <h2>Торговая лента</h2>
          </div>
          <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/auctions">
            Все аукционы
          </Link>
        </div>
        <div className="card-grid card-grid-2">
          {filtered.auctions.slice(0, 2).map((item) => (
            <AuctionCard key={item.id} auction={item} sellerCompanyId={item.sellerCompanyId} />
          ))}
        </div>
      </section>
    </div>
  );
}
