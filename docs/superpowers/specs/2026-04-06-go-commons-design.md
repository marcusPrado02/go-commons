# go-commons — Design Spec

**Data:** 2026-04-06  
**Autor:** Marcus Prado Silva  
**Status:** Aprovado

---

## 1. Visão Geral

### Objetivo

`go-commons` é uma biblioteca de infraestrutura reutilizável em Go, equivalente idiomática do `commons-platform` Java. Fornece blocos de construção para aplicações que seguem Arquitetura Hexagonal (Ports & Adapters) e Domain-Driven Design (DDD).

### Problema que resolve

Aplicações Go de médio/grande porte replicam os mesmos padrões de infraestrutura em cada projeto: abstrações de envio de email, armazenamento de arquivos, resiliência, observabilidade, outbox pattern. `go-commons` centraliza essas abstrações com qualidade de produção, permitindo que times foquem na lógica de domínio.

### Filosofia

- **Zero dependências externas no core** — `kernel` e `ports` dependem apenas da stdlib Go. `app` e `testkit` podem ter dependências mínimas e bem-curadas (`gobreaker`, `robfig/cron`, `testify`).
- **Adaptadores isolados** — cada adaptador externo é um módulo Go independente; consumidores pagam apenas pelo que usam.
- **Idiomático Go sobre tradução literal Java** — `(T, error)` nas interfaces públicas, composição via embedding, interfaces pequenas, context propagation.
- **Híbrido Result[T]** — `Result[T]` existe como utilitário para pipelines funcionais; ports usam `(T, error)` com `DomainError` rico.

---

## 2. Arquitetura

### Diagrama de camadas

```
┌─────────────────────────────────────────────┐
│                  caller app                 │
└──────────────────┬──────────────────────────┘
                   │ usa
┌──────────────────▼──────────────────────────┐
│              ports/*                        │  ← interfaces puras
│   email · files · persistence · template   │
│   cache · queue · sms · push · secrets     │
│   excel · compression · observability      │
└──────────────────┬──────────────────────────┘
                   │ implementado por
┌──────────────────▼──────────────────────────┐
│             adapters/*  (submódulos)        │
│  sendgrid · ses · smtp · s3 · gcs · stripe │
│  elasticsearch · opensearch · otel · twilio │
│  inmemory                                   │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│               app/*                         │  ← cross-cutting concerns
│      resilience · observability             │
│      outbox · scheduler                     │
└──────────────────┬──────────────────────────┘
                   │ fundamentado por
┌──────────────────▼──────────────────────────┐
│              kernel/*                       │  ← zero deps externas
│       errors · result · ddd                │
└─────────────────────────────────────────────┘
```

### Regras de dependência entre camadas

| Camada | Pode importar | Não pode importar |
|---|---|---|
| `kernel` | stdlib apenas | qualquer coisa |
| `ports` | `kernel` + stdlib | `app`, `adapters` |
| `app` | `kernel` + `ports` + stdlib + deps curadas¹ | `adapters` |
| `testkit` | `kernel` + `ports` + `testify` | `adapters` |
| `adapters/*` | módulo raiz + SDK próprio | outros adapters |

¹ Deps curadas em `app/`: `github.com/sony/gobreaker` (resilience), `github.com/robfig/cron/v3` (scheduler).

### Estrutura de módulos

```
go-commons/                          ← módulo raiz: github.com/marcusPrado02/go-commons
├── go.mod                           ← Go 1.22
├── go.work                          ← workspace unifica raiz + submódulos
├── Makefile
├── .golangci.yml
│
├── kernel/
│   ├── errors/
│   ├── result/
│   └── ddd/
│
├── ports/
│   ├── email/
│   ├── files/
│   ├── persistence/
│   ├── template/
│   ├── cache/
│   ├── queue/
│   ├── sms/
│   ├── push/
│   ├── secrets/
│   ├── excel/
│   ├── compression/
│   └── observability/
│
├── app/
│   ├── resilience/
│   ├── observability/
│   ├── outbox/
│   └── scheduler/
│
├── testkit/
│   ├── assert/
│   └── contracts/
│
└── adapters/                        ← cada subdir é um submódulo Go independente
    ├── email/
    │   ├── sendgrid/go.mod          ← github.com/marcusPrado02/go-commons/adapters/email/sendgrid
    │   ├── ses/go.mod
    │   └── smtp/go.mod
    ├── files/
    │   ├── s3/go.mod
    │   └── gcs/go.mod
    ├── payment/
    │   └── stripe/go.mod
    ├── search/
    │   ├── elasticsearch/go.mod
    │   └── opensearch/go.mod
    ├── persistence/
    │   └── inmemory/go.mod
    └── tracing/
        └── otel/go.mod
```

