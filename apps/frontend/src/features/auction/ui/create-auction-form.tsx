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
import { isOwnedLot } from "@/shared/lib/access";
import { toDateTimeLocalValue } from "@/shared/lib/format";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
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

  const mutation = useMutation({
    mutationFn: (values: Values) => {
      setError(null);
      setIsCreated(false);
      if (!session) {
        throw new ApiError("Войдите в профиль, чтобы выставить аукцион", 400, "MISSING_COMPANY_ID");
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
      setError(error instanceof ApiError ? error.message : "Не удалось создать аукцион.");
    },
  });

  return (
    <Card className="form-card">
      <div className="stack-lg">
        <div>
          <p className="eyebrow">Аукционы</p>
          <h1>Выставить аукцион</h1>
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
        ) : null}

        {error ? (
          <Notice tone="warning" title="Ошибка создания аукциона">
            {error}
          </Notice>
        ) : null}

        <form className="stack-md" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
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
                <Input placeholder="lot-123" {...form.register("lotId")} />
              ) : (
                <Select {...form.register("lotId")}>
                  <option value="">Выберите лот</option>
                  {availableLots.map((lot) => (
                    <option key={lot.id} value={lot.id}>
                      {lot.productLabel} · {lot.id}
                    </option>
                  ))}
                </Select>
              )}
            </div>
          </Field>
          <Field label="Старт" error={form.formState.errors.startsAt?.message}>
            <Input type="datetime-local" {...form.register("startsAt")} />
          </Field>
          <Field label="Завершение" error={form.formState.errors.endsAt?.message}>
            <Input type="datetime-local" {...form.register("endsAt")} />
          </Field>
          <div className="inline-actions">
            <Button disabled={mutation.isPending || !session?.companyId || !session.userId} type="submit">
              {mutation.isPending ? "Отправляем..." : "Создать аукцион"}
            </Button>
          </div>
        </form>
      </div>
    </Card>
  );
}
