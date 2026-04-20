"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useLotsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { createAuction } from "@/shared/api/trading-service";
import { isOwnedLot, isSellerSession } from "@/shared/lib/access";
import { formatDateTime, formatMoney, toDateTimeLocalValue } from "@/shared/lib/format";
import { Button } from "@/shared/ui/button";
import { EntityPhoto } from "@/shared/ui/entity-photo";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Select } from "@/shared/ui/select";

const schema = z
  .object({
    lotId: z.string().min(1, "Выберите лот"),
    startsAt: z.string().min(1, "Укажите старт"),
    endsAt: z.string().min(1, "Укажите завершение"),
  })
  .refine((value) => new Date(value.endsAt) > new Date(value.startsAt), {
    message: "Время завершения должно быть позже старта",
    path: ["endsAt"],
  });

type Values = z.infer<typeof schema>;

export function CreateAuctionForm() {
  const { session } = useAuth();
  const canCreateAuction = isSellerSession(session);
  const queryClient = useQueryClient();
  const lotsQuery = useLotsQuery();
  const [isCreated, setIsCreated] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [useManualLotId, setUseManualLotId] = useState(false);
  const availableLots = (lotsQuery.data?.data ?? []).filter(
    (lot) => isOwnedLot(lot, session) && lot.status === "PUBLISHED" && !lot.auctionId,
  );

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      lotId: "",
      startsAt: toDateTimeLocalValue(new Date(Date.now() + 30 * 60 * 1000)),
      endsAt: toDateTimeLocalValue(new Date(Date.now() + 3 * 60 * 60 * 1000)),
    },
  });
  const selectedLotId = form.watch("lotId");
  const selectedLot = availableLots.find((lot) => lot.id === selectedLotId);

  const mutation = useMutation({
    mutationFn: (values: Values) => {
      setError(null);
      setIsCreated(false);
      if (!session) {
        throw new ApiError("Войдите в профиль, чтобы выставить аукцион", 400, "MISSING_COMPANY_ID");
      }
      if (!canCreateAuction) {
        throw new ApiError("Выставление аукциона доступно только компании-поставщику", 403, "SELLER_ONLY");
      }
      return createAuction(
        {
          lotId: values.lotId,
          startsAt: new Date(values.startsAt).toISOString(),
          endsAt: new Date(values.endsAt).toISOString(),
        },
        session,
      );
    },
    onSuccess: () => {
      setIsCreated(true);
      void queryClient.invalidateQueries({ queryKey: ["auctions"] });
      void queryClient.invalidateQueries({ queryKey: ["lots"] });
    },
    onError: (error) => {
      setError(error instanceof ApiError ? error.message : "Не удалось выставить аукцион.");
    },
  });

  return (
    <section className="auction-panel">
      <div className="stack-lg">
        <div>
          <p className="eyebrow">Торги</p>
          <h1>Выставить аукцион</h1>
          <p className="muted">Выберите опубликованный лот, проверьте фото партии и задайте окно приема ставок.</p>
        </div>

        {isCreated ? (
          <Notice tone="success" title="Аукцион выставлен">
            Новый аукцион появится в общем списке.
          </Notice>
        ) : null}

        {!session ? (
          <Notice tone="warning" title="Нужен вход">
            Войдите в профиль, чтобы выставить аукцион.
          </Notice>
        ) : !canCreateAuction ? (
          <Notice tone="warning" title="Доступно продавцам">
            Выставление аукциона доступно только компании-поставщику.
          </Notice>
        ) : null}

        {error ? (
          <Notice tone="warning" title="Не удалось выставить аукцион">
            {error}
          </Notice>
        ) : null}

        <form className="auction-choice" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
          <div className="auction-lot-preview stack-md">
            <EntityPhoto src={selectedLot?.photo} alt={selectedLot?.productLabel ?? "Фото лота"} />
            <div>
              <p className="eyebrow">Выбранный лот</p>
              <h2>{selectedLot?.productLabel ?? "Лот не выбран"}</h2>
              <p className="muted">
                {selectedLot
                  ? `${formatMoney(selectedLot.startPrice)} · старт ${formatDateTime(selectedLot.auctionStartsAt)}`
                  : "Выберите опубликованный лот из списка."}
              </p>
            </div>
          </div>

          <div className="stack-md">
            <Field label="Лот" error={form.formState.errors.lotId?.message}>
              <div className="stack-md">
                <div className="inline-actions inline-actions-start">
                  <Button
                    onClick={() => setUseManualLotId(false)}
                    size="sm"
                    type="button"
                    variant={useManualLotId ? "ghost" : "secondary"}
                  >
                    Выбрать из списка
                  </Button>
                  <Button
                    onClick={() => setUseManualLotId(true)}
                    size="sm"
                    type="button"
                    variant={useManualLotId ? "secondary" : "ghost"}
                  >
                    Ввести номер вручную
                  </Button>
                </div>

                {useManualLotId ? (
                  <Input placeholder="Номер лота" {...form.register("lotId")} />
                ) : (
                  <Select {...form.register("lotId")}>
                    <option value="">Выберите лот</option>
                    {availableLots.map((lot) => (
                      <option key={lot.id} value={lot.id}>
                        {lot.productLabel} · {formatMoney(lot.startPrice)}
                      </option>
                    ))}
                  </Select>
                )}
              </div>
            </Field>
            <div className="form-grid-2">
              <Field label="Старт" error={form.formState.errors.startsAt?.message}>
                <Input type="datetime-local" {...form.register("startsAt")} />
              </Field>
              <Field label="Завершение" error={form.formState.errors.endsAt?.message}>
                <Input type="datetime-local" {...form.register("endsAt")} />
              </Field>
            </div>
            <div className="inline-actions">
              <Button
                disabled={mutation.isPending || !session?.companyId || !session.userId || !canCreateAuction}
                type="submit"
              >
                {mutation.isPending ? "Выставляем..." : "Выставить аукцион"}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </section>
  );
}
