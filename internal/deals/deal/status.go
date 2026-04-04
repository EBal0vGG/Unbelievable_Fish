package deal

// DealStatus - статус сделки
type DealStatus string

const (
	DealStatusPending           DealStatus = "pending"            // Ожидает подтверждения (создана при AuctionWon)
	DealStatusConfirmed         DealStatus = "confirmed"          // Подтверждена
	DealStatusContractPrepared  DealStatus = "contract_prepared"  // Контракт подготовлен
	DealStatusContractSigned    DealStatus = "contract_signed"    // Контракт подписан
	DealStatusPaymentRequested  DealStatus = "payment_requested"  // Запрос на оплату
	DealStatusPaid              DealStatus = "paid"               // Оплачена
	DealStatusShipmentRequested DealStatus = "shipment_requested" // Запрос на доставку
	DealStatusShipped           DealStatus = "shipped"            // Отправлена
	DealStatusCompleted         DealStatus = "completed"          // Завершена
	DealStatusCancelled         DealStatus = "cancelled"          // Отменена
)

// DealType - тип сделки
type DealType string

const (
	DealTypeDirect  DealType = "direct"  // Прямая покупка
	DealTypeAuction DealType = "auction" // Аукцион
)

// Статусные группы для проверок
var (
	// ActiveStatuses - статусы, в которых сделка считается активной
	ActiveStatuses = []DealStatus{
		DealStatusPending,
		DealStatusConfirmed,
		DealStatusContractPrepared,
		DealStatusContractSigned,
		DealStatusPaymentRequested,
		DealStatusPaid,
		DealStatusShipmentRequested,
		DealStatusShipped,
	}

	// ModifiableStatuses - статусы, в которых можно изменять сделку
	ModifiableStatuses = []DealStatus{
		DealStatusPending,
	}

	// ConfirmableStatuses - статусы, из которых можно подтвердить сделку
	ConfirmableStatuses = []DealStatus{
		DealStatusPending,
	}
)

// IsActive проверяет, является ли статус активным
func (s DealStatus) IsActive() bool {
	for _, status := range ActiveStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// IsModifiable проверяет, можно ли изменять сделку в данном статусе
func (s DealStatus) IsModifiable() bool {
	for _, status := range ModifiableStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// IsConfirmable проверяет, можно ли подтвердить сделку из данного статуса
func (s DealStatus) IsConfirmable() bool {
	for _, status := range ConfirmableStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// CanTransitionTo проверяет возможен ли переход в новый статус
func (s DealStatus) CanTransitionTo(newStatus DealStatus) bool {
	// Запрещаем переходы из завершенных и отмененных
	if s == DealStatusCompleted || s == DealStatusCancelled {
		return false
	}

	// Определяем разрешенные переходы
	switch s {
	case DealStatusPending:
		return newStatus == DealStatusConfirmed ||
			newStatus == DealStatusCancelled
	case DealStatusConfirmed:
		return newStatus == DealStatusContractPrepared ||
			newStatus == DealStatusCancelled
	case DealStatusContractPrepared:
		return newStatus == DealStatusContractSigned ||
			newStatus == DealStatusCancelled
	case DealStatusContractSigned:
		return newStatus == DealStatusPaymentRequested ||
			newStatus == DealStatusCancelled
	case DealStatusPaymentRequested:
		return newStatus == DealStatusPaid ||
			newStatus == DealStatusCancelled
	case DealStatusPaid:
		return newStatus == DealStatusShipmentRequested ||
			newStatus == DealStatusCancelled
	case DealStatusShipmentRequested:
		return newStatus == DealStatusShipped ||
			newStatus == DealStatusCancelled
	case DealStatusShipped:
		return newStatus == DealStatusCompleted ||
			newStatus == DealStatusCancelled
	default:
		return false
	}
}
