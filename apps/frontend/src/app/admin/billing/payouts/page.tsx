"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useAuth } from "@/entities/session/model/auth-context";
import { AuthGuard } from "@/features/auth/ui/auth-guard";
import {
  adminConfirmDealInvoice,
  adminMarkPayoutFailed,
  adminMarkPayoutPaid,
  adminMarkPayoutReady,
  listAdminPayoutQueue,
  listAdminPendingDealInvoices,
  type AdminPayoutQueueRow,
} from "@/shared/api/billing-service";
import { ApiError } from "@/shared/api/http-client";
import { env } from "@/shared/config/env";
import { isAdminSession } from "@/shared/lib/access";
import { displayId } from "@/shared/lib/display";
import { formatDateTime, formatMoney } from "@/shared/lib/format";
import { payoutStatusLabel } from "@/entities/billing/lib/status-labels";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Notice } from "@/shared/ui/notice";
import { PageHeader } from "@/shared/ui/page-header";

function invoiceHint(row: AdminPayoutQueueRow): string {
  const inv = row.invoice_status;
  if (!inv) {
    return "Инвойс не найден в billing (аномалия).";
  }
  if (inv !== "PAID") {
    return `Инвойс ещё не PAID (сейчас: ${inv}) — выплата не должна продвигаться, пока покупатель не оплатил.`;
  }
  return "Инвойс оплачен — можно READY → PAID (зачисление на внутренний баланс).";
}

