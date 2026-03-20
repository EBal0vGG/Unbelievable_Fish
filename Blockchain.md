# 1. Самое главное — блокчейн это НЕ backend

Ваш backend:

```
Auction
Catalog
Deal
DB
EventBus
```

Блокчейн — это как **внешняя база данных с API**, только очень медленная и с особыми правилами.

Пример аналогии:

| Система    | Аналог                    |
| ---------- | ------------------------- |
| PostgreSQL | обычная БД                |
| Redis      | кэш                       |
| Kafka      | EventBus                  |
| Blockchain | внешняя БД с транзакциями |

То есть вы будете делать:

```
Deal -> Blockchain API -> Blockchain network
```

как сейчас:

```
Deal -> Postgres
Deal -> EventBus
Deal -> HTTP API
```

---

# 2. Что значит "сделать блокчейн"

Это НЕ значит писать блокчейн.

Это значит:

✅ подключиться к готовому блокчейну
✅ написать контракт
✅ вызывать контракт из backend

---

# 3. Из чего состоит блокчейн-интеграция

Всего 3 части:

```
1. Smart contract (код в блокчейне)
2. Node / RPC (доступ к блокчейну)
3. Backend service (ваш Go сервис)
```

Схема:

```
Deal service
   |
EventBus
   |
Blockchain service (Go)
   |
RPC
   |
Blockchain node
   |
Smart contract
```

---

# 4. На каком языке писать

Вот важный ответ.

| Что                | Язык          |
| ------------------ | ------------- |
| backend            | Go            |
| blockchain service | Go            |
| smart contracts    | Solidity      |
| node               | готовый       |
| RPC client         | Go библиотека |

То есть:

✅ ваш backend остаётся на Go
✅ новый сервис на Go
❗ только контракт на Solidity

---

# 5. Что такое smart contract (простыми словами)

Это как stored procedure в БД.

Вы пишете:

```
function createDeal()
function pay()
function confirm()
```

И backend их вызывает.

Пример:

```
contract DealContract {

    function createDeal(uint id) public {}

    function confirmDeal(uint id) public {}

}
```

Это пишется на Solidity.

---

# 6. Где будет новый домен

Сейчас:

```
Auction
Catalog
Deal
```

Добавится:

```
Blockchain
```

или

```
Ledger
Chain
Web3
Wallet
```

лучше:

```
blockchain-service
```

---

# 7. Что делает blockchain-service

Он делает ровно одно:

```
получает event -> отправляет tx -> ждёт -> шлёт event
```

То есть адаптер.

Как адаптер к Kafka.

---

# 8. Как это выглядит логически

Сейчас:

```
DealCreated
 -> save DB
 -> emit event
```

Будет:

```
DealCreated
 -> emit event

BlockchainService
 -> send tx
 -> wait confirm
 -> emit TxConfirmed

DealService
 -> update deal
```

---

# 9. Как это выглядит в коде (Go)

## 9.1 Blockchain service

```
/blockchain
   service.go
   client.go
   contract.go
   listener.go
```

---

## 9.2 client.go

Подключение к блокчейну

```go
client, err := ethclient.Dial(RPC_URL)
```

RPC_URL:

```
https://polygon-rpc.com
https://infura.io
```

---

## 9.3 contract.go

Вызываем контракт

```go
func (c *Contract) CreateDeal(id int) error {

    tx, err := c.instance.CreateDeal(
        c.auth,
        big.NewInt(int64(id)),
    )

    return err
}
```

Это обычный Go код.

---

## 9.4 service.go

Слушаем EventBus

```go
func (s *Service) HandleDealCreated(e DealCreated) {

    err := s.contract.CreateDeal(e.ID)

    if err != nil {
        return
    }

}
```

Всё.

Это обычный сервис.

---

# 10. Как выглядит полный flow

```
Auction -> event

Deal -> event DealCreated

BlockchainService
 -> CreateDeal()
 -> send tx

Blockchain
 -> mined

BlockchainService listener
 -> TxConfirmed

EventBus
 -> DealConfirmed

DealService
 -> update DB
```

---

# 11. Где писать Solidity

Отдельная папка:

```
/contracts
   Deal.sol
   Token.sol
   Escrow.sol
```

Это НЕ Go.

Это отдельный код.

---

# 12. Что будет в репозитории

```
/auction
/catalog
/deal
/blockchain
/contracts
/common
/eventbus
```

---

# 13. Что реально вам нужно сделать первым

Вот правильный порядок.

### 1. Понять зачем блокчейн

Без этого нельзя писать.

Вопрос:

* деньги?
* токены?
* подтверждение сделки?
* NFT?
* escrow?

