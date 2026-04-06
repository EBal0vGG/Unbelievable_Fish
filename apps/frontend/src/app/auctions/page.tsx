"use client";

import { useDeferredValue, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { AuctionCard } from "@/entities/auction/ui/auction-card";
import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { FilterBar } from "@/features/marketplace/ui/filter-bar";
import { ApiError } from "@/shared/api/http-client";
import { placeBid } from "@/shared/api/trading-service";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Select } from "@/shared/ui/select";

export default function AuctionsPage() {
  const { session } = useAuth();
  const auctionsQuery = useAuctionsQuery();
  const lotsQuery = useLotsQuery();
  const productsQuery = useProductsQuery();
  const fishQuery = useFishCatalogQuery(session);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [sellerFilter, setSellerFilter] = useState("all");
  const [manualAuctionId, setManualAuctionId] = useState("");
  const [manualAmount, setManualAmount] = useState("");
  const [manualBidError, setManualBidError] = useState<string | null>(null);
  const [manualBidOk, setManualBidOk] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const deferredSearch = useDeferredValue(search);

  const productMap = useMemo(
    () => new Map((productsQuery.data?.data ?? []).map((product) => [product.id, product])),
    [productsQuery.data?.data],
  );
  const fishMap = useMemo(
    () => new Map((fishQuery.data?.data ?? []).map((fish) => [fish.id, fish])),
    [fishQuery.data?.data],
  );
  const lotMap = useMemo(
    () => new Map((lotsQuery.data?.data ?? []).map((lot) => [lot.id, lot])),
    [lotsQuery.data?.data],
  );
  const sellerOptions = useMemo(() => {
    const sellerIds = new Set<string>();
    for (const item of auctionsQuery.data?.data ?? []) {
      const lot = lotMap.get(item.lotId);
      const sellerId = item.sellerCompanyId ?? lot?.sellerCompanyId;
      if (sellerId) {
        sellerIds.add(sellerId);
      }
    }
    return Array.from(sellerIds).sort();
  }, [auctionsQuery.data?.data, lotMap]);

  const items = useMemo(() => {
    return (auctionsQuery.data?.data ?? []).filter((item) => {
      if (item.state === "DRAFT") {
        return false;
      }
      const lot = lotMap.get(item.lotId);
      const product = lot ? productMap.get(lot.productId) : undefined;
      const fish = product ? fishMap.get(product.fishId) : undefined;
      const searchMatch = `${item.id} ${item.lotId} ${item.state} ${item.sellerCompanyId ?? lot?.sellerCompanyId ?? ""} ${lot?.productLabel ?? ""} ${product?.fishName ?? ""} ${fish?.description ?? ""}`
        .toLowerCase()
        .includes(deferredSearch.toLowerCase());
      const statusMatch = status === "all" || item.state === status;
      const sellerMatch =
        sellerFilter === "all" ||
        (item.sellerCompanyId ?? lot?.sellerCompanyId ?? "") === sellerFilter;
      return searchMatch && statusMatch && sellerMatch;
    });
  }, [auctionsQuery.data?.data, deferredSearch, fishMap, lotMap, productMap, sellerFilter, status]);

  const manualBidMutation = useMutation({
    mutationFn: async () => {
      setManualBidError(null);
      setManualBidOk(null);
      if (!manualAuctionId.trim()) {
        throw new ApiError("Введите auction_id", 400, "MISSING_AUCTION_ID");
      }
      const amount = Number(manualAmount);
      if (!Number.isFinite(amount) || amount <= 0) {
        throw new ApiError("Введите корректную сумму ставки", 400, "INVALID_AMOUNT");
      }
      return placeBid({ auctionId: manualAuctionId.trim(), amount }, session);
    },
    onSuccess: (result) => {
      setManualBidOk(`Ставка отправлена: ${result.data.amount}`);
      void queryClient.invalidateQueries({ queryKey: ["auctions"] });
      void queryClient.invalidateQueries({ queryKey: ["auction", manualAuctionId.trim()] });
    },
    onError: (error) => {
      setManualBidError(error instanceof ApiError ? error.message : "Не удалось отправить ставку");
    },
  });

  return (
    <div className="page-stack">
      <div className="page-heading">
        <p className="eyebrow">Аукционы</p>
        <h1>Аукционы</h1>
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
          { label: "WON", value: "WON" },
          { label: "CANCELLED", value: "CANCELLED" },
        ]}
        source="all"
        onSourceChange={() => undefined}
        showSource={false}
        searchPlaceholder="Номер аукциона, лот, продукт или компания"
        extraFilters={
          <Field label="Продавец">
            <Select value={sellerFilter} onChange={(event) => setSellerFilter(event.target.value)}>
              <option value="all">Все</option>
              {sellerOptions.map((sellerId) => (
                <option key={sellerId} value={sellerId}>
                  {sellerId}
                </option>
              ))}
            </Select>
          </Field>
        }
      />

      <Card className="form-card">
        <div className="stack-md">
          <div>
            <p className="eyebrow">Fallback для текущего backend</p>
            <h2>Сделать ставку по auction_id</h2>
            <p className="muted">
              Используйте этот блок, если аукцион уже есть в БД, но отсутствует Trading read-model для списка/деталей.
            </p>
          </div>
          {manualBidError ? (
            <Notice tone="warning" title="Ставка не отправлена">
              {manualBidError}
            </Notice>
          ) : null}
          {manualBidOk ? (
            <Notice tone="success" title="Ставка отправлена">
              {manualBidOk}
            </Notice>
          ) : null}
          <div className="inline-form">
            <Field label="auction_id">
              <Input
                placeholder="например, 09678a5ff7438b8e18e3156ceef857ab"
                value={manualAuctionId}
                onChange={(event) => setManualAuctionId(event.target.value)}
              />
            </Field>
            <Field label="Сумма ставки">
              <Input
                type="number"
                min={1}
                value={manualAmount}
                onChange={(event) => setManualAmount(event.target.value)}
              />
            </Field>
          </div>
          <div className="inline-actions">
            <Button
              type="button"
              onClick={() => manualBidMutation.mutate()}
              disabled={manualBidMutation.isPending || !session?.companyId || !session?.userId}
            >
              {manualBidMutation.isPending ? "Отправляем..." : "Сделать ставку по ID"}
            </Button>
          </div>
        </div>
      </Card>

      {items.length ? (
        <div className="card-grid card-grid-2">
          {items.map((item) => {
            const lot = lotMap.get(item.lotId);
            const product = lot ? productMap.get(lot.productId) : undefined;

            return (
              <AuctionCard
                key={item.id}
                auction={item}
                fishName={product?.fishName}
                productLabel={lot?.productLabel}
                sellerCompanyId={item.sellerCompanyId ?? lot?.sellerCompanyId}
              />
            );
          })}
        </div>
      ) : (
        <EmptyState
          title="Аукционы не найдены"
          description="Создайте аукцион или снимите фильтры."
          actionHref="/create/auction"
          actionLabel="Создать аукцион"
        />
      )}
    </div>
  );
}
