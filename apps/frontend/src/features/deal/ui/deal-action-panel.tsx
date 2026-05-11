"use client";

import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { type DealActionInput, runDealAction } from "@/shared/api/deals-service";
import { getDealParticipantSide, type DealParticipantSide } from "@/shared/lib/access";
import { formatMoney, toDateTimeLocalValue } from "@/shared/lib/format";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { Notice } from "@/shared/ui/notice";
import { Textarea } from "@/shared/ui/textarea";
import type { DealConfirmationRecord, DealConfirmationStage, DealRecord } from "@/shared/types/domain";

function primaryActionTitle(status: DealRecord["status"], side: DealParticipantSide): string {
  if (side === "customer") {
    switch (status) {
      case "pending":
        return "Ожидаем подтверждение продавца";
      case "confirmed":
        return "Ожидаем контракт от продавца";
      case "contract_prepared":
        return "Подписать контракт";
      case "contract_signed":
        return "Ожидаем инвойс";
      case "payment_requested":
        return "Оплатить счёт (блок «Оплата» ниже)";
      case "paid":
        return "Ожидаем отгрузку";
      case "shipment_requested":
        return "Подтвердить отгрузку";
      case "shipped":
        return "Подтвердить получение";
      case "completed":
        return "Закупка завершена";
      case "cancelled":
        return "Сделка отменена";
    }
  }

  if (side === "supplier") {
    switch (status) {
      case "pending":
        return "Подтвердить победителя";
      case "confirmed":
        return "Подготовить контракт";
      case "contract_prepared":
        return "Ожидаем подпись покупателя";
      case "contract_signed":
        return "Запросить оплату";
      case "payment_requested":
        return "Ожидаем оплату покупателя";
      case "paid":
        return "Запросить отгрузку";
      case "shipment_requested":
        return "Подтвердить отгрузку";
      case "shipped":
        return "Закрыть поставку";
      case "completed":
        return "Продажа завершена";
      case "cancelled":
        return "Сделка отменена";
    }
  }

  switch (status) {
    case "pending":
      return "Ожидает подтверждение сделки";
    case "confirmed":
      return "Подготовить контракт";
    case "contract_prepared":
      return "Подписать контракт";
    case "contract_signed":
      return "Запросить оплату";
    case "payment_requested":
      return "Оплата через billing";
    case "paid":
      return "Запросить отгрузку";
    case "shipment_requested":
      return "Подтвердить отгрузку";
    case "shipped":
      return "Закрыть поставку";
    case "completed":
      return "Сделка завершена";
    case "cancelled":
      return "Сделка отменена";
  }
}

function sideIntro(side: DealParticipantSide): string {
  switch (side) {
    case "supplier":
      return "Вы продавец в этой сделке: доступны действия поставщика по контракту, оплате и отгрузке.";
    case "customer":
      return "Вы покупатель: подпись контракта, подтверждения этапов (кроме оплаты) и оплата счёта в блоке billing после «Запросить оплату» со стороны продавца.";
    case "outsider":
      return "Действия доступны только покупателю и продавцу этой сделки.";
  }
}

function confirmationStageLabel(stage: DealConfirmationStage): string {
  switch (stage) {
    case "confirmed":
      return "подтверждение сделки";
    case "paid":
      return "подтверждение оплаты";
    case "shipped":
      return "подтверждение отгрузки";
    case "completed":
      return "подтверждение завершения";
    case "cancelled":
      return "подтверждение отмены";
  }
}

function nextConfirmationRequestStage(
  deal: DealRecord,
  companyId: string | undefined,
): DealConfirmationStage | null {
  if (!companyId) {
    return null;
  }
  const isSupplier = companyId === deal.supplierId;
  const isCustomer = companyId === deal.customerId;

  switch (deal.status) {
    case "pending":
      return isSupplier ? "confirmed" : null;
    case "payment_requested":
      // Оплата и переход в paid только через billing (подтверждение этапа paid отключено на сервере).
      return null;
    case "shipment_requested":
      return isSupplier ? "shipped" : null;
    case "shipped":
      return isSupplier ? "completed" : null;
    default:
      return null;
  }
}

