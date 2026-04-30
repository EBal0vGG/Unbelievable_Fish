"use client";

import Link from "next/link";
import { useParams } from "next/navigation";

import { useDealConfirmationsQuery, useDealDetailsQuery } from "@/entities/deal/model/hooks";
import { dealStatusLabels } from "@/entities/deal/ui/deal-card";
import { useAuth } from "@/entities/session/model/auth-context";
import { DealActionPanel } from "@/features/deal/ui/deal-action-panel";
import { formatDateTime, formatMoney, shortId } from "@/shared/lib/format";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { EmptyState } from "@/shared/ui/empty-state";
import { Notice } from "@/shared/ui/notice";
import type { DealConfirmationRecord, DealStatus } from "@/shared/types/domain";

const steps: Array<{ status: DealStatus; label: string }> = [
  { status: "pending", label: "Победитель" },
  { status: "confirmed", label: "Подтверждение" },
  { status: "contract_prepared", label: "Контракт" },
  { status: "contract_signed", label: "Подпись" },
  { status: "payment_requested", label: "Инвойс" },
  { status: "paid", label: "Оплата" },
  { status: "shipment_requested", label: "Отгрузка" },
  { status: "shipped", label: "В пути" },
  { status: "completed", label: "Закрыта" },
];

function stepState(current: DealStatus, step: DealStatus): "done" | "active" | "todo" {
  if (current === "cancelled") {
    return "todo";
  }
  const currentIndex = steps.findIndex((item) => item.status === current);
  const stepIndex = steps.findIndex((item) => item.status === step);
  if (stepIndex < currentIndex) {
    return "done";
  }
  if (stepIndex === currentIndex) {
    return "active";
  }
  return "todo";
}