Без этого нельзя.

---

### 2. Выбрать сеть

Для старта:

```
Polygon testnet
```

---

### 3. Подключиться к RPC из Go

Это первый код.

---

### 4. Написать самый простой контракт

```
createDeal
confirmDeal
```

---

### 5. Вызвать контракт из Go

---

### 6. Сделать blockchain-service

---

### 7. Подключить EventBus

---

# 1. В блокчейне нет "реальных денег"

В блокчейне нет:

* евро
* долларов
* банковских счетов

В блокчейне есть только:

```
токены
криптовалюта
```

Примеры:

* ETH
* USDT
* USDC
* BTC (в своей сети)
* кастомные токены

Поэтому всегда схема такая:

```
Fiat (EUR/USD)
   ↓
Payment provider / bank
   ↓
Crypto / Token
   ↓
Blockchain
```

---

# 2. Где происходит настоящая оплата

Есть 3 варианта архитектуры.

---

## Вариант 1 — через криптовалюту (самый простой)

Пользователь платит криптой.

Flow:

```
User wallet -> Blockchain -> Contract -> Deal confirmed
```

Пример:

```
MetaMask -> send USDT -> contract -> deal done
```

В этом варианте backend вообще не работает с деньгами.

Backend только проверяет tx.

Это самый простой вариант.

---

## Вариант 2 — через платёжный провайдер (реальные деньги)

Пользователь платит картой.

Flow:

```
User -> Stripe / PayPal / Adyen
      -> backend
      -> mint token
      -> blockchain
```

Схема:

```
Card payment
   ↓
Payment provider
   ↓
Backend
   ↓
create tokens
   ↓
Blockchain
```

То есть:

реальные деньги → токены → блокчейн

Так делают почти все биржи.

---

## Вариант 3 — custodial биржа (как Binance)

Самая частая схема.

```
User -> bank / card -> exchange balance -> blockchain
```

Flow:

```
User pays EUR
 -> backend balance +100
 -> user buys lot
 -> backend sends blockchain tx
```

То есть:

```
DB balance
+
Blockchain settlement
```

---

# 3. Где в архитектуре появляется оплата

Добавляется новый домен:

```
Payment
Wallet
Balance
Ledger
```

Теперь архитектура:

```
Auction
Catalog
Deal
Wallet
Payment
Blockchain
EventBus
DB
```

---

# 4. Как это выглядит логически

### Пользователь пополняет баланс

```
POST /deposit
```

Flow:

```
PaymentService
 -> Stripe
 -> success
 -> Balance +100
 -> event DepositCompleted
```

---

### Пользователь покупает

```
DealCreated
```

Flow:

```
DealService
 -> check balance
 -> reserve money
 -> event DealCreated
```

---

### Blockchain подтверждает

```
BlockchainService
 -> send tx
 -> confirmed
 -> event TxConfirmed
```

---

### Деньги списываются

```
BalanceService
 -> minus
 -> plus seller
```

---

# 5. Где реально лежат деньги

Есть 2 варианта.

---

## Custodial (чаще всего)

Деньги лежат у вас.

```
Bank account
Stripe
Crypto wallet
```

В блокчейне лежит только подтверждение.

Это как у Binance.

---

## Non-custodial

Деньги у пользователя.

```
User wallet -> contract
```

Backend не хранит деньги.

Это сложнее.

---

# 6. Как обычно делают блокчейн-биржу

Реальная схема:

```
Users
Balances (DB)
Payment provider
Blockchain
Smart contract
```

Flow:

```
Deposit -> DB
Trade -> DB
Settlement -> Blockchain
```

Это называется:

```
off-chain trading
on-chain settlement
```

---

# 7. Как это будет у вас

Сейчас:

```
Auction
Catalog
Deal
```

Будет:

```
Auction
Catalog
Deal
Wallet
Payment
Blockchain
Balance
```

---

# 8. Где в коде появляется оплата

Добавится сервис:

```
payment-service
```

Пример:

```go
type PaymentService struct {}

func (s *PaymentService) Deposit(userID int, amount int) {}

func (s *PaymentService) Withdraw() {}
```

---

Добавится баланс:

```go
type Balance struct {
    UserID
    Amount
}
```

---

Добавится blockchain:

```go
type BlockchainService struct {}

func (s *BlockchainService) ConfirmDeal() {}
```

---

# 9. Самый правильный вопрос сейчас

Вам нужно решить:

> Мы хотим чтобы пользователи платили чем?

Варианты:

1. Только криптой
2. Картой + криптой
3. Только картой
4. Внутренним балансом
5. NFT / токены
6. Escrow сделки

Без этого нельзя проектировать.

