"use client";

import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { dealShowsBillingPanel } from "@/entities/billing/lib/deal-billing";
import { invoiceStatusLabel, payoutStatusLabel } from "@/entities/billing/lib/status-labels";
import { useBillingBalanceQuery, useDealInvoiceBillingQuery, useSellerPayoutsQuery } from "@/entities/billing/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { ApiError } from "@/shared/api/http-client";
import { fakeConfirmDealInvoice } from "@/shared/api/billing-service";
import { env, isFakeBillingUiAllowed } from "@/shared/config/env";
import { getDealParticipantSide, isAdminSession } from "@/shared/lib/access";
import { formatDateTime, formatMoney } from "@/shared/lib/format";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Notice } from "@/shared/ui/notice";
import type { DealRecord } from "@/shared/types/domain";

type Props = { deal: DealRecord };

export function DealBillingPanel({ deal }: Props) {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const side = session ? getDealParticipantSide(deal, session) : "outsider";
  const billingVisible = dealShowsBillingPanel(deal);
  const invoiceQuery = useDealInvoiceBillingQuery(deal.id, session, billingVisible && side !== "outsider");
  const pollPayouts =
    billingVisible &&
    side === "supplier" &&
    (deal.status === "paid" || deal.status === "shipment_requested" || deal.status === "shipped" || deal.status === "completed");
  const payoutsQuery = useSellerPayoutsQuery(session, billingVisible && side === "supplier", {
    dealId: deal.id,
    pollUntilPaidForDeal: pollPayouts,
  });
  const balanceQuery = useBillingBalanceQuery(session);

  const payoutForDeal = payoutsQuery.data?.payouts.find((p) => p.deal_id === deal.id) ?? null;

  const fakePay = useMutation({
    mutationFn: async () => {
      if (!session || !invoiceQuery.data) {
        return;
      }
      await fakeConfirmDealInvoice(invoiceQuery.data.id, session);
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["deal", deal.id] }),
        queryClient.invalidateQueries({ queryKey: ["billing-invoice", deal.id] }),
        queryClient.invalidateQueries({ queryKey: ["billing-payouts"] }),
        queryClient.invalidateQueries({ queryKey: ["billing-balance"] }),
      ]);
    },
  });

  if (!session || side === "outsider" || !billingVisible) {
    return null;
  }

  const inv = invoiceQuery.data;
  const invInitialLoading = invoiceQuery.isLoading;
  const invBackgroundFetch = invoiceQuery.isFetching && !invoiceQuery.isLoading;
  const invErr = invoiceQuery.error;
  const invoiceNotReady =
    invErr instanceof ApiError && invErr.status === 404 && side === "customer";
  const fakeConfirmEnabled = isFakeBillingUiAllowed(balanceQuery.data?.deal_invoice_fake_confirm_enabled);
  const fakeUiHardOff = process.env.NEXT_PUBLIC_ENABLE_FAKE_BILLING === "false";
  const serverSaysNoFake =
    balanceQuery.isSuccess && balanceQuery.data?.deal_invoice_fake_confirm_enabled !== true;

  return (
    <Card className="form-card">
      <div className="stack-md">
        <div>
          <p className="eyebrow">Оплата (billing)</p>
          <h2>Инвойс и выплата</h2>
          <p className="muted">
            Инвойс в системе создаётся не кнопкой покупателя: после подписания контракта продавец нажимает «Запросить оплату» — тогда
            здесь появится счёт. Демо: при fake-provider доступна кнопка подтверждения оплаты (флаг с баланса).
          </p>
          {session && isAdminSession(session) && env.enableBillingAdminUI ? (
            <p className="muted">
              Админ:{" "}
              <Link className="underline" href="/admin/billing/invoices">
                счета в ожидании
              </Link>
              ,{" "}
              <Link className="underline" href="/admin/billing/payouts">
                выплаты
              </Link>
              .
            </p>
          ) : null}
        </div>

        {side === "customer" ? (
          <section aria-label="Счёт покупателя" className="stack-md billing-section">
            <div className="billing-section-body" style={{ minHeight: "4.5rem" }}>
              {invInitialLoading ? <p className="muted">Загрузка счёта…</p> : null}
              {invBackgroundFetch ? <p className="muted text-xs">Обновление данных…</p> : null}
              {!invInitialLoading && invoiceNotReady ? (
                <Notice tone="info" title="Счёт создаётся">
                  Инвойс в billing ещё не появился (задержка relay или сделка не дошла до выставления счёта). Страница
                  обновит данные автоматически.
                </Notice>
              ) : null}
              {!invInitialLoading && invErr && !invoiceNotReady ? (
                <Notice tone="warning" title="Инвойс недоступен">
                  {invErr instanceof ApiError ? invErr.message : "Не удалось загрузить счёт из billing."}
                </Notice>
              ) : null}
              {!invInitialLoading && inv ? (
                <div className="stack-sm">
                  <p>
                    <strong>Статус:</strong> {invoiceStatusLabel(inv.status)} · <strong>Сумма:</strong>{" "}
                    {formatMoney(inv.total_amount)} <span className="muted">{inv.currency}</span>
                  </p>
                  <p className="muted">
                    Товары: {formatMoney(inv.goods_amount)} · Комиссия: {formatMoney(inv.platform_fee_due_amount)}
                  </p>
                  <p className="muted">Оплатить до: {formatDateTime(inv.due_at)}</p>
                  {inv.payment_url ? (
                    <p className="muted">
                      Ссылка провайдера: <code className="text-xs">{inv.payment_url}</code>
                    </p>
                  ) : null}
                  {deal.status === "payment_requested" && inv.status === "PAYMENT_PENDING" && fakeConfirmEnabled ? (
                    <button
                      className={buttonStyles({ variant: "primary" })}
                      disabled={fakePay.isPending}
                      onClick={() => fakePay.mutate()}
                      type="button"
                    >
                      {fakePay.isPending ? "Отправка…" : "Fake pay (демо)"}
                    </button>
                  ) : null}
                  {deal.status === "payment_requested" && inv.status === "PAYMENT_PENDING" && balanceQuery.isSuccess && !fakeConfirmEnabled ? (
                    <p className="muted">
                      {fakeUiHardOff
                        ? "Кнопка демо-оплаты отключена во фронте (NEXT_PUBLIC_ENABLE_FAKE_BILLING=false)."
                        : serverSaysNoFake
                          ? "Billing не сообщает fake-confirm: проверьте BILLING_ENABLE_FAKE_PROVIDER на сервисе billing и перезапуск."
                          : "Оплата через демо-кнопку недоступна."}
                    </p>
                  ) : null}
                  {fakePay.error ? (
                    <Notice tone="warning" title="Не удалось подтвердить оплату">
                      {fakePay.error instanceof ApiError && fakePay.error.status === 404
                        ? "Запрос отклонён: на billing выключен fake-provider или нет прав (ожидается BILLING_ENABLE_FAKE_PROVIDER и покупатель той же компании)."
                        : fakePay.error instanceof ApiError
                          ? fakePay.error.message
                          : String(fakePay.error)}
                    </Notice>
                  ) : null}
                </div>
              ) : null}
            </div>
          </section>
        ) : null}

        {side === "supplier" ? (
          <section aria-label="Выплата продавцу" className="stack-md billing-section">
            <div className="billing-section-body" style={{ minHeight: "4.5rem" }}>
              {payoutsQuery.isLoading ? <p className="muted">Загрузка выплат…</p> : null}
              {payoutsQuery.isFetching && !payoutsQuery.isLoading ? <p className="muted text-xs">Обновление выплат…</p> : null}
              {payoutsQuery.error ? (
                <Notice tone="warning" title="Выплаты">
                  {payoutsQuery.error instanceof ApiError ? payoutsQuery.error.message : "Ошибка загрузки выплат."}
                </Notice>
              ) : null}
              {!payoutsQuery.isLoading && payoutForDeal ? (
                <div className="stack-sm">
                  <p>
                    <strong>Выплата по сделке:</strong> {formatMoney(payoutForDeal.amount)}{" "}
                    <span className="muted">{payoutForDeal.currency}</span>
                  </p>
                  <p>
                    <strong>Статус:</strong> {payoutStatusLabel(payoutForDeal.status)}
                  </p>
                  {payoutForDeal.ready_at ? (
                    <p className="muted">Ready: {formatDateTime(payoutForDeal.ready_at)}</p>
                  ) : null}
                  {payoutForDeal.paid_at ? (
                    <p className="muted">Paid: {formatDateTime(payoutForDeal.paid_at)}</p>
                  ) : null}
                  {payoutForDeal.status === "PAID" ? (
                    <div className="stack-sm">
                      <Notice tone="success" title="Средства зачислены">
                        Внутренний баланс продавца обновлён. Текущий available:{" "}
                        {balanceQuery.isLoading ? (
                          "…"
                        ) : balanceQuery.data ? (
                          formatMoney(balanceQuery.data.available)
                        ) : (
                          "н/д"
                        )}{" "}
                        {balanceQuery.data?.currency}
                      </Notice>
                      <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href="/me">
                        Открыть профиль / баланс
                      </Link>
                    </div>
                  ) : (
                    <p className="muted">
                      После оплаты покупателя выплата появится в статусе «В очереди», затем «Готова к выплате» и «Выплачена»
                      (оператор / admin).
                    </p>
                  )}
                </div>
              ) : null}
              {!payoutsQuery.isLoading && !payoutForDeal && !payoutsQuery.error ? (
                <p className="muted">Запись выплаты появится после оплаты инвойса покупателем.</p>
              ) : null}
            </div>
          </section>
        ) : null}
      </div>
    </Card>
  );
}
