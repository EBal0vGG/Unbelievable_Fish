"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useLotsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { LotCard } from "@/entities/lot/ui/lot-card";
import { isOwnedLot } from "@/shared/lib/access";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { EmptyState } from "@/shared/ui/empty-state";

export default function LotsPage() {
  const { session } = useAuth();
  const lotsQuery = useLotsQuery();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const deferredSearch = useDeferredValue(search);

  const items = useMemo(() => {
    return (lotsQuery.data?.data ?? []).filter((item) => {
      if (!isOwnedLot(item, session)) {
        return false;
      }
      const searchMatch = `${item.productLabel} ${item.sellerCompanyId} ${item.id}`
        .toLowerCase()
        .includes(deferredSearch.toLowerCase());
      const statusMatch = status === "all" || item.status === status;
      return searchMatch && statusMatch;
    });
  }, [deferredSearch, lotsQuery.data?.data, session, status]);

  return (
    <div className="page-stack">
      <div className="page-heading">
        <p className="eyebrow">Lots</p>
        <h1>Лоты</h1>
      </div>

      <FilterBar
        search={search}
        onSearchChange={setSearch}
        status={status}
        onStatusChange={setStatus}
        statusOptions={[
          { label: "Все статусы", value: "all" },
          { label: "DRAFT", value: "DRAFT" },
          { label: "PUBLISHED", value: "PUBLISHED" },
          { label: "CLOSED", value: "CLOSED" },
          { label: "CANCELLED", value: "CANCELLED" },
        ]}
        source="all"
        onSourceChange={() => undefined}
        showSource={false}
      />

      {items.length ? (
        <div className="card-grid card-grid-2">
          {items.map((item) => (
            <LotCard key={item.id} lot={item} />
          ))}
        </div>
      ) : (
        <EmptyState
          title="Лоты не найдены"
          description="У вас пока нет доступных лотов. Создайте свой лот или снимите фильтры."
          actionHref="/create/lot"
          actionLabel="Создать лот"
        />
      )}
    </div>
  );
}