function waitingCounterpartyLabel(deal: DealRecord, companyId: string | undefined): string | null {
  if (!companyId) {
    return null;
  }
  const isSupplier = companyId === deal.supplierId;
  const isCustomer = companyId === deal.customerId;

  switch (deal.status) {
    case "pending":
      return isCustomer ? "Ожидается запрос подтверждения от продавца." : null;
    case "confirmed":
      return isCustomer ? "Ожидается подготовка контракта продавцом." : null;
    case "contract_prepared":
      return isSupplier ? "Ожидается подпись контракта покупателем." : null;
    case "contract_signed":
      return isCustomer ? "Ожидается инвойс от продавца." : null;
    case "payment_requested":
      return isSupplier
        ? "Ожидается оплата счёта покупателем (в блоке «Оплата (billing)» на этой странице, не через «подтверждение этапа»)."
        : isCustomer
          ? "Счёт выставил продавец. Оплатите в блоке «Оплата (billing)» ниже (в демо — «Fake pay», если включён fake-provider)."
          : null;
    case "paid":
      return isCustomer ? "Ожидается запрос отгрузки от продавца." : null;
    case "shipment_requested":
      return isCustomer ? "Ожидается запрос подтверждения отгрузки от продавца." : null;
    case "shipped":
      return isCustomer ? "Ожидается запрос подтверждения получения от продавца." : null;
    default:
      return null;
  }
}