export default function DealDetailsPage() {
  const params = useParams();
  const { session } = useAuth();
  const rawId = params.id;
  const dealId = Array.isArray(rawId) ? rawId[0] : rawId ?? "";
  const dealQuery = useDealDetailsQuery(dealId, session);
  const confirmationsQuery = useDealConfirmationsQuery(dealId, session);
  const deal = dealQuery.data?.data;
  const confirmations = confirmationsQuery.data?.data ?? [];

  if (!session) {
    return (
      <EmptyState
        title="Войдите, чтобы открыть сделку"
        description="Сделка доступна только компаниям-участникам контракта."
        actionHref={`/login?next=/deals/${dealId}`}
        actionLabel="Войти"
      />
    );
  }

  if (!dealId) {
    return (
      <EmptyState
        title="Сделка не выбрана"
        description="Откройте сделку из списка твоих сделок."
        actionHref="/deals"
        actionLabel="К твоим сделкам"
      />
    );
  }

  if (!deal) {
    return (
      <EmptyState
        title="Сделка не найдена"
        description="Проверьте ссылку или вернитесь к списку сделок."
        actionHref="/deals"
        actionLabel="Вернуться к твоим сделкам"
      />
    );
  }

  return (
    <div className="page-stack">
      <section className="page-hero compact-hero">
        <div>
          <p className="eyebrow">Сделка {shortId(deal.id)}</p>
          <h1>{deal.productSnapshot.name || "Контрактная поставка"}</h1>
          <p className="hero-copy">
            {deal.supplierId} поставляет {deal.customerId}. Статус: {dealStatusLabels[deal.status]}.
          </p>
        </div>
        <div className="hero-actions">
          <Link className={buttonStyles({ variant: "secondary" })} href={`/auctions/${deal.auctionId}`}>
            Аукцион
          </Link>
          <button
            className={buttonStyles({ variant: "ghost" })}
            onClick={() => {
              void dealQuery.refetch();
              void confirmationsQuery.refetch();
            }}
            type="button"
          >
            Обновить
          </button>
        </div>
      </section>

      {deal.status === "cancelled" ? (
        <Notice tone="warning" title="Сделка отменена">
          Дальнейшие действия по контракту закрыты.
        </Notice>
      ) : null}

      <Card className="timeline-card">
        <div className="deal-timeline">
          {steps.map((step) => (
            <div key={step.status} className={`timeline-step timeline-${stepState(deal.status, step.status)}`}>
              <span />
              <strong>{step.label}</strong>
            </div>
          ))}
        </div>
      </Card>

      <div className="info-grid deal-detail-grid">
        <Card className="form-card">
          <div className="stack-md">
            <div>
              <p className="eyebrow">Экономика</p>
              <h2>Финансовая сводка</h2>
            </div>
            <div className="metric-grid">
              <div>
                <span>Сумма</span>
                <strong>{formatMoney(deal.totalAmount)}</strong>
              </div>
              <div>
                <span>Цена</span>
                <strong>{formatMoney(deal.unitPrice)}</strong>
              </div>
              <div>
                <span>Количество</span>
                <strong>
                  {deal.quantity} {deal.productSnapshot.unit || "ед."}
                </strong>
              </div>
              <div>
                <span>Тип</span>
                <strong>{deal.type}</strong>
              </div>
              <div>
                <span>Создана</span>
                <strong>{formatDateTime(deal.createdAt)}</strong>
              </div>
              <div>
                <span>Подтверждена</span>
                <strong>{formatDateTime(deal.confirmedAt)}</strong>
              </div>
            </div>
          </div>
        </Card>

        <Card className="form-card">
          <DealActionPanel deal={deal} confirmations={confirmations} />
        </Card>
      </div>

      <div className="info-grid">
        <Card className="form-card">
          <div className="stack-md">
            <div>
              <p className="eyebrow">Продукт</p>
              <h2>{deal.productSnapshot.name || "Продукт"}</h2>
              <p className="muted">{deal.productSnapshot.description || "Описание не задано"}</p>
            </div>
            <div className="metric-grid">
              <div>
                <span>Категория</span>
                <strong>{deal.productSnapshot.category || "н/д"}</strong>
              </div>
              <div>
                <span>Обработка</span>
                <strong>{deal.productSnapshot.processingType || "н/д"}</strong>
              </div>
              <div>
                <span>Размер</span>
                <strong>{deal.productSnapshot.size || "н/д"}</strong>
              </div>
              <div>
                <span>Вес</span>
                <strong>
                  {deal.productSnapshot.weight} {deal.productSnapshot.unit}
                </strong>
              </div>
              <div>
                <span>Страна</span>
                <strong>{deal.productSnapshot.originCountry || "н/д"}</strong>
              </div>
              <div>
                <span>Product ID</span>
                <strong>{shortId(deal.productSnapshot.productId)}</strong>
              </div>
            </div>
          </div>
        </Card>

        <Card className="form-card">
          <ConfirmationHistory confirmations={confirmations} />
        </Card>

        <Card className="form-card">
          <div className="stack-md">
            <div>
              <p className="eyebrow">Контракт</p>
              <h2>{deal.contract?.number ?? "Контракт не подготовлен"}</h2>
            </div>
            <div className="metric-grid">
              <div>
                <span>Подготовлен</span>
                <strong>{formatDateTime(deal.contract?.preparedAt)}</strong>
              </div>
              <div>
                <span>Подписан</span>
                <strong>{formatDateTime(deal.contract?.signedAt)}</strong>
              </div>
              <div>
                <span>Подписант</span>
                <strong>{deal.contract?.signedBy ?? "н/д"}</strong>
              </div>
              <div>
                <span>Signature ref</span>
                <strong>{deal.contract?.signatureRef ?? "н/д"}</strong>
              </div>
            </div>
            {deal.contract?.documentUrl ? (
              <a className={buttonStyles({ variant: "ghost", size: "sm" })} href={deal.contract.documentUrl}>
                Открыть документ
              </a>
            ) : null}
          </div>
        </Card>
      </div>
    </div>
  );
}

function ConfirmationHistory({ confirmations }: { confirmations: DealConfirmationRecord[] }) {
  return (
    <div className="stack-md">
      <div>
        <p className="eyebrow">Подтверждения</p>
        <h2>История согласований</h2>
      </div>
      {confirmations.length === 0 ? (
        <p className="muted">По этой сделке еще нет запросов на подтверждение этапов.</p>
      ) : (
        <div className="stack-md">
          {confirmations.map((confirmation) => (
            <div key={confirmation.id} className="stack-sm">
              <strong>
                {confirmation.stage} · {confirmation.status}
              </strong>
              <p className="muted">
                {confirmation.requestedByCompanyId} → {confirmation.counterpartyCompanyId} ·{" "}
                {formatDateTime(confirmation.requestedAt)}
              </p>
              {confirmation.comment ? <p>{confirmation.comment}</p> : null}
              {confirmation.reason ? <p className="muted">Причина: {confirmation.reason}</p> : null}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