---

## 3. Módulos

### 3.1 `kernel/errors`

Tipos de erro de domínio com semântica rica. Nenhuma dependência externa.

```go
// ErrorCode é um type alias de string com validação explícita
type ErrorCode string
func NewErrorCode(code string) (ErrorCode, error)  // valida formato não-vazio

type ErrorCategory string
const (
    CategoryValidation   ErrorCategory = "VALIDATION"
    CategoryBusiness     ErrorCategory = "BUSINESS"
    CategoryTechnical    ErrorCategory = "TECHNICAL"
    CategoryNotFound     ErrorCategory = "NOT_FOUND"
    CategoryUnauthorized ErrorCategory = "UNAUTHORIZED"
)

type Severity string
const (
    SeverityInfo     Severity = "INFO"
    SeverityWarning  Severity = "WARNING"
    SeverityError    Severity = "ERROR"
    SeverityCritical Severity = "CRITICAL"
)

// Problem é imutável — construção via builders seguros
type Problem struct {
    Code     ErrorCode
    Category ErrorCategory
    Severity Severity
    Message  string
    Details  map[string]any  // cópia defensiva na construção
    Cause    error            // preserva chain completa para logging/tracing
}

func NewProblem(code ErrorCode, category ErrorCategory, severity Severity, message string) Problem

// Builders — retornam nova cópia, nunca modificam o receptor
func (p Problem) WithDetail(key string, value any) Problem
func (p Problem) WithDetails(details map[string]any) Problem
func (p Problem) WithCause(err error) Problem

// Problem implementa error e suporta errors.Is/As via Unwrap
func (p Problem) Error() string
func (p Problem) Unwrap() error  // delega para Cause

// DomainError — interface para erros retornados pelos ports
// Adapters envolvem erros SDK nesta interface antes de retornar
type DomainError interface {
    error
    Code() ErrorCode
    Category() ErrorCategory
    Severity() Severity
    Details() map[string]any
    Unwrap() error
}

// Erros sentinela pré-definidos
var (
    ErrNotFound     = NewProblem("NOT_FOUND",     CategoryNotFound,     SeverityError,   "resource not found")
    ErrUnauthorized = NewProblem("UNAUTHORIZED",  CategoryUnauthorized, SeverityWarning, "unauthorized access")
    ErrValidation   = NewProblem("VALIDATION",    CategoryValidation,   SeverityWarning, "validation failed")
    ErrTechnical    = NewProblem("TECHNICAL",     CategoryTechnical,    SeverityError,   "technical error")
)
```

### 3.2 `kernel/result`

Tipo utilitário para pipelines funcionais. Não é mandatório — ports usam `(T, error)`.

```go
type Result[T any] struct{ /* campos não exportados */ }

func Ok[T any](value T) Result[T]
func Fail[T any](problem errors.Problem) Result[T]
func FromError[T any](err error) Result[T]  // bridge idiomático

func (r Result[T]) IsOk() bool
func (r Result[T]) IsFail() bool
func (r Result[T]) Value() T           // panic se IsFail() — só usar com garantia prévia
func (r Result[T]) ValueOrZero() T
func (r Result[T]) Problem() errors.Problem  // panic se IsOk()
func (r Result[T]) Unwrap() (T, error)

// Funções standalone — Go não suporta métodos genéricos adicionais
func Map[T, U any](r Result[T], f func(T) U) Result[U]
func FlatMap[T, U any](r Result[T], f func(T) Result[U]) Result[U]
```

### 3.3 `kernel/ddd`

Primitivos DDD embeddable. Sem herança — composição via embedding.

```go
type DomainEvent interface {
    OccurredAt() time.Time
    EventType() string
}

// AggregateRoot é embeddable: type Order struct { ddd.AggregateRoot[OrderID]; ... }
type AggregateRoot[ID any] struct {
    id     ID
    events []DomainEvent
}

func NewAggregateRoot[ID any](id ID) AggregateRoot[ID]
func (a *AggregateRoot[ID]) ID() ID
func (a *AggregateRoot[ID]) RegisterEvent(event DomainEvent)

// PullDomainEvents retorna cópia e limpa — sem efeitos colaterais externos
func (a *AggregateRoot[ID]) PullDomainEvents() []DomainEvent
```

### 3.4 `ports/email`

