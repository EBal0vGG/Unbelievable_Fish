"use client";

import { useMemo } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { listActivitiesStore } from "@/shared/api/mock-store";
import { isOwnedLot, isOwnedProduct } from "@/shared/lib/access";
import { formatDateTime } from "@/shared/lib/format";
import { Card } from "@/shared/ui/card";
import { Notice } from "@/shared/ui/notice";

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
    <div className="page-stack">
      <div className="page-heading">
        <p className="eyebrow">Профиль</p>
        <h1>Мой профиль</h1>
      </div>

      {!session ? (
        <Notice tone="warning" title="Нет активной сессии">
          Войдите, чтобы увидеть персональные данные и историю действий.
        </Notice>
      ) : null}

      <div className="info-grid">
        <Card className="form-card">
          <div className="stack-md">
            <h2>Профиль компании</h2>
            <div className="metric-grid">
              <div>
                <span>Компания</span>
                <strong>{session?.companyId ?? "не задана"}</strong>
              </div>
              <div>
                <span>Пользователь</span>
                <strong>{session?.userId ?? "не задан"}</strong>
              </div>
              <div>
                <span>Роль</span>
                <strong>{session?.role ?? "гость"}</strong>
              </div>
              <div>
                <span>Профиль</span>
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
  );
}
