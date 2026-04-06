"use client";

import Link from "next/link";
import { useDeferredValue, useMemo, useState } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { AuctionCard } from "@/entities/auction/ui/auction-card";
import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { FishCard } from "@/entities/fish/ui/fish-card";
import { useLotsQuery } from "@/entities/lot/model/hooks";
import { LotCard } from "@/entities/lot/ui/lot-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";

export function MarketOverview() {
  const { session } = useAuth();
  const fishQuery = useFishCatalogQuery(session);
  const lotsQuery = useLotsQuery();
  const auctionsQuery = useAuctionsQuery();
  const [search, setSearch] = useState("");
  const [source, setSource] = useState("all");
  const deferredSearch = useDeferredValue(search);

  const fish = fishQuery.data?.data ?? [];
  const lots = lotsQuery.data?.data ?? [];
  const auctions = auctionsQuery.data?.data ?? [];

  const filtered = useMemo(() => {
    const normalized = deferredSearch.trim().toLowerCase();
    if (!normalized) {
      return {
        fish,
        lots,
        auctions,
      };
    }

    return {
      fish: fish.filter(
        (item) =>
          `${item.name} ${item.description}`.toLowerCase().includes(normalized) &&
          (source === "all" || item.source === source),
      ),
      lots: lots.filter((item) =>
        `${item.productLabel} ${item.sellerCompanyId} ${item.id}`.toLowerCase().includes(normalized) &&
        (source === "all" || item.source === source),
      ),
      auctions: auctions.filter(
        (item) =>
          `${item.id} ${item.lotId} ${item.state} ${item.sellerCompanyId ?? ""}`
            .toLowerCase()
            .includes(normalized) &&
          (source === "all" || item.source === source),
      ),
    };
  }, [auctions, deferredSearch, fish, lots, source]);

  return (
    <div className="stack-xl">
      <section className="hero-panel">
        <div className="stack-md">
          <h1>Рыбная биржа</h1>
        </div>
        <div className="hero-actions">
          <Link className={buttonStyles({ variant: "primary" })} href="/create/lot">
            Разместить лот
          </Link>
          <Link className={buttonStyles({ variant: "secondary" })} href="/auctions">
            Открыть аукционы
          </Link>
        </div>
      </section>

      <section className="stats-grid">
        <Card className="stat-card">
          <span>Рыба в каталоге</span>
          <strong>{fish.length}</strong>
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
          <span>Текущий контекст</span>
          <strong>{session ? `${session.companyId} / ${session.userId}` : "гость"}</strong>
        </Card>
      </section>

      <FilterBar
        search={search}
        onSearchChange={setSearch}
        status="all"
        onStatusChange={() => undefined}
        statusOptions={[{ label: "Все статусы", value: "all" }]}
        source={source}
        onSourceChange={setSource}
      />

      <section className="stack-md">
        <div className="section-heading">
          <div>
            <p className="eyebrow">Рыба / каталог</p>
            <h2>Ключевые позиции</h2>
          </div>
          <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/catalog">
            Весь каталог
          </Link>
        </div>
        <div className="card-grid card-grid-3">
          {filtered.fish.slice(0, 3).map((item) => (
            <FishCard key={item.id} fish={item} />
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