```go
type EmailPort interface {
    Send(ctx context.Context, email Email) (EmailReceipt, error)
    SendWithTemplate(ctx context.Context, req TemplateEmailRequest) (EmailReceipt, error)
    Ping(ctx context.Context) error
}

type EmailAddress struct{ Value string }
func NewEmailAddress(value string) (EmailAddress, error)  // valida RFC 5322

type Email struct {
    From    EmailAddress
    To      []EmailAddress
    CC      []EmailAddress
    BCC     []EmailAddress
    Subject string
    HTML    string
    Text    string
    ReplyTo *EmailAddress
}

// Validate garante integridade antes de chegar no adapter:
// - len(To) >= 1
// - HTML != "" || Text != ""
// - From válido
func (e Email) Validate() error

type EmailReceipt struct{ MessageID string }

type TemplateEmailRequest struct {
    From         EmailAddress
    To           []EmailAddress
    TemplateName string
    Variables    map[string]any
}
```

### 3.5 `ports/files`

```go
type FileStorePort interface {
    Upload(ctx context.Context, id FileID, content io.Reader, opts ...UploadOption) (UploadResult, error)
    Download(ctx context.Context, id FileID) (FileObject, error)
    Delete(ctx context.Context, id FileID) error
    DeleteAll(ctx context.Context, ids []FileID) (DeleteResult, error)
    Exists(ctx context.Context, id FileID) (bool, error)
    GetMetadata(ctx context.Context, id FileID) (FileMetadata, error)
    List(ctx context.Context, bucket, prefix string, opts ...ListOption) (ListResult, error)
    GeneratePresignedURL(ctx context.Context, id FileID, op PresignedOperation, ttl time.Duration, opts ...PresignOption) (*url.URL, error)
    Copy(ctx context.Context, src, dst FileID) error
}

type FileID struct{ Bucket, Key string }

// FileObject — caller é responsável por fechar Content
type FileObject struct {
    Content  io.ReadCloser
    Metadata FileMetadata
}

type PresignedOperation string
const (
    PresignGet    PresignedOperation = "GET"
    PresignPut    PresignedOperation = "PUT"
    PresignDelete PresignedOperation = "DELETE"
)

type StorageClass string
const (
    StorageClassStandard StorageClass = "STANDARD"
    StorageClassGlacier  StorageClass = "GLACIER"
    StorageClassIA       StorageClass = "STANDARD_IA"
)

// Options: WithContentType, WithStorageClass, WithMetadata, WithMaxKeys
// List: prefix é path-like ("uploads/2026/") — sem leading slash
```

### 3.6 `ports/persistence`

```go
type Repository[E any, ID any] interface {
    // Save é upsert — pode modificar entity (ID gerado, timestamps atualizados)
    Save(ctx context.Context, entity E) (E, error)
    // FindByID: (entity, true, nil) = encontrado | (zero, false, nil) = não encontrado | (zero, false, err) = erro técnico
    FindByID(ctx context.Context, id ID) (E, bool, error)
    DeleteByID(ctx context.Context, id ID) error
    Delete(ctx context.Context, entity E) error
}

type PageableRepository[E any, ID any] interface {
    Repository[E, ID]
    FindAll(ctx context.Context, req PageRequest, spec Specification[E]) (PageResult[E], error)
    Search(ctx context.Context, req PageRequest, spec Specification[E], sort Sort) (PageResult[E], error)
}

// Specification como interface — extensível para SQL, Elasticsearch, etc.
type Specification[E any] interface {
    ToPredicate() func(E) bool
}

// FuncSpec — adapter de conveniência para uso direto
func Spec[E any](fn func(E) bool) Specification[E]

type Sort struct{ Field string; Descending bool }
type PageRequest struct{ Page, Size int }
type PageResult[E any] struct {
    Content       []E
    TotalElements int
    TotalPages    int
    Page, Size    int
}
```

### 3.7 `ports/template`

```go
type TemplatePort interface {
    Render(ctx context.Context, name string, data map[string]any) (TemplateResult, error)
    Exists(ctx context.Context, name string) (bool, error)
}

const (
    ContentTypeHTML = "text/html"
    ContentTypeText = "text/plain"
    ContentTypeXML  = "application/xml"
)

type TemplateResult struct {
    TemplateName string
    Content      string
    ContentType  string  // usar constants acima
    Charset      string
}

func HTMLResult(name, content string) TemplateResult
func TextResult(name, content string) TemplateResult
func XMLResult(name, content string) TemplateResult
func (t TemplateResult) Bytes() []byte
func (t TemplateResult) IsEmpty() bool
```