export default function AdminBillingPayoutsPage() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const enabled = Boolean(session) && env.enableBillingAdminUI && isAdminSession(session);

  const queueQuery = useQuery({
    queryKey: ["admin-billing-payout-queue"],
    queryFn: () => listAdminPayoutQueue(session!),
    enabled,
    staleTime: 10_000,
  });

  const pendingInvoicesQuery = useQuery({
    queryKey: ["admin-billing-pending-invoices"],
    queryFn: () => listAdminPendingDealInvoices(session!),
    enabled,
    staleTime: 10_000,
  });

  const readyMu = useMutation({
    mutationFn: (id: string) => adminMarkPayoutReady(id, session!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin-billing-payout-queue"] }),
  });
  const paidMu = useMutation({
    mutationFn: (id: string) => adminMarkPayoutPaid(id, session!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin-billing-payout-queue"] }),
  });
  const failedMu = useMutation({
    mutationFn: (id: string) => adminMarkPayoutFailed(id, session!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin-billing-payout-queue"] }),
  });
  const confirmInvMu = useMutation({
    mutationFn: (id: string) => adminConfirmDealInvoice(id, session!),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin-billing-pending-invoices"] });
      void queryClient.invalidateQueries({ queryKey: ["admin-billing-payout-queue"] });
    },
  });

  if (!env.enableBillingAdminUI) {
    return (
      <AuthGuard roles={["admin"]}>
        <div className="page-stack">
          <Notice tone="warning" title="Выключено">
            Задайте <code>NEXT_PUBLIC_ENABLE_BILLING_ADMIN=true</code> и включите{" "}
            <code>BILLING_ENABLE_ADMIN_ACTIONS</code> на billing.
          </Notice>
        </div>
      </AuthGuard>
    );
  }

  return (
    <AuthGuard roles={["admin"]}>
      <div className="page-stack">
        <PageHeader
          eyebrow="Admin · Billing"
          title="Очередь выплат продавцам"
          description="Операторский экран: кто, сколько, по какой сделке; статус инвойса; действия READY / PAID / FAILED."
        />

        <Notice tone="warning" title="Demo / внутренний баланс">
          Кнопка <strong>PAID</strong> не отправляет деньги в банк: она помечает выплату и{" "}
          <strong>зачисляет сумму на внутренний баланс продавца</strong> в billing (ledger). Это ожидаемо для
          fake-mode и песочницы.
        </Notice>

        <Card className="form-card">
          <h2 className="text-base font-medium">Инвойсы в ожидании оплаты</h2>
          <p className="muted text-sm">
            Полный экран со всеми колонками и ссылками на сделки:{" "}
            <Link className="underline" href="/admin/billing/invoices">
              /admin/billing/invoices
            </Link>
            . Ниже — краткий список для контекста рядом с выплатами.
          </p>
          {pendingInvoicesQuery.isLoading ? <p className="muted text-sm">Загрузка инвойсов…</p> : null}
          {pendingInvoicesQuery.error ? (
            <Notice tone="warning" title="Инвойсы">
              {pendingInvoicesQuery.error instanceof ApiError
                ? pendingInvoicesQuery.error.message
                : "Не удалось загрузить список."}
            </Notice>
          ) : null}
          {pendingInvoicesQuery.data?.invoices?.length ? (
            <div className="mt-3 overflow-x-auto">
              <table className="admin-data-table text-left text-sm">
                <thead>
                  <tr className="border-b border-white/10 text-xs uppercase muted">
                    <th>deal_id</th>
                    <th>Инвойс</th>
                    <th>Сумма</th>
                    <th>Создан</th>
                    <th>Действие</th>
                  </tr>
                </thead>
                <tbody>
                  {pendingInvoicesQuery.data.invoices.map((inv) => (
                    <tr key={inv.id} className="border-b border-white/5">
                      <td>
                        <code className="text-xs">{inv.deal_id}</code>
                      </td>
                      <td>
                        <code className="text-xs">{displayId(inv.id, "#")}</code>
                      </td>
                      <td>{formatMoney(inv.total_amount)}</td>
                      <td className="muted">{formatDateTime(inv.created_at)}</td>
                      <td>
                        <Button
                          type="button"
                          variant="secondary"
                          size="sm"
                          disabled={confirmInvMu.isPending}
                          onClick={() => confirmInvMu.mutate(inv.id)}
                        >
                          Подтвердить оплату (admin)
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : !pendingInvoicesQuery.isLoading ? (
            <p className="muted text-sm">Нет счетов в статусе PAYMENT_PENDING.</p>
          ) : null}
        </Card>

        <Card className="form-card">
          {queueQuery.isLoading ? <p className="muted">Загрузка…</p> : null}
          {queueQuery.error ? (
            <Notice tone="warning" title="Ошибка">
              {queueQuery.error instanceof ApiError ? queueQuery.error.message : "Не удалось загрузить очередь."}
            </Notice>
          ) : null}
          {queueQuery.data?.payouts?.length ? (
            <div className="overflow-x-auto">
              <table className="admin-data-table admin-payout-table text-left text-sm">
                <colgroup>
                  <col />
                  <col />
                  <col />
                  <col />
                  <col />
                  <col />
                  <col />
                  <col />
                  <col />
                </colgroup>
                <thead>
                  <tr className="border-b border-white/10 text-xs uppercase muted">
                    <th>Статус</th>
                    <th>Продавец</th>
                    <th>Покупатель</th>
                    <th>Сумма</th>
                    <th>Сделка</th>
                    <th>Инвойс</th>
                    <th>Аукцион</th>
                    <th>Создана</th>
                    <th>Действия</th>
                  </tr>
                </thead>
                <tbody>
                  {queueQuery.data.payouts.map((row) => (
                    <tr key={row.payout_id} className="border-b border-white/5 align-top">
                      <td>
                        <span className="font-medium">{payoutStatusLabel(row.status)}</span>
                        <p className="table-hint">{invoiceHint(row)}</p>
                      </td>
                      <td>
                        <div>{row.seller_company_name || displayId(row.seller_company_id, "")}</div>
                        <code className="text-xs muted">{row.seller_company_id}</code>
                      </td>
                      <td>
                        <div>{row.buyer_company_name || displayId(row.buyer_company_id, "")}</div>
                        <code className="text-xs muted">{row.buyer_company_id}</code>
                      </td>
                      <td>
                        {formatMoney(row.amount)} {row.currency}
                      </td>
                      <td>
                        <code className="text-xs">{row.deal_id}</code>
                      </td>
                      <td>
                        <div>{row.invoice_status || "—"}</div>
                        <code className="text-xs">{row.invoice_id}</code>
                      </td>
                      <td>
                        <code className="text-xs">{row.auction_id}</code>
                      </td>
                      <td className="text-xs">
                        {formatDateTime(row.created_at)}
                        {row.ready_at ? <div className="muted">ready: {formatDateTime(row.ready_at)}</div> : null}
                        {row.paid_at ? <div className="muted">paid: {formatDateTime(row.paid_at)}</div> : null}
                        {row.failed_at ? <div className="muted">failed: {formatDateTime(row.failed_at)}</div> : null}
                      </td>
                      <td>
                        <div className="stack-sm">
                          {row.status === "PENDING" ? (
                            <Button
                              type="button"
                              size="sm"
                              variant="secondary"
                              disabled={readyMu.isPending}
                              onClick={() => readyMu.mutate(row.payout_id)}
                            >
                              READY
                            </Button>
                          ) : null}
                          {row.status === "READY" ? (
                            <Button
                              type="button"
                              size="sm"
                              disabled={paidMu.isPending}
                              onClick={() => paidMu.mutate(row.payout_id)}
                            >
                              PAID (на баланс)
                            </Button>
                          ) : null}
                          {(row.status === "PENDING" || row.status === "READY") && (
                            <Button
                              type="button"
                              size="sm"
                              variant="ghost"
                              disabled={failedMu.isPending}
                              onClick={() => failedMu.mutate(row.payout_id)}
                            >
                              FAILED
                            </Button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : !queueQuery.isLoading && !queueQuery.error ? (
            <p className="muted">Записей выплат пока нет.</p>
          ) : null}
        </Card>
      </div>
    </AuthGuard>
  );
}
