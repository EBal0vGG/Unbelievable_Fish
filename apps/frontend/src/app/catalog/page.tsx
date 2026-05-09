"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { FishCard } from "@/entities/fish/ui/fish-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { isAdminSession } from "@/shared/lib/access";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { EmptyState } from "@/shared/ui/empty-state";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";

export default function CatalogPage() {
  const { session } = useAuth();
  const canManageFish = isAdminSession(session);
  const fishQuery = useFishCatalogQuery(session);
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);

  const items = useMemo(() => {
    return (fishQuery.data?.data ?? []).filter((item) => {
      return `${item.name} ${item.description}`.toLowerCase().includes(deferredSearch.toLowerCase());
    });
  }, [deferredSearch, fishQuery.data?.data]);
  const categoryChips = items.slice(0, 6).map((item) => item.name);

  return (
    <div className="page-stack">
      <PageHeader
        compact
        eyebrow="Каталог"
        title="Каталог рыбы"
        description="Базовая витрина товарных категорий для продуктов, лотов и торгов."
        metrics={
          <>
            <div>
              <span>Позиций</span>
              <strong>{items.length}</strong>
            </div>
            <div>
              <span>Справочник</span>
              <strong>Платформа</strong>
            </div>
          </>
        }
      />

      <SectionCard eyebrow="Directory" title="Поиск по каталогу">
        <div className="catalog-toolbar">
          <div className="category-strip">
            {categoryChips.map((item) => (
              <span className="category-chip" key={item}>
                {item}
              </span>
            ))}
          </div>
          <span className="muted">Компактная витрина рыбных активов</span>
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
          searchPlaceholder="Название рыбы или описание"
        />
      </SectionCard>

      {items.length ? (
        <div className="directory-grid">
          {items.map((item) => (
            <FishCard key={item.id} fish={item} />
          ))}
        </div>
      ) : (
        <EmptyState
          title="Каталог пуст"
          description={
            canManageFish
              ? "Создайте первую позицию каталога, чтобы продавцы могли собирать продукты."
              : "Каталог рыбы пока пуст."
          }
          actionHref={canManageFish ? "/create/fish" : undefined}
          actionLabel={canManageFish ? "Создать рыбу" : undefined}
        />
      )}
    </div>
  );
}
