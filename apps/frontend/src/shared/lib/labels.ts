import type { AuctionState, LotStatus, ProductStatus, UserRole } from "@/shared/types/domain";

export const roleLabels: Record<UserRole, string> = {
  admin: "администратор",
  seller: "продавец",
  buyer: "покупатель",
};

export const productStatusLabels: Record<ProductStatus, string> = {
  DRAFT: "Черновик",
  PUBLISHED: "Опубликован",
};

export const lotStatusLabels: Record<LotStatus, string> = {
  DRAFT: "Черновик",
  PUBLISHED: "Опубликован",
  CLOSED: "Закрыт",
  CANCELLED: "Отменен",
};

export const auctionStateLabels: Record<AuctionState, string> = {
  DRAFT: "Черновик",
  PUBLISHED: "Идет прием ставок",
  CLOSED: "Закрыт",
  WON: "Победитель выбран",
  CANCELLED: "Отменен",
};
