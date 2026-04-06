"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { FishCard } from "@/entities/fish/ui/fish-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { EmptyState } from "@/shared/ui/empty-state";

export default function CatalogPage() {
  const { session } = useAuth();
  const fishQuery = useFishCatalogQuery(session);
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);

  const items = useMemo(() => {
    return (fishQuery.data?.data ?? []).filter((item) => {
      return `${item.name} ${item.description}`.toLowerCase().includes(deferredSearch.toLowerCase());
    });
  }, [deferredSearch, fishQuery.data?.data]);

  return (
    <div className="page-stack">
      <div className="page-heading">
        <p className="eyebrow">Catalog</p>
        <h1>Рыба / каталог</h1>
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
      />

      {items.length ? (
        <div className="card-grid card-grid-3">
          {items.map((item) => (
            <FishCard key={item.id} fish={item} />
          ))}
        </div>
      ) : (
        <EmptyState
          title="Каталог пуст"
          description="Создайте первую карточку рыбы для дальнейшего product / lot flow."
          actionHref="/create/fish"
          actionLabel="Создать рыбу"
        />
      )}
    </div>
  );
}
