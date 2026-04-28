"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { type DealActionInput, runDealAction } from "@/shared/api/deals-service";
import { toDateTimeLocalValue } from "@/shared/lib/format";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import type { DealRecord } from "@/shared/types/domain";

function primaryActionTitle(status: DealRecord["status"]): string {
  switch (status) {
    case "pending":
      return "Подтвердить сделку";
    case "confirmed":
      return "Подготовить контракт";
    case "contract_prepared":
      return "Подписать контракт";
    case "contract_signed":
      return "Запросить оплату";
    case "payment_requested":
      return "Отметить оплату";
    case "paid":
      return "Запросить отгрузку";
    case "shipment_requested":
      return "Отметить отгрузку";
    case "shipped":
      return "Завершить сделку";
    case "completed":
      return "Сделка завершена";
    case "cancelled":
      return "Сделка отменена";
  }
}

export function DealActionPanel({ deal }: { deal: DealRecord }) {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [contractNumber, setContractNumber] = useState(deal.contract?.number ?? `CNT-${deal.id.slice(-6).toUpperCase()}`);
  const [documentUrl, setDocumentUrl] = useState(deal.contract?.documentUrl ?? "");
  const [signatureRef, setSignatureRef] = useState(deal.contract?.signatureRef ?? `SIG-${deal.id.slice(-6).toUpperCase()}`);
  const [invoiceNumber, setInvoiceNumber] = useState(`INV-${deal.id.slice(-6).toUpperCase()}`);
  const [dueDate, setDueDate] = useState(toDateTimeLocalValue(new Date(Date.now() + 5 * 24 * 60 * 60 * 1000)));
  const [paymentId, setPaymentId] = useState(`PAY-${deal.id.slice(-6).toUpperCase()}`);
  const [paymentType, setPaymentType] = useState("bank_transfer");
  const [trackingNumber, setTrackingNumber] = useState(`TRK-${deal.id.slice(-6).toUpperCase()}`);
  const [carrier, setCarrier] = useState("Рефрижераторная линия");
  const [newPrice, setNewPrice] = useState(String(deal.unitPrice));
  const [cancelReason, setCancelReason] = useState("Отмена по соглашению сторон");

  const mutation = useMutation({
    mutationFn: (input: DealActionInput) => {
      setError(null);
      return runDealAction(deal.id, input, session);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["deal", deal.id] });
      void queryClient.invalidateQueries({ queryKey: ["deals"] });
      void queryClient.invalidateQueries({ queryKey: ["auction", deal.auctionId] });
    },
    onError: (error) => {
      setError(error instanceof ApiError ? error.message : "Не удалось выполнить действие по сделке.");
    },
  });

  const isFinal = deal.status === "completed" || deal.status === "cancelled";
  const canUpdatePrice = deal.status === "pending";
  const canCancel = !isFinal;

  const submit = (input: DealActionInput) => {
    mutation.mutate(input);
  };

  return (
    <div className="deal-actions stack-lg">
      <div>
        <p className="eyebrow">Lifecycle</p>
        <h2>{primaryActionTitle(deal.status)}</h2>
        <p className="muted">Операционный контур фиксирует подтверждение, контракт, оплату и отгрузку.</p>
      </div>

      {error ? (
        <Notice tone="warning" title="Действие не выполнено">
          {error}
        </Notice>
      ) : null}

      {!session ? (
        <Notice tone="warning" title="Нужен вход">
          Войдите в профиль, чтобы управлять сделками.
        </Notice>
      ) : null}

      {deal.status === "pending" ? (
        <Button disabled={mutation.isPending || !session} onClick={() => submit({ type: "confirm" })} type="button">
          Подтвердить
        </Button>
      ) : null}

      {deal.status === "pending" || deal.status === "confirmed" ? (
        <form
          className="stack-md"
          onSubmit={(event) => {
            event.preventDefault();
            submit({ type: "prepareContract", contractNumber, documentUrl });
          }}
        >
          <Field label="Номер контракта">
            <Input value={contractNumber} onChange={(event) => setContractNumber(event.target.value)} />
          </Field>
          <Field label="Документ URL">
            <Input value={documentUrl} onChange={(event) => setDocumentUrl(event.target.value)} placeholder="https://..." />
          </Field>
          <Button disabled={mutation.isPending || !session} type="submit">
            Подготовить контракт
          </Button>
        </form>
      ) : null}

      {deal.status === "contract_prepared" ? (
        <form
          className="stack-md"
          onSubmit={(event) => {
            event.preventDefault();
            submit({ type: "signContract", signatureRef });
          }}
        >
          <Field label="Signature ref">
            <Input value={signatureRef} onChange={(event) => setSignatureRef(event.target.value)} />
          </Field>
          <Button disabled={mutation.isPending || !session || !signatureRef} type="submit">
            Подписать контракт
          </Button>
        </form>
      ) : null}

      {deal.status === "contract_signed" ? (
        <form
          className="stack-md"
          onSubmit={(event) => {
            event.preventDefault();
            submit({ type: "requestPayment", invoiceNumber, dueDate });
          }}
        >
          <Field label="Номер инвойса">
            <Input value={invoiceNumber} onChange={(event) => setInvoiceNumber(event.target.value)} />
          </Field>
          <Field label="Срок оплаты">
            <Input type="datetime-local" value={dueDate} onChange={(event) => setDueDate(event.target.value)} />
          </Field>
          <Button disabled={mutation.isPending || !session || !invoiceNumber} type="submit">
            Запросить оплату
          </Button>
        </form>
      ) : null}

      {deal.status === "payment_requested" ? (
        <form
          className="stack-md"
          onSubmit={(event) => {
            event.preventDefault();
            submit({ type: "markPaid", paymentId, paymentType });
          }}
        >
          <Field label="Payment ID">
            <Input value={paymentId} onChange={(event) => setPaymentId(event.target.value)} />
          </Field>
          <Field label="Тип платежа">
            <Input value={paymentType} onChange={(event) => setPaymentType(event.target.value)} />
          </Field>
          <Button disabled={mutation.isPending || !session || !paymentId || !paymentType} type="submit">
            Отметить оплату
          </Button>
        </form>
      ) : null}

      {deal.status === "paid" ? (
        <Button disabled={mutation.isPending || !session} onClick={() => submit({ type: "requestShipment" })} type="button">
          Запросить отгрузку
        </Button>
      ) : null}

      {deal.status === "shipment_requested" ? (
        <form
          className="stack-md"
          onSubmit={(event) => {
            event.preventDefault();
            submit({ type: "markShipped", trackingNumber, carrier });
          }}
        >
          <Field label="Трек-номер">
            <Input value={trackingNumber} onChange={(event) => setTrackingNumber(event.target.value)} />
          </Field>
          <Field label="Перевозчик">
            <Input value={carrier} onChange={(event) => setCarrier(event.target.value)} />
          </Field>
          <Button disabled={mutation.isPending || !session || !trackingNumber || !carrier} type="submit">
            Отметить отгрузку
          </Button>
        </form>
      ) : null}

      {deal.status === "shipped" ? (
        <Button disabled={mutation.isPending || !session} onClick={() => submit({ type: "complete" })} type="button">
          Завершить
        </Button>
      ) : null}

      {isFinal ? <p className="muted">Для финального статуса доступны только просмотр и аудит.</p> : null}

      <div className="side-action-grid">
        {canUpdatePrice ? (
          <form
            className="stack-md"
            onSubmit={(event) => {
              event.preventDefault();
              submit({ type: "updatePrice", newPrice: Number(newPrice) });
            }}
          >
            <Field label="Новая цена">
              <Input type="number" value={newPrice} onChange={(event) => setNewPrice(event.target.value)} />
            </Field>
            <Button disabled={mutation.isPending || !session || Number(newPrice) <= 0} type="submit" variant="secondary">
              Обновить цену
            </Button>
          </form>
        ) : null}

        {canCancel ? (
          <form
            className="stack-md"
            onSubmit={(event) => {
              event.preventDefault();
              submit({ type: "cancel", reason: cancelReason });
            }}
          >
            <Field label="Причина отмены">
              <Input value={cancelReason} onChange={(event) => setCancelReason(event.target.value)} />
            </Field>
            <Button disabled={mutation.isPending || !session || !cancelReason} type="submit" variant="danger">
              Отменить сделку
            </Button>
          </form>
        ) : null}
      </div>
    </div>
  );
}
