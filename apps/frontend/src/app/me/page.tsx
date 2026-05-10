"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { BILLING_POLL_MS, useBillingBalanceQuery, useTopUpsQuery } from "@/entities/billing/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { AuthGuard } from "@/features/auth/ui/auth-guard";
import { ApiError } from "@/shared/api/http-client";
import {
  adminConfirmDealInvoice,
  adminExpireDealInvoice,
  adminMarkPayoutPaid,
  adminMarkPayoutReady,
  createTopUp,
  fakeConfirmTopUp,
} from "@/shared/api/billing-service";
import { listUsers, promoteUserToAdmin } from "@/shared/api/identity-service";
import { env, isFakeBillingUiAllowed } from "@/shared/config/env";
import { listActivitiesStore } from "@/shared/api/mock-store";
import { isAdminSession, isOwnedLot, isOwnedProduct } from "@/shared/lib/access";
import { displayCompany, displayId, displayPerson } from "@/shared/lib/display";
import { formatDateTime, formatMoney } from "@/shared/lib/format";
import { roleLabels } from "@/shared/lib/labels";
import type { UserRole } from "@/shared/types/domain";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { EmptyState } from "@/shared/ui/empty-state";
import { Field } from "@/shared/ui/field";
import { Notice } from "@/shared/ui/notice";
import { PageHeader } from "@/shared/ui/page-header";
import { Select } from "@/shared/ui/select";
import { Input } from "@/shared/ui/input";