### 3.8 `ports/observability`

```go
// Field para structured logging/metrics — reutilizado em toda a API
type Field struct{ Key string; Value any }
func F(key string, value any) Field

// Helpers de conveniência — reduzem boilerplate
func Err(err error) Field       { return F("error", err) }
func RequestID(id string) Field { return F("request.id", id) }
func UserID(id string) Field    { return F("user.id", id) }

type Logger interface {
    Info(ctx context.Context, msg string, fields ...Field)
    Warn(ctx context.Context, msg string, fields ...Field)
    Error(ctx context.Context, msg string, fields ...Field)  // erro sempre via Err(err)
    Debug(ctx context.Context, msg string, fields ...Field)
}

type Counter interface{ Inc(); Add(float64) }
type Observer interface{ Observe(float64) }

type Metrics interface {
    Counter(name string, labels ...Field) Counter      // labels como Field — ordem não importa
    Histogram(name string, labels ...Field) Observer
}

type Span interface {
    End()
    RecordError(err error)                  // alinhado com OpenTelemetry API
    SetAttribute(key string, value any)
}

type Tracer interface {
    StartSpan(ctx context.Context, name string) (context.Context, Span)
}
```

#### Convenções de naming

**Métricas** — padrão `<domínio>.<recurso>.<operação>.<tipo>`:
```
app.requests.total          app.requests.duration_ms
infra.s3.uploads.total      infra.s3.downloads.total
infra.email.sent.total      infra.email.failed.total
infra.cache.hits.total      infra.cache.misses.total
outbox.processed.total      outbox.failed.total
outbox.latency_ms
```

**Atributos de tracing/log** — padrão `<namespace>.<atributo>`:
```
request.id    user.id       file.key      file.bucket
email.to      queue.topic   error.code    error.category
```

Convenções documentadas como constantes em `ports/observability/conventions.go`.

### 3.9 Ports menores

| Port | Métodos principais |
|---|---|
| `ports/cache` | `Get`, `Set(ctx, key, value, ttl)`, `Delete`, `Exists` |
| `ports/queue` | `Publish(ctx, topic string, msg Message)`, `Subscribe(ctx, topic string, handler Handler)` |
| `ports/sms` | `Send(ctx, to, body string) (SMSReceipt, error)`, `Ping(ctx) error` |
| `ports/push` | `Send(ctx, PushNotification) (PushReceipt, error)` |
| `ports/secrets` | `Get(ctx, key string) (string, error)`, `GetJSON(ctx, key string, dest any) error` |
| `ports/excel` | `Generate(ctx, ExcelRequest) (io.Reader, error)` |
| `ports/compression` | `Compress(ctx, io.Reader, Format) (io.Reader, error)`, `Decompress(ctx, io.Reader, Format) (io.Reader, error)` |

---

## 4. Padrões

### 4.1 Result[T] vs (T, error)

- **Interfaces de port:** sempre `(T, error)`. Erros implementam `DomainError` com código, categoria, severidade.
- **`Result[T]`:** disponível como utilitário para pipelines funcionais internos onde encadeamento é mais expressivo.
- **Bridge:** `result.FromError[T](err)` converte `(T, error)` → `Result[T]` quando necessário.

```go
// Port (idiomático Go)
receipt, err := emailPort.Send(ctx, email)
if err != nil { /* err implementa DomainError */ }

// Pipeline funcional com Result[T]
r := result.Ok(user)
r = result.Map(r, enrichUser)
value, err := r.Unwrap()
```

### 4.2 DomainError

Erros retornados pelos adapters devem implementar `DomainError`:
- Erros SDK externos são mapeados para `Problem` com categoria correta (ex: `404` → `CategoryNotFound`)
- `Unwrap()` preserva o erro original para diagnóstico
- Compatível com `errors.Is/As` para testes precisos

### 4.3 Resilience (backoff com jitter)

```go
type ResiliencePolicySet struct {
    RetryAttempts   int
    RetryDelay      time.Duration  // delay inicial
    RetryMaxDelay   time.Duration  // teto do backoff exponencial
    TimeoutDuration time.Duration
    CircuitBreaker  *CircuitBreakerConfig
}
```

**Estratégia de retry:** backoff exponencial com full jitter — `delay = random(0, min(cap, base * 2^attempt))`. Evita thundering herd em falhas simultâneas. Context cancelado interrompe o retry imediatamente.

