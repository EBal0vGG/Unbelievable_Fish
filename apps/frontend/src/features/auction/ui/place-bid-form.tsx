"use client";

import { useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";

import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { placeBid } from "@/shared/api/trading-service";
import { isBuyerSession } from "@/shared/lib/access";
import { getBidAccessError, getBidValidationError, getMinAllowedBid } from "@/shared/lib/trading-domain";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import type { BidRecord } from "@/shared/types/domain";

const schema = z.object({
  amount: z.coerce.number().int().positive("Введите сумму ставки"),
});

type Values = z.infer<typeof schema>;

export function PlaceBidForm({
  auctionId,
  auctionState,
  startsAt,
  endsAt,
  currentPrice = 0,
  minBidStep = 1,
  sellerCompanyId,
  leaderCompanyId,
  bids = [],
}: {
  auctionId: string;
  auctionState: "DRAFT" | "PUBLISHED" | "CLOSED" | "WON" | "CANCELLED";
  startsAt: string;
  endsAt: string;
  currentPrice?: number;
  minBidStep?: number;
  sellerCompanyId?: string;
  leaderCompanyId?: string;
  bids?: BidRecord[];
}) {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const canBid = isBuyerSession(session);
  const sessionError =
    !session?.companyId ? "Войдите в профиль, чтобы делать ставки" : !session.userId ? "Войдите в профиль, чтобы делать ставки" : null;
  const roleError = session && !canBid ? "Ставки доступны только покупателям" : null;
  const bidAccessError = getBidAccessError({
    actorCompanyId: session?.companyId,
    sellerCompanyId,
    leaderCompanyId,
    bids,
  });

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      amount: Math.max(currentPrice, 1),
    },
  });
  const minAllowedBid = getMinAllowedBid(
    {
      currentPrice,
      finalPrice: undefined,
      minBidStep,
      leaderCompanyId,
    },
    bids,
  );
  const bidValidationError = getBidValidationError(
    {
      state: auctionState,
      startsAt,
      endsAt,
      currentPrice,
      finalPrice: undefined,
      minBidStep,
      leaderCompanyId,
    },
    form.watch("amount") || 0,
    new Date(),
    bids,
  );
  const blockingError = sessionError ?? bidAccessError ?? bidValidationError;
  const hardBlockingError = sessionError ?? roleError ?? bidAccessError;

  useEffect(() => {
    form.setValue("amount", Math.max(minAllowedBid, 1));
  }, [form, minAllowedBid]);

  const mutation = useMutation({
    mutationFn: (values: Values) =>
      placeBid(
        {
          auctionId,
          amount: values.amount,
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
    const validationError = getBidValidationError(
      {
        state: auctionState,
        startsAt,
        endsAt,
        currentPrice,
        finalPrice: undefined,
      },
      values.amount,
      new Date(),
      bids,
    );
    const accessError = getBidAccessError({
      actorCompanyId: session?.companyId,
      sellerCompanyId,
      leaderCompanyId,
      bids,
    });

    if (roleError || accessError || validationError) {
      form.setError("amount", {
        type: "manual",
        message: roleError ?? accessError ?? validationError ?? "Не удалось отправить ставку",
      });
      return;
    }

    mutation.mutate(values);
  });

  return (
    <div className="stack-md">
      {hardBlockingError && form.formState.isSubmitted === false ? (
        <Notice tone="warning" title="Ставка недоступна">
          {hardBlockingError}
        </Notice>
      ) : null}

      <form className="stack-md" onSubmit={onSubmit}>
        <Field label="Сумма ставки" error={form.formState.errors.amount?.message}>
          <Input disabled={Boolean(hardBlockingError)} type="number" {...form.register("amount")} />
        </Field>
        <p className="muted">Минимально допустимая ставка: {minAllowedBid}</p>
        <Button disabled={mutation.isPending || Boolean(hardBlockingError)} type="submit">
          {mutation.isPending ? "Отправляем..." : "Сделать ставку"}
        </Button>
      </form>
    </div>
  );
}
