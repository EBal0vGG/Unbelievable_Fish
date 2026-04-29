# ADR-0007: Structured Logging

## Context
Logs were written through the standard `log` package with free-form text
messages. This made failures hard to locate: service name, component, operation,
HTTP status, route, correlation IDs, and source file were not consistently
available.

The system has multiple HTTP services, background schedulers, an outbox relay,
an in-memory event bus, and CLI commands. They need one logging style so
operational failures can be searched and compared across bounded contexts.
Message flow between services must also be visible: when an event is queued,
when the relay tries to publish it, when it succeeds or fails, and which bus
handlers received it.

---

Логи писались через стандартный `log` package в виде свободного текста. Из-за
этого было сложно понять, где именно произошла ошибка: service name, component,
operation, HTTP status, route, correlation IDs и source file не были доступны
единообразно.

В системе есть несколько HTTP-сервисов, background schedulers, outbox relay,
in-memory event bus и CLI commands. Для них нужен единый стиль логирования,
чтобы ошибки можно было искать и сопоставлять между bounded contexts.
Поток сообщений между сервисами тоже должен быть видимым: когда event положен в
outbox, когда relay пытается его опубликовать, когда публикация успешна или
падает, и какие bus handlers получили сообщение.

## Decision
- Use Go `log/slog` for structured logs
- Configure one logger per executable with a `service` field
- Enable source reporting so logs include file and line information
- Default to JSON logs; allow text logs through `LOG_FORMAT=text`
- Control verbosity through `LOG_LEVEL` (`debug`, `info`, `warn`, `error`)
- Add HTTP middleware that logs:
    - method
    - path
    - chi route pattern
    - status
    - response bytes
    - duration
    - remote address
    - user agent
    - correlation and causation IDs
- Log panics at the HTTP boundary with stack traces
- Log mapped HTTP errors in handlers with bounded context, status, code, and metadata
- Log outbox enqueue events with:
    - message ID
    - event ID
    - event type
    - aggregate ID
    - source context
    - correlation and causation IDs where available
- Log outbox relay delivery lifecycle:
    - batch loaded
    - publish attempt
    - publish success
    - publish failure
    - marked published
- Log in-memory event bus delivery:
    - no handlers at debug level
    - successful dispatch with handler count
    - handler failures with handler index and error
- Convert scheduler, integration runtime, outbox relay, migration, admin, and chain runner logs
  to the same structured format

---

- Использовать Go `log/slog` для structured logs
- Настраивать отдельный logger на каждый executable с полем `service`
- Включить source reporting, чтобы в логах были file и line
- По умолчанию писать JSON logs; текстовый формат включается через `LOG_FORMAT=text`
- Управлять уровнем через `LOG_LEVEL` (`debug`, `info`, `warn`, `error`)
- Добавить HTTP middleware, который логирует:
    - method
    - path
    - chi route pattern
    - status
    - response bytes
    - duration
    - remote address
    - user agent
    - correlation and causation IDs
- Логировать panics на HTTP boundary со stack trace
- Логировать mapped HTTP errors в handlers с bounded context, status, code и metadata
- Логировать постановку сообщений в outbox с:
    - message ID
    - event ID
    - event type
    - aggregate ID
    - source context
    - correlation and causation IDs, если они доступны
- Логировать lifecycle доставки через outbox relay:
    - batch loaded
    - publish attempt
    - publish success
    - publish failure
    - marked published
- Логировать доставку через in-memory event bus:
    - отсутствие handlers на debug level
    - успешный dispatch с handler count
    - ошибки handler-ов с handler index и error
- Перевести scheduler, integration runtime, outbox relay, migration, admin и chain runner logs
  на тот же structured формат

## Consequences
- Logs are searchable by stable fields instead of fragile text fragments
- Every service log carries the service name and source location
- HTTP failures show both route-level context and mapped domain/application error information
- Panics no longer disappear into generic server output
- Cross-service message flow is traceable through `outbox_message_enqueued`,
  `outbox_relay_publish_attempt`, `outbox_relay_publish_success`,
  `outbox_relay_failure`, and `event_bus_dispatched`
- Message delivery can be correlated by `message_id`, `event_id`, `event_type`,
  `aggregate_id`, `source_context`, and correlation metadata
- Operators can filter by `service`, `component`, `operation`, `status`, `code`,
  `event_type`, `message_id`, `correlation_id`, or entity IDs

---

- Логи можно искать по стабильным полям, а не по текстовым фрагментам
- Каждый service log содержит service name и source location
- HTTP failures показывают route-level context и mapped domain/application error details
- Panics больше не теряются в generic server output
- Межсервисный message flow можно отследить через `outbox_message_enqueued`,
  `outbox_relay_publish_attempt`, `outbox_relay_publish_success`,
  `outbox_relay_failure` и `event_bus_dispatched`
- Доставку сообщений можно связывать по `message_id`, `event_id`, `event_type`,
  `aggregate_id`, `source_context` и correlation metadata
- Операторы могут фильтровать по `service`, `component`, `operation`, `status`, `code`,
  `event_type`, `message_id`, `correlation_id` или entity IDs