**Circuit breaker:** via `github.com/sony/gobreaker`. Estados: Closed → Open (após `FailureThreshold`) → Half-Open (após `Timeout`) → Closed (após `MaxRequests` bem-sucedidos).

### 4.4 Transactional Outbox

```go
type OutboxMessage struct {
    ID          string     // chave idempotente — UUID gerado pelo caller
    AggregateID string
    EventType   string
    Payload     []byte     // JSON canônico — serialização é responsabilidade do caller
    CreatedAt   time.Time
    ProcessedAt *time.Time
    Attempts    int
}
```

**Idempotência:**
- `OutboxMessage.ID` é a chave idempotente global
- `FetchPending` filtra por `ProcessedAt IS NULL`
- `MarkProcessed` é idempotente — chamadas duplicadas não são erro
- Integração com SQS FIFO usa `ID` como `MessageDeduplicationId`

**Lifecycle do publisher:**
```go
// Start é NÃO bloqueante — lança goroutine interna
func (p *OutboxPublisher) Start(ctx context.Context) error
// Stop aguarda ciclo atual terminar — graceful shutdown
func (p *OutboxPublisher) Stop(ctx context.Context) error
```

**Configurações de polling:**
- `WithPollingInterval(d time.Duration)` — padrão: 5s
- `WithBatchSize(n int)` — padrão: 100 mensagens por ciclo
- `WithConcurrency(n int)` — padrão: 1 (processamento sequencial, evita reordenação)

**Métricas emitidas:**
```
outbox.processed.total
outbox.failed.total
outbox.latency_ms
```

### 4.5 Observability

`app/observability` fornece implementações concretas das interfaces de `ports/observability`:
- `HealthChecks` — agrega checks por tipo (Liveness/Readiness), reporta status agregado (UP se todos UP, DEGRADED se algum DOWN mas não crítico, DOWN se crítico DOWN)
- `DefaultSanitizer` — redacta campos sensíveis: `password`, `token`, `secret`, `cpf`, `credit_card`, `authorization` + campos adicionais configuráveis

---

## 5. Adaptadores

### Profundidade de implementação

| Adaptador | Nível | Lib |
|---|---|---|
| `adapters/persistence/inmemory` | Completo | stdlib |
| `adapters/email/sendgrid` | Completo | `github.com/sendgrid/sendgrid-go` |
| `adapters/files/s3` | Completo | `github.com/aws/aws-sdk-go-v2` |
| `adapters/email/ses` | Essencial | `github.com/aws/aws-sdk-go-v2` |
| `adapters/files/gcs` | Essencial | `cloud.google.com/go/storage` |
| `adapters/email/smtp` | Essencial | `net/smtp` (stdlib) |
| `adapters/payment/stripe` | Essencial | `github.com/stripe/stripe-go/v76` |
| `adapters/search/elasticsearch` | Essencial | `github.com/elastic/go-elasticsearch/v8` |
| `adapters/search/opensearch` | Essencial | `github.com/opensearch-project/opensearch-go/v2` |
| `adapters/tracing/otel` | Essencial | `go.opentelemetry.io/otel` |
| `adapters/sms/twilio` | Essencial | `github.com/twilio/twilio-go` |

### `adapters/persistence/inmemory`

```go
type InMemoryRepository[E any, ID comparable] struct {
    storage     map[ID]E
    idExtractor func(E) ID  // type-safe, sem reflection
    mu          sync.RWMutex
}

func NewInMemoryRepository[E any, ID comparable](idExtractor func(E) ID) *InMemoryRepository[E, ID]
```

- `Save`: upsert thread-safe com write lock
- `FindAll`: aplica `Specification.ToPredicate()` + paginação em memória
- `Search`: spec + sort por campo (usando função de comparação injetável via option)

### `adapters/files/s3`

- `Upload`: `PutObject` com multipart automático para arquivos > 5MB via `manager.Uploader`
- `Download`: `GetObject` — retorna `FileObject{Content: resp.Body}` (caller fecha)
- `GeneratePresignedURL`: `PresignGetObject` / `PresignPutObject` / `PresignDeleteObject`

### `adapters/email/sendgrid`

- `Send`: `POST /v3/mail/send` com retry em 429/5xx
- `Ping`: `GET /v3/mail/settings` com timeout de 5s
- `baseURL` configurável para testes com servidor mock

---

## 6. Testkit

### `testkit/assert`

