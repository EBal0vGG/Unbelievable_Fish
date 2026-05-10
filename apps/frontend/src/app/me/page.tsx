"use client";

import { useEffect, useMemo, useState } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { useBillingBalanceQuery } from "@/entities/billing/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { AuthGuard } from "@/features/auth/ui/auth-guard";
import { ApiError } from "@/shared/api/http-client";
import { listUsers, promoteUserToAdmin } from "@/shared/api/identity-service";
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

export default function MyProfilePage() {
  const { session } = useAuth();
  const [targetUserID, setTargetUserID] = useState("");
  const [availableUsers, setAvailableUsers] = useState<Array<{ id: string; login: string; role: UserRole }>>([]);
  const [loadUsersError, setLoadUsersError] = useState<string | null>(null);
  const [promoteError, setPromoteError] = useState<string | null>(null);
  const [promoteSuccess, setPromoteSuccess] = useState<string | null>(null);
  const [promotePending, setPromotePending] = useState(false);
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
            </div>
          </Card>
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