export default function MyProfilePage() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const [targetUserID, setTargetUserID] = useState("");
  const [availableUsers, setAvailableUsers] = useState<Array<{ id: string; login: string; role: UserRole }>>([]);
  const [loadUsersError, setLoadUsersError] = useState<string | null>(null);
  const [promoteError, setPromoteError] = useState<string | null>(null);
  const [promoteSuccess, setPromoteSuccess] = useState<string | null>(null);
  const [promotePending, setPromotePending] = useState(false);
  const [topUpAmount, setTopUpAmount] = useState("");
  const [adminInvoiceId, setAdminInvoiceId] = useState("");
  const [adminPayoutId, setAdminPayoutId] = useState("");
  const productsQuery = useProductsQuery();
  const lotsQuery = useLotsQuery();
  const auctionsQuery = useAuctionsQuery(session);
  const personName = displayPerson(session);
  const companyName = session?.companyId ? displayCompany(session.companyId) : "Компания не привязана";

  const activities = useMemo(() => listActivitiesStore(session), [session]);
  const visibleProducts = useMemo(
    () => (productsQuery.data?.data ?? []).filter((product) => isOwnedProduct(product, session)),
    [productsQuery.data?.data, session],
  );
  const visibleLots = useMemo(
    () => (lotsQuery.data?.data ?? []).filter((lot) => isOwnedLot(lot, session)),
    [lotsQuery.data?.data, session],
  );
  const publicAuctions = useMemo(
    () => (auctionsQuery.data?.data ?? []).filter((auction) => auction.state !== "DRAFT"),
    [auctionsQuery.data?.data],
  );
  const canPromoteAdmins = isAdminSession(session);
  const balanceQuery = useBillingBalanceQuery(session);
  const topUpsQuery = useTopUpsQuery(session);

  useEffect(() => {
    const list = topUpsQuery.data?.top_ups ?? [];
    if (!list.some((t) => t.status === "PENDING")) {
      return;
    }
    const id = window.setInterval(() => {
      void queryClient.invalidateQueries({ queryKey: ["billing-balance"] });
    }, BILLING_POLL_MS);
    return () => window.clearInterval(id);
  }, [topUpsQuery.data?.top_ups, queryClient]);

  const createTopUpMu = useMutation({
    mutationFn: async () => {
      if (!session) {
        throw new Error("Нет сессии");
      }
      const rub = Math.round(Number.parseFloat(topUpAmount.replace(",", ".")));
      if (!Number.isFinite(rub) || rub <= 0) {
        throw new Error("Укажите сумму пополнения (руб., целое число).");
      }
      await createTopUp(rub, session);
    },
    onSuccess: async () => {
      setTopUpAmount("");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["billing-topups"] }),
        queryClient.invalidateQueries({ queryKey: ["billing-balance"] }),
      ]);
    },
  });

  const fakeTopUpMu = useMutation({
    mutationFn: async (topUpId: string) => {
      if (!session) {
        throw new Error("Нет сессии");
      }
      await fakeConfirmTopUp(topUpId, session);
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["billing-topups"] }),
        queryClient.invalidateQueries({ queryKey: ["billing-balance"] }),
      ]);
    },
  });

  const adminConfirmInvMu = useMutation({
    mutationFn: async () => {
      if (!session || !adminInvoiceId.trim()) {
        throw new Error("Укажите invoice id");
      }
      await adminConfirmDealInvoice(adminInvoiceId.trim(), session);
    },
  });

  const adminExpireInvMu = useMutation({
    mutationFn: async () => {
      if (!session || !adminInvoiceId.trim()) {
        throw new Error("Укажите invoice id");
      }
      await adminExpireDealInvoice(adminInvoiceId.trim(), session);
    },
  });

  const adminPayoutReadyMu = useMutation({
    mutationFn: async () => {
      if (!session || !adminPayoutId.trim()) {
        throw new Error("Укажите payout id");
      }
      await adminMarkPayoutReady(adminPayoutId.trim(), session);
    },
  });

  const adminPayoutPaidMu = useMutation({
    mutationFn: async () => {
      if (!session || !adminPayoutId.trim()) {
        throw new Error("Укажите payout id");
      }
      await adminMarkPayoutPaid(adminPayoutId.trim(), session);
    },
  });

  useEffect(() => {
    if (!session || !canPromoteAdmins) {
      setAvailableUsers([]);
      setLoadUsersError(null);
      return;
    }
    let isCancelled = false;
    const loadUsers = async () => {
      try {
        const users = await listUsers(session);
        if (isCancelled) {
          return;
        }
        const options = users
          .filter((user) => user.role !== "admin")
          .map((user) => ({ id: user.id, login: user.login, role: user.role }));
        setAvailableUsers(options);
        setLoadUsersError(null);
      } catch (error) {
        if (isCancelled) {
          return;
        }
        setLoadUsersError(error instanceof ApiError ? error.message : "Не удалось загрузить список пользователей.");
      }
    };
    void loadUsers();
    return () => {
      isCancelled = true;
    };
  }, [session, canPromoteAdmins]);

  const submitPromoteAdmin = async () => {
    if (!session) {
      return;
    }
    const userID = targetUserID.trim();
    if (!userID) {
      setPromoteError("Укажите ID пользователя для назначения администратором.");
      setPromoteSuccess(null);
      return;
    }
    setPromotePending(true);
    setPromoteError(null);
    setPromoteSuccess(null);
    try {
      const result = await promoteUserToAdmin(userID, session);
      setPromoteSuccess(`Пользователь ${result.login} (${result.id}) назначен администратором.`);
      setTargetUserID("");
    } catch (error) {
      setPromoteError(error instanceof ApiError ? error.message : "Не удалось назначить администратора.");
    } finally {
      setPromotePending(false);
    }
  };

  return (
    <AuthGuard>
      <div className="page-stack">
        <PageHeader
          compact
          eyebrow="Профиль"
          title={session?.companyId ? "Профиль компании" : "Мой профиль"}
          description="Данные пользователя, роль, компания и операционная активность в бирже."
          metrics={
            <>
              <div>
                <span>Роль</span>
                <strong>{session ? roleLabels[session.role] : "гость"}</strong>
              </div>
              <div>
                <span>Рейтинг</span>
                <strong>4.8</strong>
              </div>
            </>
          }
        />

        <div className="profile-grid">
          <div className="stack-lg">
          <Card className="form-card profile-card profile-card-primary">
            <div className="stack-md">
              <div className="profile-identity">
                <span>{personName.slice(0, 2).toUpperCase()}</span>
                <div>
                  <p className="eyebrow">Профиль компании</p>
                  <h2>{companyName}</h2>
                  <p className="muted">Рабочий аккаунт: {personName}</p>
                </div>
              </div>
              <div className="metric-grid">
                <div>
                  <span>Пользователь</span>
                  <strong title={session?.login}>{personName}</strong>
                </div>
                <div>
                  <span>Роль</span>
                  <strong>{session ? roleLabels[session.role] : "Гость"}</strong>
                </div>
                <div>
                  <span>Логин</span>
                  <strong title={session?.login}>{session?.login ?? "Не указан"}</strong>
                </div>
                <div>
                  <span>Технический номер</span>
                  <strong title={session?.companyId}>{displayId(session?.companyId, "")}</strong>
                </div>
                <div>
                  <span>Сценарий</span>
                  <strong>{session?.mode === "register" ? "Новая регистрация" : "Рабочий вход"}</strong>
                </div>
                <div>
                  <span>Обновлено</span>
                  <strong>{session ? formatDateTime(session.updatedAt) : "—"}</strong>
                </div>
              </div>
            </div>
          </Card>

          <Card className="form-card">
            <div className="stack-md">
              <h2>Мои действия</h2>
              {activities.length ? (
                <div className="activity-list">
                  {activities.map((activity) => (
                    <div className="activity-item" key={activity.id}>
                      <div>
                        <strong>{activity.title}</strong>
                        <p className="muted">{activity.description}</p>
                      </div>
                      <span className="muted">{formatDateTime(activity.at)}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState
                  title="Действия не записаны"
                  description="Когда появятся новые продукты, лоты, торги или сделки, они будут видны в этом разделе."
                  framed={false}
                />
              )}
            </div>
          </Card>
          </div>

          <div className="stack-lg">
          <Card className="form-card profile-card rating-card">
            <div className="stack-md">
              <div className="rating-header">
                <div>
                  <p className="eyebrow">Рейтинг</p>
                  <h2>Профиль надежности</h2>
                </div>
                <strong>4.8</strong>
              </div>
              <div className="metric-grid">
                <div>
                  <span>Общий рейтинг</span>
                  <strong>4.8 / 5</strong>
                </div>
                <div>
                  <span>Продукты</span>
                  <strong>{visibleProducts.length}</strong>
                </div>
                <div>
                  <span>Лоты</span>
                  <strong>{visibleLots.length}</strong>
                </div>
                <div>
                  <span>Публичные аукционы</span>
                  <strong>{publicAuctions.length}</strong>
                </div>
              </div>
            </div>
          </Card>

          <Card className="form-card profile-card">
            <div className="stack-md">
              <h2>Доступ</h2>
              <div className="metric-grid">
                <div>
                  <span>Роль</span>
                  <strong>{session ? roleLabels[session.role] : "Гость"}</strong>
                </div>
                <div>
                  <span>Технический номер</span>
                  <strong title={session?.companyId}>{displayId(session?.companyId, "")}</strong>
                </div>
              </div>
            </div>
          </Card>

          <Card className="form-card profile-card">
            <div className="stack-md">
              <div>
                <p className="eyebrow">Billing</p>
                <h2>Баланс компании</h2>
              </div>
              <p className="muted">Внутренний счет компании (RUB).</p>
              {balanceQuery.isLoading ? <p className="muted">Загрузка...</p> : null}
              {balanceQuery.error ? (
                <Notice tone="warning" title="Баланс недоступен">
                  {balanceQuery.error instanceof ApiError ? balanceQuery.error.message : "Ошибка запроса к billing."}
                </Notice>
              ) : null}
              {balanceQuery.data ? (
                <div className="metric-grid">
                  <div>
                    <span>Доступно</span>
                    <strong>{formatMoney(balanceQuery.data.available)}</strong>
                  </div>
                  <div>
                    <span>Зарезервировано</span>
                    <strong>{formatMoney(balanceQuery.data.held)}</strong>
                  </div>
                  <div>
                    <span>Всего</span>
                    <strong>{formatMoney(balanceQuery.data.total)}</strong>
                  </div>
                  <div>
                    <span>Валюта</span>
                    <strong>{balanceQuery.data.currency}</strong>
                  </div>
                </div>
              ) : null}

              <div className="stack-md border-t border-white/10 pt-4">
                <h3 className="text-sm font-medium">Пополнение счёта</h3>
                <p className="muted text-sm">
                  Создание заявки на пополнение (провайдер / fake-flow на стороне billing). Если на сервере включён
                  авто-колбэк fake-провайдера (<code className="text-xs">BILLING_FAKE_PROVIDER_AUTO_CONFIRM</code>
                  ), заявка подтвердится вебхуком через несколько секунд и баланс обновится без кнопки ниже.
                </p>
                <div className="flex flex-wrap items-end gap-2">
                  <Field label="Сумма (руб.)" className="min-w-[10rem]">
                    <Input
                      value={topUpAmount}
                      onChange={(e) => setTopUpAmount(e.target.value)}
                      inputMode="numeric"
                      placeholder="1000"
                    />
                  </Field>
                  <Button
                    type="button"
                    onClick={() => createTopUpMu.mutate()}
                    disabled={createTopUpMu.isPending || !session}
                  >
                    {createTopUpMu.isPending ? "Создаём…" : "Создать пополнение"}
                  </Button>
                </div>
                {createTopUpMu.error ? (
                  <p className="text-sm text-amber-600">{createTopUpMu.error.message}</p>
                ) : null}
                {topUpsQuery.isLoading ? <p className="muted text-sm">Загрузка заявок…</p> : null}
                {topUpsQuery.data?.top_ups?.length ? (
                  <ul className="stack-sm text-sm">
                    {topUpsQuery.data.top_ups.map((tu) => (
                      <li key={tu.id} className="flex flex-wrap items-center justify-between gap-2">
                        <span>
                          {formatMoney(tu.amount)} {tu.currency} · {tu.status} ·{" "}
                          <code className="text-xs">{tu.id}</code>
                        </span>
                        {isFakeBillingUiAllowed(balanceQuery.data?.top_up_fake_confirm_enabled) &&
                        tu.status === "PENDING" ? (
                          <Button
                            type="button"
                            variant="secondary"
                            size="sm"
                            disabled={fakeTopUpMu.isPending}
                            onClick={() => fakeTopUpMu.mutate(tu.id)}
                          >
                            Fake confirm
                          </Button>
                        ) : null}
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            </div>
          </Card>

          {canPromoteAdmins && env.enableBillingAdminUI ? (
            <Card className="form-card profile-card">
              <div className="stack-md">
                <Notice tone="warning" title="Demo / admin billing">
                  Операции ниже обходят реальный платёжный провайдер. Включайте только вместе с{" "}
                  <code>BILLING_ENABLE_ADMIN_ACTIONS</code> на сервере.
                </Notice>
                <p className="flex flex-wrap gap-x-4 gap-y-1 text-sm">
                  <Link className="underline" href="/admin/billing/invoices">
                    Счета в ожидании оплаты (список + подтверждение)
                  </Link>
                  <Link className="underline" href="/admin/billing/payouts">
                    Очередь выплат продавцам
                  </Link>
                </p>
                <h2>Billing (admin)</h2>
                <Field label="Invoice ID">
                  <Input value={adminInvoiceId} onChange={(e) => setAdminInvoiceId(e.target.value)} placeholder="inv-…" />
                </Field>
                <div className="inline-actions">
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={adminConfirmInvMu.isPending}
                    onClick={() => adminConfirmInvMu.mutate()}
                  >
                    Confirm invoice (высокий риск)
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={adminExpireInvMu.isPending}
                    onClick={() => adminExpireInvMu.mutate()}
                  >
                    Expire invoice
                  </Button>
                </div>
                {(adminConfirmInvMu.error || adminExpireInvMu.error) && (
                  <p className="text-sm text-amber-600">
                    {adminConfirmInvMu.error?.message ?? adminExpireInvMu.error?.message}
                  </p>
                )}
                <Field label="Payout ID">
                  <Input value={adminPayoutId} onChange={(e) => setAdminPayoutId(e.target.value)} placeholder="pay-…" />
                </Field>
                <p className="muted text-sm">
                  <strong>PAID</strong> здесь зачисляет средства на <strong>внутренний баланс</strong> продавца в billing, а не в банк.
                </p>
                <div className="inline-actions">
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={adminPayoutReadyMu.isPending}
                    onClick={() => adminPayoutReadyMu.mutate()}
                  >
                    Mark payout ready
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={adminPayoutPaidMu.isPending}
                    onClick={() => adminPayoutPaidMu.mutate()}
                  >
                    Mark payout paid
                  </Button>
                </div>
                {(adminPayoutReadyMu.error || adminPayoutPaidMu.error) && (
                  <p className="text-sm text-amber-600">
                    {adminPayoutReadyMu.error?.message ?? adminPayoutPaidMu.error?.message}
                  </p>
                )}
              </div>
            </Card>
          ) : null}
          </div>
        </div>

        {canPromoteAdmins ? (
          <Card className="form-card">
            <div className="stack-md">
              <h2>Администрирование</h2>
              <p className="muted">Назначьте пользователя администратором из списка.</p>
              {loadUsersError ? (
                <Notice tone="warning" title="Не удалось загрузить пользователей">
                  {loadUsersError}
                </Notice>
              ) : null}
              {promoteSuccess ? (
                <Notice tone="success" title="Роль обновлена">
                  {promoteSuccess}
                </Notice>
              ) : null}
              {promoteError ? (
                <Notice tone="warning" title="Не удалось назначить администратора">
                  {promoteError}
                </Notice>
              ) : null}
              <Field label="Пользователь">
                <Select value={targetUserID} onChange={(event) => setTargetUserID(event.target.value)}>
                  <option value="">Выберите пользователя</option>
                  {availableUsers.map((user) => (
                    <option key={user.id} value={user.id}>
                      {user.login} ({user.id}) - {roleLabels[user.role]}
                    </option>
                  ))}
                </Select>
              </Field>
              <div className="inline-actions">
                <Button type="button" onClick={submitPromoteAdmin} disabled={promotePending || !targetUserID}>
                  {promotePending ? "Назначаем..." : "Назначить администратором"}
                </Button>
              </div>
            </div>
          </Card>
        ) : null}
      </div>
    </AuthGuard>
  );
}
