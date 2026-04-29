# ADR-0006: Chi HTTP Transport Router

## Context
The HTTP transport layer previously used custom route dispatching based on
manual `net/http` handlers, `ServeMux`, method checks inside handlers, and
path parsing with string operations.

As the number of endpoints grew, this made the transport boundary harder to
read and maintain. Routing rules were mixed with request handling logic, and
path parameters were extracted in different ways across bounded contexts.

---

HTTP transport layer раньше использовал самописную маршрутизацию на базе
`net/http`, `ServeMux`, проверок метода внутри handlers и ручного разбора path
через строки.

По мере роста количества endpoints это ухудшало читаемость transport boundary:
правила маршрутизации смешивались с обработкой запроса, а path-параметры
извлекались по-разному в разных bounded contexts.

## Decision
- Use `github.com/go-chi/chi/v5` as the HTTP router for service transport
- Replace custom `Router` structs and `ServeHTTP` route dispatching with `chi.NewRouter()`
- Define HTTP method and path matching in router tables, not inside handlers
- Read route parameters through `chi.URLParam`
- Remove manual route parsing based on `URL.Path`, `strings.Split`, prefixes, and suffixes
- Keep `net/http` request/response types as the transport contract, because chi is built on
  the standard Go HTTP interfaces
- Keep handlers focused on:
    - request body parsing
    - auth/context metadata extraction
    - use case invocation
    - response and error mapping
- Apply the same routing approach to Identity, Trading, Deals, and Catalog HTTP entrypoints

---

- Использовать `github.com/go-chi/chi/v5` как HTTP router для transport layer
- Заменить самописные `Router` structs и ручной `ServeHTTP` dispatch на `chi.NewRouter()`
- Описывать HTTP method и path matching в таблице маршрутов, а не внутри handlers
- Читать route params через `chi.URLParam`
- Убрать ручной разбор маршрутов через `URL.Path`, `strings.Split`, prefix/suffix checks
- Оставить `net/http` request/response types как транспортный контракт, потому что chi
  работает поверх стандартных Go HTTP interfaces
- Ограничить ответственность handlers:
    - разбор body
    - извлечение auth/context metadata
    - вызов use case
    - формирование response и error mapping
- Применить единый подход к HTTP entrypoints в Identity, Trading, Deals и Catalog

## Consequences
- Routing is explicit and centralized in router definitions
- Handlers no longer duplicate router responsibilities
- Path parameter extraction is consistent across bounded contexts
- Method handling is delegated to chi, reducing boilerplate in handlers
- Existing middleware and server startup remain compatible through `http.Handler`
- The transport layer remains framework-light: chi is used only at the HTTP adapter boundary

---

- Маршрутизация стала явной и централизованной в router definitions
- Handlers больше не дублируют ответственность router-а
- Path-параметры извлекаются единообразно во всех bounded contexts
- Обработка HTTP methods делегирована chi, boilerplate в handlers уменьшен
- Существующие middleware и запуск сервера остаются совместимыми через `http.Handler`
- Transport layer остаётся лёгким: chi используется только на границе HTTP adapter
