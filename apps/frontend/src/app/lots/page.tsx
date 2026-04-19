"use client";

import { useDeferredValue, useMemo, useState } from "react";

import { useLotsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { LotCard } from "@/entities/lot/ui/lot-card";
import { isOwnedLot, isSellerSession } from "@/shared/lib/access";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { Select } from "@/shared/ui/select";

export default function LotsPage() {
  const { session } = useAuth();
  const canCreateSupply = isSellerSession(session);
  const lotsQuery = useLotsQuery();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [auctionLink, setAuctionLink] = useState("all");
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
      const auctionMatch =
        auctionLink === "all" ||
        (auctionLink === "linked" ? Boolean(item.auctionId) : !item.auctionId);
      return searchMatch && statusMatch && auctionMatch;
    });
  }, [auctionLink, deferredSearch, lotsQuery.data?.data, session, status]);

  return (
    <div className="page-stack">
      <div className="page-heading">
        <p className="eyebrow">Лоты</p>
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
        searchPlaceholder="Название продукта, компания или номер лота"
        extraFilters={
          <Field label="Аукцион">
            <Select value={auctionLink} onChange={(event) => setAuctionLink(event.target.value)}>
              <option value="all">Все</option>
              <option value="linked">Есть аукцион</option>
              <option value="free">Без аукциона</option>
            </Select>
          </Field>
        }
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
          actionHref={canCreateSupply ? "/create/lot" : undefined}
          actionLabel={canCreateSupply ? "Создать лот" : undefined}
        />
      )}
    </div>
  );
}
