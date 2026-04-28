"use client";

import { useMemo } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { AuthGuard } from "@/features/auth/ui/auth-guard";
import { listActivitiesStore } from "@/shared/api/mock-store";
import { isOwnedLot, isOwnedProduct } from "@/shared/lib/access";
import { formatDateTime } from "@/shared/lib/format";
import { roleLabels } from "@/shared/lib/labels";
import { Card } from "@/shared/ui/card";

export default function MyProfilePage() {
  const { session } = useAuth();
  const productsQuery = useProductsQuery();
  const lotsQuery = useLotsQuery();
  const auctionsQuery = useAuctionsQuery();

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

  return (
    <AuthGuard>
      <div className="page-stack">
        <div className="page-heading">
          <p className="eyebrow">Профиль</p>
          <h1>Мой профиль</h1>
        </div>

        <div className="info-grid">
          <Card className="form-card">
            <div className="stack-md">
              <h2>{session?.companyId ? "Профиль компании" : "Профиль пользователя"}</h2>
              <div className="metric-grid">
                <div>
                  <span>Компания</span>
                  <strong>{session?.companyId ?? "не задана"}</strong>
                </div>
                <div>
                  <span>Пользователь</span>
                  <strong>{session?.name ?? "не задан"}</strong>
                </div>
                <div>
                  <span>Логин</span>
                  <strong>{session?.login ?? "не задан"}</strong>
                </div>
                <div>
                  <span>Роль</span>
                  <strong>{session ? roleLabels[session.role] : "гость"}</strong>
                </div>
                <div>
                  <span>Сценарий</span>
                  <strong>{session?.mode === "register" ? "Новая регистрация" : "Рабочий вход"}</strong>
                </div>
                <div>
                  <span>Обновлено</span>
                  <strong>{session ? formatDateTime(session.updatedAt) : "н/д"}</strong>
                </div>
              </div>
            </div>
          </Card>

          <Card className="form-card">
            <div className="stack-md">
              <h2>Рейтинг</h2>
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
        </div>

        <Card className="form-card">
          <div className="stack-md">
            <h2>Мои действия</h2>
            {activities.length ? (
              <table className="table">
                <thead>
                  <tr>
                    <th>Событие</th>
                    <th>Описание</th>
                    <th>Время</th>
                  </tr>
                </thead>
                <tbody>
                  {activities.map((activity) => (
                    <tr key={activity.id}>
                      <td>{activity.title}</td>
                      <td>{activity.description}</td>
                      <td>{formatDateTime(activity.at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p className="muted">Действия еще не записаны.</p>
            )}
          </div>
        </Card>
      </div>
    </AuthGuard>
  );
}