export function DealActionPanel({
  deal,
  confirmations,
}: {
  deal: DealRecord;
  confirmations: DealConfirmationRecord[];
}) {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [contractNumber, setContractNumber] = useState(deal.contract?.number ?? `CNT-${deal.id.slice(-6).toUpperCase()}`);
  const [documentUrl, setDocumentUrl] = useState(deal.contract?.documentUrl ?? "");
  const [signatureRef, setSignatureRef] = useState(deal.contract?.signatureRef ?? `SIG-${deal.id.slice(-6).toUpperCase()}`);
  const [invoiceNumber, setInvoiceNumber] = useState(`INV-${deal.id.slice(-6).toUpperCase()}`);
  const [dueDate, setDueDate] = useState(toDateTimeLocalValue(new Date(Date.now() + 5 * 24 * 60 * 60 * 1000)));
  const [cancelReason, setCancelReason] = useState("Отмена по соглашению сторон");
  const [confirmationComment, setConfirmationComment] = useState("");
  const [rejectReason, setRejectReason] = useState("Нужна дополнительная проверка");

  const pendingConfirmation = useMemo(
    () => confirmations.find((item) => item.status === "pending") ?? null,
    [confirmations],
  );

  const mutation = useMutation({
    mutationFn: (input: DealActionInput) => {
      setError(null);
      return runDealAction(deal.id, input, session);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["deal", deal.id] });
      void queryClient.invalidateQueries({ queryKey: ["deals"] });
      void queryClient.invalidateQueries({ queryKey: ["auction", deal.auctionId] });
      void queryClient.invalidateQueries({ queryKey: ["deal-confirmations", deal.id] });
    },
    onError: (nextError) => {
      setError(nextError instanceof ApiError ? nextError.message : "Не удалось выполнить действие по сделке.");
    },
  });

  const currentCompanyId = session?.companyId;
  const side = getDealParticipantSide(deal, session);
  const isFinal = deal.status === "completed" || deal.status === "cancelled";
  const isSupplier = side === "supplier";
  const isCustomer = side === "customer";
  const canCancel = !isFinal;
  const nextStage = nextConfirmationRequestStage(deal, currentCompanyId);
  const waitingLabel = waitingCounterpartyLabel(deal, currentCompanyId);
  const canApprovePending =
    pendingConfirmation !== null && currentCompanyId === pendingConfirmation.counterpartyCompanyId;
  const requestedByCurrentUser =
    pendingConfirmation !== null && currentCompanyId === pendingConfirmation.requestedByCompanyId;

  const submit = (input: DealActionInput) => {
    mutation.mutate(input);
  };

  return (
    <div className="deal-actions stack-lg">
      <div>
        <p className="eyebrow">{isSupplier ? "Интерфейс продавца" : isCustomer ? "Интерфейс покупателя" : "Lifecycle"}</p>
        <h2>{primaryActionTitle(deal.status, side)}</h2>
        <p className="muted">{sideIntro(side)}</p>
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

      {pendingConfirmation ? (
        <Notice
          tone={canApprovePending ? "info" : "warning"}
          title={`Ожидает ${confirmationStageLabel(pendingConfirmation.stage)}`}
        >
          {requestedByCurrentUser
            ? `Запрос отправлен компанией ${pendingConfirmation.requestedByCompanyId}. Ожидается решение контрагента ${pendingConfirmation.counterpartyCompanyId}.`
            : `Запрос инициирован компанией ${pendingConfirmation.requestedByCompanyId}. Решение требуется от ${pendingConfirmation.counterpartyCompanyId}.`}
        </Notice>
      ) : null}

      {canApprovePending && pendingConfirmation ? (
        <div className="stack-md">
          <Button
            disabled={mutation.isPending || !session}
            onClick={() => submit({ type: "approveConfirmation", confirmationId: pendingConfirmation.id })}
            type="button"
          >
            Подтвердить этап
          </Button>
          <form
            className="stack-md"
            onSubmit={(event) => {
              event.preventDefault();
              submit({ type: "rejectConfirmation", confirmationId: pendingConfirmation.id, reason: rejectReason });
            }}
          >
            <Field label="Причина отклонения">
              <Textarea value={rejectReason} onChange={(event) => setRejectReason(event.target.value)} rows={3} />
            </Field>
            <Button disabled={mutation.isPending || !session} type="submit" variant="danger">
              Отклонить этап
            </Button>
          </form>
        </div>
      ) : null}

      {!pendingConfirmation && nextStage ? (
        <form
          className="stack-md"
          onSubmit={(event) => {
            event.preventDefault();
            submit({
              type: "requestConfirmation",
              stage: nextStage,
              verificationMethod: "manual",
              comment: confirmationComment,
            });
          }}
        >
          <Field label="Комментарий к подтверждению">
            <Textarea
              value={confirmationComment}
              onChange={(event) => setConfirmationComment(event.target.value)}
              placeholder="Кратко зафиксируйте, что именно подтверждается."
              rows={3}
            />
          </Field>
          <Button disabled={mutation.isPending || !session} type="submit">
            Запросить {confirmationStageLabel(nextStage)}
          </Button>
        </form>
      ) : null}

      {!pendingConfirmation && waitingLabel ? (
        <Notice tone="info" title="Ожидание контрагента">
          {waitingLabel}
        </Notice>
      ) : null}

      {deal.status === "confirmed" && isSupplier ? (
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

      {deal.status === "contract_prepared" && isCustomer ? (
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

      {deal.status === "contract_signed" && isSupplier ? (
        <form
          className="stack-md"
          onSubmit={(event) => {
            event.preventDefault();
            submit({ type: "requestPayment", invoiceNumber, dueDate });
          }}
        >
          <div className="readonly-amount">
            <span>Сумма к оплате</span>
            <strong>{formatMoney(deal.totalAmount)}</strong>
            <p>Цена сформирована по результатам торгов и не редактируется на этапе оплаты.</p>
          </div>
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

      {deal.status === "paid" && isSupplier ? (
        <Button disabled={mutation.isPending || !session} onClick={() => submit({ type: "requestShipment" })} type="button">
          Запросить отгрузку
        </Button>
      ) : null}

      {isFinal ? <p className="muted">Для финального статуса доступны только просмотр и аудит.</p> : null}

      <div className="side-action-grid">
        {canCancel && !pendingConfirmation && (isSupplier || isCustomer) ? (
          <form
            className="stack-md"
            onSubmit={(event) => {
              event.preventDefault();
              submit({
                type: "requestConfirmation",
                stage: "cancelled",
                verificationMethod: "manual",
                comment: cancelReason,
              });
            }}
          >
            <Field label="Причина отмены">
              <Textarea value={cancelReason} onChange={(event) => setCancelReason(event.target.value)} rows={3} />
            </Field>
            <Button disabled={mutation.isPending || !session || !cancelReason} type="submit" variant="danger">
              Запросить отмену сделки
            </Button>
          </form>
        ) : null}
      </div>
    </div>
  );
}
