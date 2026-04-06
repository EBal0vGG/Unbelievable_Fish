"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useLotsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { createAuction } from "@/shared/api/trading-service";
import { toDateTimeLocalValue } from "@/shared/lib/format";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Select } from "@/shared/ui/select";
import type { ServiceMeta } from "@/shared/types/domain";

const schema = z
  .object({
    lotId: z.string().min(1, "Выберите lotId"),
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
  const [meta, setMeta] = useState<ServiceMeta | null>(null);
  const [useManualLotId, setUseManualLotId] = useState(false);

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      lotId: "",
      startsAt: toDateTimeLocalValue(new Date(Date.now() + 30 * 60 * 1000)),
      endsAt: toDateTimeLocalValue(new Date(Date.now() + 3 * 60 * 60 * 1000)),
    },
  });

  const mutation = useMutation({
    mutationFn: (values: Values) =>
      createAuction(
        {
          lotId: values.lotId,
          startsAt: new Date(values.startsAt).toISOString(),
          endsAt: new Date(values.endsAt).toISOString(),
        },
        session,
      ),
    onSuccess: (result) => {
      setMeta(result.meta);
      void queryClient.invalidateQueries({ queryKey: ["auctions"] });
    },
  });

  return (
    <Card className="form-card">
      <div className="stack-lg">
        <div>
          <p className="eyebrow">Trading Command</p>
          <h1>Выставить аукцион</h1>
        </div>

        {meta?.note ? (
          <Notice tone={meta.source === "api" ? "success" : "warning"} title={`Источник: ${meta.source}`}>
            {meta.note}
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
                  Ввести lotId вручную
                </Button>
              </div>

              {useManualLotId ? (
                <Input placeholder="lot-123" {...form.register("lotId")} />
              ) : (
                <Select {...form.register("lotId")}>
                  <option value="">Выберите лот</option>
                  {(lotsQuery.data?.data ?? []).map((lot) => (
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
            <Button disabled={mutation.isPending} type="submit">
              {mutation.isPending ? "Отправляем..." : "Создать аукцион"}
            </Button>
          </div>
        </form>
      </div>
    </Card>
  );
}
