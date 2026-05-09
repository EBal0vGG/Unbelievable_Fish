const INVOICE_STATUS_LABELS: Record<string, string> = {
  PAYMENT_PENDING: "Ожидает оплаты",
  PAID: "Оплачен",
  EXPIRED: "Просрочен",
  CANCELLED: "Отменён",
};

const PAYOUT_STATUS_LABELS: Record<string, string> = {
  PENDING: "В очереди",
  READY: "Готова к выплате",
  PAID: "Выплачена",
  CANCELLED: "Отменена",
  FAILED: "Ошибка",
};

export function invoiceStatusLabel(code: string): string {
  return INVOICE_STATUS_LABELS[code] ?? code;
}

export function payoutStatusLabel(code: string): string {
  return PAYOUT_STATUS_LABELS[code] ?? code;
}
