"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { placeBid } from "@/shared/api/trading-service";
import { toDateTimeLocalValue } from "@/shared/lib/format";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";

const schema = z.object({
  amount: z.coerce.number().int().positive("Введите сумму ставки"),
  placedAt: z.string().min(1, "Укажите время ставки"),
});

type Values = z.infer<typeof schema>;

function normalizeCompanyId(value?: string | null): string {
  return (value ?? "").trim().toLowerCase();
}

export function PlaceBidForm({
  auctionId,
  existingAmounts = [],
  sellerCompanyId,
  currentLeaderCompanyId,
  existingBidderCompanyIds = [],
}: {
  auctionId: string;
  existingAmounts?: number[];
  sellerCompanyId?: string;
  currentLeaderCompanyId?: string;
  existingBidderCompanyIds?: string[];
}) {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const actorCompanyId = normalizeCompanyId(session?.companyId);
  const isOwnLot =
    actorCompanyId !== "" &&
    actorCompanyId === normalizeCompanyId(sellerCompanyId);
  const hasOwnBid = existingBidderCompanyIds.some(
    (companyId) => normalizeCompanyId(companyId) === actorCompanyId,
  );
  const isLeader = actorCompanyId !== "" && actorCompanyId === normalizeCompanyId(currentLeaderCompanyId);
  const isOwnBidAttempt = hasOwnBid || isLeader;

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      amount: 600000,
      placedAt: toDateTimeLocalValue(new Date()),
    },
  });

  const mutation = useMutation({
    mutationFn: (values: Values) =>
      placeBid(
        {
          auctionId,
          amount: values.amount,
          placedAt: new Date(values.placedAt).toISOString(),
          sellerCompanyId,
        },
        session,
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auction", auctionId] });
      void queryClient.invalidateQueries({ queryKey: ["auctions"] });
    },
    onError: (error) => {
      form.setError("amount", {
        type: "manual",
        message:
          error instanceof ApiError
            ? error.message
            : "Не удалось отправить ставку",
      });
    },
  });

  const onSubmit = form.handleSubmit((values) => {
    if (isOwnLot) {
      form.setError("amount", {
        type: "manual",
        message: "нельзя ставить ставки на свой товар",
      });
      return;
    }

    if (isOwnBidAttempt) {
      form.setError("amount", {
        type: "manual",
        message: "нельзя перебивать свою же ставку",
      });
      return;
    }

    const highestAmount = existingAmounts.length ? Math.max(...existingAmounts) : 0;

    if (existingAmounts.includes(values.amount)) {
      form.setError("amount", {
        type: "manual",
        message: "нельзя ставить ту же сумму",
      });
      return;
    }

    if (highestAmount > 0 && values.amount < highestAmount) {
      form.setError("amount", {
        type: "manual",
        message: "нельзя ставить меньшую сумму",
      });
      return;
    }

    mutation.mutate(values);
  });

  return (
    <div className="stack-md">
      {isOwnLot ? (
        <Notice tone="warning" title="Ставка недоступна">
          нельзя ставить ставки на свой товар
        </Notice>
      ) : null}

      {isOwnBidAttempt ? (
        <Notice tone="warning" title="Ставка недоступна">
          нельзя перебивать свою же ставку
        </Notice>
      ) : null}

      <form className="stack-md" onSubmit={onSubmit}>
        <Field label="Сумма ставки" error={form.formState.errors.amount?.message}>
          <Input disabled={isOwnLot || isOwnBidAttempt} type="number" {...form.register("amount")} />
        </Field>
        <Field label="Время" error={form.formState.errors.placedAt?.message}>
          <Input disabled={isOwnLot || isOwnBidAttempt} type="datetime-local" {...form.register("placedAt")} />
        </Field>
        <Button disabled={mutation.isPending || isOwnLot || isOwnBidAttempt} type="submit">
          {mutation.isPending ? "Отправляем..." : "Сделать ставку"}
        </Button>
      </form>
    </div>
  );
}
