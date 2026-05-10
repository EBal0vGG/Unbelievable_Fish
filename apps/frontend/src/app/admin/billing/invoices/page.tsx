"use client";

import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { useAuth } from "@/entities/session/model/auth-context";
import { AuthGuard } from "@/features/auth/ui/auth-guard";
import { adminConfirmDealInvoice, listAdminPendingDealInvoices } from "@/shared/api/billing-service";
import { ApiError } from "@/shared/api/http-client";
import { env } from "@/shared/config/env";
import { isAdminSession } from "@/shared/lib/access";
import { displayId } from "@/shared/lib/display";
import { formatDateTime, formatMoney } from "@/shared/lib/format";
import { invoiceStatusLabel } from "@/entities/billing/lib/status-labels";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Notice } from "@/shared/ui/notice";
import { PageHeader } from "@/shared/ui/page-header";

export default function AdminBillingInvoicesPage() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const isAdmin = Boolean(session && isAdminSession(session));
  const enabled = Boolean(session) && env.enableBillingAdminUI && isAdmin;

  const pendingQuery = useQuery({
    queryKey: ["admin-billing-pending-invoices"],
    queryFn: () => listAdminPendingDealInvoices(session!),
    enabled,
    staleTime: 5_000,
    refetchInterval: 15_000,
  });

  const confirmMu = useMutation({
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
            Задайте <code>NEXT_PUBLIC_ENABLE_BILLING_ADMIN=true</code> и <code>BILLING_ENABLE_ADMIN_ACTIONS</code> на billing.
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
          title="Счета сделок в ожидании оплаты"
          description="Все инвойсы в статусе PAYMENT_PENDING. Подтверждение здесь эквивалентно успешной оплате (как fake-confirm для покупателя)."
        />

        <Notice tone="info" title="Доступ">
          Запросы идут в <code className="text-xs">GET /billing/admin/invoices/pending</code> с JWT администратора. Если список
          пуст при ожидаемых счетах, проверьте роль в сессии и логи billing.
        </Notice>

        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="secondary" size="sm" disabled={pendingQuery.isFetching} onClick={() => pendingQuery.refetch()}>
            Обновить список
          </Button>
          <Link className="inline-flex items-center text-sm underline" href="/admin/billing/payouts">
            К очереди выплат
          </Link>
        </div>

        <Card className="form-card">
          {pendingQuery.isLoading ? <p className="muted text-sm">Загрузка…</p> : null}
          {pendingQuery.error ? (
            <Notice tone="warning" title="Не удалось загрузить">
              {pendingQuery.error instanceof ApiError ? (
                <>
                  HTTP {pendingQuery.error.status}: {pendingQuery.error.message}
                  {pendingQuery.error.status === 403 ? (
                    <span className="block pt-2">
                      Нужен вход под компанией с ролью admin и токеном, который billing принимает как администратора.
                    </span>
                  ) : null}
                </>
              ) : (
                "Ошибка запроса"
              )}
            </Notice>
          ) : null}

          {pendingQuery.data?.invoices?.length ? (
            <div className="mt-2 overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-white/10 text-xs uppercase muted">
                    <th className="py-2 pr-3">Сделка</th>
                    <th className="py-2 pr-3">Инвойс</th>
                    <th className="py-2 pr-3">Статус</th>
                    <th className="py-2 pr-3">Покупатель</th>
                    <th className="py-2 pr-3">Продавец</th>
                    <th className="py-2 pr-3">Сумма</th>
                    <th className="py-2 pr-3">Срок</th>
                    <th className="py-2 pr-3">Создан</th>
                    <th className="py-2 pr-3">Действие</th>
                  </tr>
                </thead>
                <tbody>
                  {pendingQuery.data.invoices.map((inv) => (
                    <tr key={inv.id} className="border-b border-white/5 align-top">
                      <td className="py-2 pr-3">
                        <Link className="underline" href={`/deals/${inv.deal_id}`}>
                          <code className="text-xs">{displayId(inv.deal_id, "")}</code>
                        </Link>
                        <div className="muted text-xs">аукцион {displayId(inv.auction_id, "")}</div>
                      </td>
                      <td className="py-2 pr-3">
                        <code className="text-xs">{displayId(inv.id, "")}</code>
                        {inv.provider ? <div className="muted text-xs">{inv.provider}</div> : null}
                      </td>
                      <td className="py-2 pr-3">{invoiceStatusLabel(inv.status)}</td>
                      <td className="py-2 pr-3">
                        <code className="text-xs">{displayId(inv.buyer_company_id, "")}</code>
                      </td>
                      <td className="py-2 pr-3">
                        <code className="text-xs">{displayId(inv.seller_company_id, "")}</code>
                      </td>
                      <td className="py-2 pr-3">
                        {formatMoney(inv.total_amount)} <span className="muted">{inv.currency}</span>
                        <div className="muted text-xs">товары {formatMoney(inv.goods_amount)}</div>
                      </td>
                      <td className="py-2 pr-3 muted">{formatDateTime(inv.due_at)}</td>
                      <td className="py-2 pr-3 muted">{formatDateTime(inv.created_at)}</td>
                      <td className="py-2 pr-3">
                        <Button
                          type="button"
                          variant="secondary"
                          size="sm"
                          disabled={confirmMu.isPending}
                          onClick={() => confirmMu.mutate(inv.id)}
                        >
                          Подтвердить оплату
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : !pendingQuery.isLoading && !pendingQuery.error ? (
            <p className="muted text-sm">Нет счетов в статусе PAYMENT_PENDING.</p>
          ) : null}

          {confirmMu.error ? (
            <div className="mt-4">
              <Notice tone="warning" title="Подтверждение">
                {confirmMu.error instanceof ApiError ? confirmMu.error.message : String(confirmMu.error)}
              </Notice>
            </div>
          ) : null}
        </Card>
      </div>
    </AuthGuard>
  );
}