```go
// Constraint estrutural — qualquer struct com PullDomainEvents() satisfaz
func AssertAggregate[T interface{ PullDomainEvents() []ddd.DomainEvent }](
    t testing.TB, actual T,
) *AggregateAssertion[T]

// API fluente
a.HasDomainEvents(2).
  HasEventOfType("OrderPlaced").
  FirstEventSatisfies(func(e ddd.DomainEvent) bool { ... })
```

### `testkit/contracts`

```go
// Suite reutilizável — zero duplicação entre implementações de Repository
type RepositoryContract[E any, ID comparable] struct {
    suite.Suite
    Repo         persistence.Repository[E, ID]
    NewEntity    func() E
    ExtractID    func(E) ID
    MutateEntity func(E) E
}

// Testes providos automaticamente:
// TestSave_insertsNewEntity, TestSave_updatesExistingEntity,
// TestFindByID_found, TestFindByID_notFound,
// TestDeleteByID_removes, TestDeleteByID_notFound_noError
//
// TODO: cenários de concorrência (save/delete simultâneos) — próxima versão
```

---

## 7. Como usar

### Exemplo 1 — Envio de email com resiliência

```go
import (
    "github.com/marcusPrado02/go-commons/ports/email"
    "github.com/marcusPrado02/go-commons/app/resilience"
    sendgrid "github.com/marcusPrado02/go-commons/adapters/email/sendgrid"
)

// Setup
from, _ := email.NewEmailAddress("noreply@acme.com")
client, _ := sendgrid.New(os.Getenv("SENDGRID_KEY"), from)

exec := resilience.NewExecutor()
policies := resilience.ResiliencePolicySet{
    RetryAttempts:   3,
    RetryDelay:      500 * time.Millisecond,
    RetryMaxDelay:   5 * time.Second,
    TimeoutDuration: 10 * time.Second,
}

// Uso com Supply[T]
receipt, err := resilience.Supply(ctx, exec, "send-welcome-email", policies,
    func(ctx context.Context) (email.EmailReceipt, error) {
        to, _ := email.NewEmailAddress("user@example.com")
        return client.Send(ctx, email.Email{
            From:    from,
            To:      []email.EmailAddress{to},
            Subject: "Bem-vindo!",
            HTML:    "<h1>Olá!</h1>",
        })
    },
)
```

### Exemplo 2 — Aggregate com eventos de domínio

```go
import "github.com/marcusPrado02/go-commons/kernel/ddd"

type Order struct {
    ddd.AggregateRoot[OrderID]
    status OrderStatus
}

func PlaceOrder(id OrderID, items []Item) (*Order, error) {
    o := &Order{}
    o.AggregateRoot = ddd.NewAggregateRoot(id)
    o.status = StatusPending
    o.RegisterEvent(OrderPlaced{OrderID: id, Items: items, OccurredAt: time.Now()})
    return o, nil
}

// Teste
func TestPlaceOrder(t *testing.T) {
    order, _ := PlaceOrder(OrderID("123"), items)
    testkit.AssertAggregate(t, order).
        HasDomainEvents(1).
        HasEventOfType("OrderPlaced")
}
```

### Exemplo 3 — Repositório in-memory para testes

```go
import (
    "github.com/marcusPrado02/go-commons/adapters/persistence/inmemory"
    "github.com/marcusPrado02/go-commons/ports/persistence"
)

repo := inmemory.NewInMemoryRepository[User, UserID](func(u User) UserID { return u.ID })

saved, err := repo.Save(ctx, user)
found, ok, err := repo.FindByID(ctx, user.ID)

// Com specification
activeUsers, err := repo.FindAll(ctx,
    persistence.PageRequest{Page: 0, Size: 20},
    persistence.Spec[User](func(u User) bool { return u.Active }),
)
```

---

## 8. Roadmap

### Próximos adaptadores
- `adapters/cache/redis` — via `go-redis/v9`
- `adapters/queue/sqs` — via `aws-sdk-go-v2`
- `adapters/queue/pubsub` — via `cloud.google.com/go/pubsub`
- `adapters/secrets/awssm` — AWS Secrets Manager
- `adapters/metrics/prometheus` — implementa `ports/observability.Metrics`
- `adapters/logging/slog` — implementa `ports/observability.Logger` via `log/slog`

### Melhorias futuras
- `Specification.ToSQL() string` para repositórios SQL (PostgreSQL via `pgx`)
- `Specification.ToElasticQuery()` para Elasticsearch
- Cenários de concorrência no `RepositoryContract`
- `OutboxPublisher` com modo adaptativo de polling (backoff quando fila vazia)
- `app/saga` — orquestração de sagas com compensação
