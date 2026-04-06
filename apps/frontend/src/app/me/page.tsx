"use client";

import { useMemo } from "react";

import { useAuctionsQuery } from "@/entities/auction/model/hooks";
import { useLotsQuery, useProductsQuery } from "@/entities/lot/model/hooks";
import { useFishCatalogQuery } from "@/entities/fish/model/hooks";
import { useAuth } from "@/entities/session/model/auth-context";
import { listActivitiesStore } from "@/shared/api/mock-store";
import { formatDateTime } from "@/shared/lib/format";
import { Card } from "@/shared/ui/card";
import { Notice } from "@/shared/ui/notice";

export default function MyContextPage() {
  const { session } = useAuth();
  const fishQuery = useFishCatalogQuery(session);
  const productsQuery = useProductsQuery();
  const lotsQuery = useLotsQuery();
  const auctionsQuery = useAuctionsQuery();

  const activities = useMemo(() => listActivitiesStore(), []);

  return (
    <div className="page-stack">
      <div className="page-heading">
        <p className="eyebrow">My Context</p>
        <h1>Мой контекст и действия</h1>
      </div>

      {!session ? (
        <Notice tone="warning" title="Нет активной сессии">
          Войдите или сохраните контекст через регистрацию, чтобы UI начал пробрасывать
          `X-Company-ID` и `X-User-ID` в backend.
        </Notice>
      ) : null}

      <div className="info-grid">
        <Card className="form-card">
          <div className="stack-md">
            <h2>Текущий пользователь</h2>
            <div className="metric-grid">
              <div>
                <span>Company ID</span>
                <strong>{session?.companyId ?? "не задан"}</strong>
              </div>
              <div>
                <span>User ID</span>
                <strong>{session?.userId ?? "не задан"}</strong>
              </div>
              <div>
                <span>Mode</span>
                <strong>{session?.mode ?? "guest"}</strong>
              </div>
              <div>
                <span>Updated</span>
                <strong>{session ? formatDateTime(session.updatedAt) : "н/д"}</strong>
              </div>
            </div>
          </div>
        </Card>

        <Card className="form-card">
          <div className="stack-md">
            <h2>Локальный read model</h2>
            <div className="metric-grid">
              <div>
                <span>Fish</span>
                <strong>{fishQuery.data?.data.length ?? 0}</strong>
              </div>
              <div>
                <span>Products</span>
                <strong>{productsQuery.data?.data.length ?? 0}</strong>
              </div>
              <div>
                <span>Lots</span>
                <strong>{lotsQuery.data?.data.length ?? 0}</strong>
              </div>
              <div>
                <span>Auctions</span>
                <strong>{auctionsQuery.data?.data.length ?? 0}</strong>
              </div>
            </div>
          </div>
        </Card>
      </div>

      <Card className="form-card">
        <div className="stack-md">
          <h2>Recent actions</h2>
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
